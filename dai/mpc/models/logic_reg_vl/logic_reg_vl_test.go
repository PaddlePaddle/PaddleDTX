// Copyright (c) 2021 PaddlePaddle Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logic_reg_vl

import (
	"errors"
	"io/ioutil"
	"log"
	"testing"
	"time"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

type rpc struct {
	reqC  chan *pb.PredictRequest
	respC chan *pb.PredictResponse
}

func (r *rpc) StepPredict(req *pb.PredictRequest, peerName string) (*pb.PredictResponse, error) {
	r.reqC <- req
	resp := <-r.respC
	if resp != nil {
		return resp, nil
	}
	return nil, errors.New("test response error")
}

func (r *rpc) StepPredictWithRetry(req *pb.PredictRequest, peerName string, times int, inteSec int64) (*pb.PredictResponse, error) {
	r.reqC <- req
	resp := <-r.respC
	if resp != nil {
		return resp, nil
	}
	return nil, errors.New("test response error")
}

type res struct {
	success  bool
	outcomes []byte
}
type resHandler struct {
	resC chan *res
}

func (rd *resHandler) SaveResult(result *pbCom.PredictTaskResult) {
	rd.resC <- &res{success: result.Success, outcomes: result.Outcomes}
}

func TestAdvance(t *testing.T) {
	// new model1
	var model1 *Model
	id1 := "test-model1-1"
	address1 := "127.0.0.1:8080"
	parties1 := []string{"127.0.0.1:8081"}
	params1 := &pbCom.TrainModels{
		Thetas: map[string]float64{
			"Sepal Length": -0.13275669547477614,
			"Sepal Width":  0.5802648528895868,
		},
		Xbars: map[string]float64{
			"Sepal Length": 5.8644444444444455,
			"Sepal Width":  3.0562962962962956,
		},
		Sigmas: map[string]float64{
			"Sepal Length": 0.8418204846611077,
			"Sepal Width":  0.4421508720900439,
		},
		Label:     "Label",
		IsTagPart: false,
		IdName:    "id",
	}
	var reqC1 = make(chan *pb.PredictRequest)
	var respC1 = make(chan *pb.PredictResponse)
	rpc1 := &rpc{
		reqC:  reqC1,
		respC: respC1,
	}

	var resC1 = make(chan *res)
	rh1 := &resHandler{
		resC: resC1,
	}

	predictFile1 := "../../testdata/vl/logic_iris_plants/predict_dataA.csv"
	samplesFile1, err := ioutil.ReadFile(predictFile1)
	checkErr(err, t)

	// new model2
	var model2 *Model
	id2 := "test-learner-2"
	address2 := "127.0.0.1:8081"
	parties2 := []string{"127.0.0.1:8080"}
	params2 := &pbCom.TrainModels{
		Thetas: map[string]float64{
			"Intercept":    -0.6811624184318745,
			"Petal Length": -0.7354833057126191,
			"Petal Width":  -0.6177165958123666,
		},
		Xbars: map[string]float64{
			"Label":        0.3333333333333333,
			"Petal Length": 3.782962962962963,
			"Petal Width":  1.2022222222222227,
		},
		Sigmas: map[string]float64{
			"Label":        0.4714045207910312,
			"Petal Length": 1.7768695210747902,
			"Petal Width":  0.7591873498015923,
		},
		Label:     "Label",
		IsTagPart: true,
		IdName:    "id",
	}

	var reqC2 = make(chan *pb.PredictRequest)
	var respC2 = make(chan *pb.PredictResponse)
	rpc2 := &rpc{
		reqC:  reqC2,
		respC: respC2,
	}
	var resC2 = make(chan *res)
	rh2 := &resHandler{
		resC: resC2,
	}

	predictFile2 := "../../testdata/vl/logic_iris_plants/predict_dataB.csv"
	samplesFile2, err := ioutil.ReadFile(predictFile2)
	checkErr(err, t)

	// test starts
	go func() {
		model1, err = NewModel(id1, address1, params1, samplesFile1, parties1, rpc1, rh1)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}()
	go func() {
		model2, err = NewModel(id2, address2, params2, samplesFile2, parties2, rpc2, rh2)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}()

	var done = make(chan int)
	var stop = make(chan int)
	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for {
		select {
		case reqRecv2 := <-reqC1:
			if model2 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate Model
			}
			if model2 != nil {
				message := reqRecv2.GetPayload()
				respSend2, err := model2.Advance(message)
				if err != nil {
					log.Printf("model2.Advance err: %s", err.Error())
				}
				respC1 <- respSend2
			} else {
				log.Printf("model1.AskRequst req: %v, model2 %v", reqRecv2, model2)
				respC1 <- nil
			}
		case res1 := <-resC1:
			if !res1.success { //failed
				log.Printf("model1[isPartTag:%t] failed to predict", params1.IsTagPart)
			} else {
				outcomes1, err := vl_common.PredictResultFromBytes(res1.outcomes)
				if err != nil {
					log.Printf("model1.PredictResultFromBytes err: %s", err.Error())
				}
				log.Printf("model1[isPartTag:%t] predict successfully, and outcomes[%v]", params1.IsTagPart, outcomes1)
			}

			if !res1.success || len(res1.outcomes) > 0 {
				go func() {
					stop <- 1
				}()
			}

		case reqRecv1 := <-reqC2:
			if model1 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate Model
			}
			if model1 != nil {
				message := reqRecv1.GetPayload()
				respSend1, err := model1.Advance(message)
				if err != nil {
					log.Printf("model1.Advance err: %s", err.Error())
				}
				respC2 <- respSend1
			} else {
				log.Printf("model2.AskRequst req: %v, model1 %v", reqRecv1, model1)
				respC2 <- nil
			}
		case res2 := <-resC2:
			if !res2.success { //failed
				log.Printf("model2[isPartTag:%t] failed to predict", params2.IsTagPart)
			} else {
				outcomes2, err := vl_common.PredictResultFromBytes(res2.outcomes)
				if err != nil {
					log.Printf("model2.PredictResultFromBytes err: %s", err.Error())
				}
				log.Printf("model2[isPartTag:%t] predict successfully, and outcomes[%v]", params2.IsTagPart, outcomes2)
			}

			if !res2.success || len(res2.outcomes) > 0 {
				go func() {
					stop <- 1
				}()
			}

		case <-stop:
			close(done)
		}

		if isDone() {
			break
		}
	}
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
