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

package mpc

import (
	"strconv"
	"testing"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/dai/p2p"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

var (
	testP2P = p2p.NewP2P()
)

type testModelHolder struct {
}

func (tmh *testModelHolder) SaveModel(result *pbCom.TrainTaskResult) error {
	return nil
}

func (tmh *testModelHolder) SavePredictOut(result *pbCom.PredictTaskResult) error {
	return nil
}

func TestMpc(t *testing.T) {
	mh := &testModelHolder{}

	config := Config{
		Address:          "127.0.0.1:8080",
		TrainTaskLimit:   5,
		PredictTaskLimit: 10,
		RpcTimeout:       3,
	}
	testMpc := StartMpc(mh, testP2P, config)

	// test StartTask and StopTask
	numStartT := 6
	var startTaskReqs []*pbCom.StartTaskRequest
	for i := 0; i < numStartT; i++ {
		startTaskReqs = append(startTaskReqs, &pbCom.StartTaskRequest{
			TaskID: "Test-Mpc" + strconv.Itoa(i),
			File:   []byte("Hello,world"),
			Hosts:  []string{"127.0.0.1:8081"},
			Params: &pbCom.TaskParams{
				Algo:        pbCom.Algorithm_LINEAR_REGRESSION_VL,
				TaskType:    pbCom.TaskType_LEARN,
				TrainParams: &pbCom.TrainParams{},
				ModelParams: &pbCom.TrainModels{},
			},
		})
	}

	for i := 0; i < numStartT*2; i++ {
		startTaskReqs = append(startTaskReqs, &pbCom.StartTaskRequest{
			TaskID: "Test-Mpc" + strconv.Itoa(i),
			File:   []byte("Hello,world"),
			Hosts:  []string{"127.0.0.1:8081"},
			Params: &pbCom.TaskParams{
				Algo:        pbCom.Algorithm_LOGIC_REGRESSION_VL,
				TaskType:    pbCom.TaskType_PREDICT,
				TrainParams: &pbCom.TrainParams{},
				ModelParams: &pbCom.TrainModels{},
			},
		})
	}

	for i := 0; i < numStartT*3; i++ {
		go func(req *pbCom.StartTaskRequest) {
			err := testMpc.StartTask(req)
			if err != nil {
				t.Logf("failed StartTask[%v], error: %s", req, err.Error())
			} else {
				t.Logf("StartTask[%v]", req)
			}
		}(startTaskReqs[i])
	}

	time.Sleep(3 * time.Second)

	// test StopTask
	stopTaskReq := &pbCom.StopTaskRequest{
		TaskID: "Test-Mpc-StopTask",
	}
	err := testMpc.StopTask(stopTaskReq)
	if err != nil {
		t.Logf("failed StopTask[%v], error: %s", stopTaskReq, err.Error())
	} else {
		t.Logf("StopTask[%v]", stopTaskReq)
	}

	// test Train
	trainReq := &pb.TrainRequest{
		TaskID:  "TrainRequest-test",
		Algo:    pbCom.Algorithm_LINEAR_REGRESSION_VL,
		Payload: []byte("Hello! World!"),
	}
	resp, err := testMpc.Train(trainReq)
	t.Logf("Train request[%v], response[%v], error: %v", trainReq, resp, err.Error())

	// test Predict
	predictReq := &pb.PredictRequest{
		TaskID:  "PredictRequest-test",
		Algo:    pbCom.Algorithm_LINEAR_REGRESSION_VL,
		Payload: []byte("Hello! World!"),
	}
	respP, err := testMpc.Predict(predictReq)
	t.Logf("Predict request[%v], response[%v], error: %v", predictReq, respP, err.Error())

	// test Stop
	testMpc.Stop()

}
