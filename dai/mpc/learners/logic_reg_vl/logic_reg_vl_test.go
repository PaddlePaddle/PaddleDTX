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
	reqC  chan *pb.TrainRequest
	respC chan *pb.TrainResponse
}

func (r *rpc) StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error) {
	r.reqC <- req
	resp := <-r.respC
	if resp != nil {
		return resp, nil
	}
	return nil, errors.New("test response error")
}

type resHandler struct {
	modelC chan *[]byte
}

func (rd *resHandler) SaveResult(res *pbCom.TrainTaskResult) {
	if res.Success {
		rd.modelC <- &res.Model
	} else {
		log.Printf("training failed, and reason is %s.", res.ErrMsg)
		rd.modelC <- nil
	}
}

func TestAdvance(t *testing.T) {
	// new learner1
	var learner1 *Learner
	id1 := "test-learner-1"
	address1 := "127.0.0.1:8080"
	parties1 := []string{"127.0.0.1:8081"}
	params1 := &pbCom.TrainParams{
		Label:     "Label",
		LabelName: "Iris-setosa",
		RegMode:   0,
		RegParam:  0.1,
		Alpha:     0.1,
		Amplitude: 0.0001,
		Accuracy:  10,
		IsTagPart: false,
		IdName:    "id",
		BatchSize: 4,
	}
	var reqC1 = make(chan *pb.TrainRequest)
	var respC1 = make(chan *pb.TrainResponse)
	rpc1 := &rpc{
		reqC:  reqC1,
		respC: respC1,
	}
	var modelC1 = make(chan *[]byte)
	rh1 := &resHandler{
		modelC: modelC1,
	}

	trainFile1 := "../../testdata/vl/logic_iris_plants/train_dataA.csv"
	samplesFile1, err := ioutil.ReadFile(trainFile1)
	checkErr(err, t)

	// new learner2
	var learner2 *Learner
	id2 := "test-learner-2"
	address2 := "127.0.0.1:8081"
	parties2 := []string{"127.0.0.1:8080"}
	params2 := &pbCom.TrainParams{
		Label:     "Label",
		LabelName: "Iris-setosa",
		RegMode:   0,
		RegParam:  0.1,
		Alpha:     0.1,
		Amplitude: 0.0001,
		Accuracy:  10,
		IsTagPart: true,
		IdName:    "id",
		BatchSize: 4,
	}

	var reqC2 = make(chan *pb.TrainRequest)
	var respC2 = make(chan *pb.TrainResponse)
	rpc2 := &rpc{
		reqC:  reqC2,
		respC: respC2,
	}
	var modelC2 = make(chan *[]byte)
	rh2 := &resHandler{
		modelC: modelC2,
	}

	trainFile2 := "../../testdata/vl/logic_iris_plants/train_dataB.csv"
	samplesFile2, err := ioutil.ReadFile(trainFile2)
	checkErr(err, t)

	// test starts
	go func() {
		learner1, err = NewLearner(id1, address1, params1, samplesFile1, parties1, rpc1, rh1)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}()
	go func() {
		learner2, err = NewLearner(id2, address2, params2, samplesFile2, parties2, rpc2, rh2)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}()

	var done = make(chan int)
	var stop = make(chan int)
	var stopped1 bool
	var stopped2 bool

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
			if learner2 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			if learner2 != nil {
				message := reqRecv2.GetPayload()
				respSend2, err := learner2.Advance(message)
				if err != nil {
					log.Printf("learner2.Advance err: %s", err.Error())
				}
				respC1 <- respSend2
			} else {
				log.Printf("learner1.AskRequst req: %v, learner2 %v", reqRecv2, learner2)
				respC1 <- nil
			}
		case model1 := <-modelC1:
			if model1 == nil { //failed
				log.Printf("learner1 train model failed")
			} else {
				model, _ := vl_common.TrainModelsFromBytes(*model1)
				log.Printf("learner1 train model[%v] successfully", model)
			}
			stopped1 = true
			if stopped1 && stopped2 {
				go func() {
					stop <- 1
				}()
			}

		case reqRecv1 := <-reqC2:
			if learner1 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			if learner1 != nil {
				message := reqRecv1.GetPayload()
				respSend1, err := learner1.Advance(message)
				if err != nil {
					log.Printf("learner1.Advance err: %s", err.Error())
				}
				respC2 <- respSend1
			} else {
				log.Printf("learner2.AskRequst req: %v, learner1 %v", reqRecv1, learner1)
				respC2 <- nil
			}
		case model2 := <-modelC2:
			if model2 == nil { //failed
				log.Printf("learner2 train model failed")
			} else {
				model, _ := vl_common.TrainModelsFromBytes(*model2)
				log.Printf("learner2 train model[%v] successfully", model)
			}
			stopped2 = true
			if stopped1 && stopped2 {
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
