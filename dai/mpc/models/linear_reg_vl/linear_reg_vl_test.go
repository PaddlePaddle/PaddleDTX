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

package linear_reg_vl

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
			"CHAS":  0.06728268998721598,
			"CRIM":  -0.17945584445716345,
			"INDUS": -0.009346873185743485,
			"NOX":   -0.12257710377277357,
			"RM":    0.29176457994903005,
			"ZN":    0.1082433286977407,
		},
		Xbars: map[string]float64{
			"CHAS":  0.06798245614035088,
			"CRIM":  3.66938839912281,
			"INDUS": 11.105986842105281,
			"NOX":   0.5559028508771933,
			"RM":    6.282947368421057,
			"ZN":    10.843201754385966,
		},
		Sigmas: map[string]float64{
			"CHAS":  0.2517157956852844,
			"CRIM":  8.876833192136589,
			"INDUS": 6.88002233713496,
			"NOX":   0.11530061192506116,
			"RM":    0.6990277595585356,
			"ZN":    22.407805213487748,
		},
		Label:     "MEDV",
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

	predictFile1 := "../../testdata/vl/linear_boston_housing/predict_dataA.csv"
	samplesFile1, err := ioutil.ReadFile(predictFile1)
	checkErr(err, t)

	// new model2
	var model2 *Model
	id2 := "test-learner-2"
	address2 := "127.0.0.1:8081"
	parties2 := []string{"127.0.0.1:8080"}
	params2 := &pbCom.TrainModels{
		Thetas: map[string]float64{
			"AGE":       -0.09234474031055216,
			"B":         0.0504543586030969,
			"DIS":       -0.28368362388987955,
			"Intercept": 0.09981553681000002,
			"LSTAT":     -0.5171885214789168,
			"PTRATIO":   -0.1787815582923953,
			"RAD":       0.0852030754474994,
			"TAX":       -0.1381661164703932,
		},
		Xbars: map[string]float64{
			"AGE":     68.68355263157898,
			"B":       355.7912280701747,
			"DIS":     3.7703414473684203,
			"LSTAT":   12.748333333333346,
			"MEDV":    22.525000000000006,
			"PTRATIO": 18.443859649122768,
			"RAD":     9.55921052631579,
			"TAX":     408.36622807017545,
		},
		Sigmas: map[string]float64{
			"AGE":     28.088745117770717,
			"B":       92.34683128560424,
			"DIS":     2.1040263198495963,
			"LSTAT":   7.269428525122956,
			"MEDV":    9.22892513841814,
			"PTRATIO": 2.173294253000775,
			"RAD":     8.698961588032509,
			"TAX":     168.72716115696468,
		},
		Label:     "MEDV",
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

	predictFile2 := "../../testdata/vl/linear_boston_housing/predict_dataB.csv"
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
