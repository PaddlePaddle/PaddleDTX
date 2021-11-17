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

package cluster

import (
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/PaddlePaddle/PaddleDTX/dai/p2p"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

var (
	serverPort = "8080"
	testP2P    = p2p.NewP2P()
)

type mpc struct {
}

func (m *mpc) Predict(req *pb.PredictRequest) (resp *pb.PredictResponse, err error) {
	resp = &pb.PredictResponse{
		TaskID:  req.GetTaskID() + "-PredictResponse",
		Payload: req.GetPayload(),
	}
	return
}

func (m *mpc) Train(req *pb.TrainRequest) (resp *pb.TrainResponse, err error) {
	resp = &pb.TrainResponse{
		TaskID:  req.GetTaskID() + "-TrainResponse",
		Payload: req.GetPayload(),
	}
	return
}

func TestRpc(t *testing.T) {
	// test service
	go runServer(t)
	time.Sleep(time.Duration(3) * time.Second)

	// test rpc.StepPredict
	rpcH := NewRpcClient(testP2P, 3*time.Second)
	req := &pb.PredictRequest{
		TaskID:  "Test-Cluster-StepPredict",
		Algo:    pbCom.Algorithm_LINEAR_REGRESSION_VL,
		Payload: []byte("Hello-This-Is-PredictRequest-Test"),
	}
	resp, err := rpcH.StepPredict(req, "127.0.0.1:"+serverPort)
	if err != nil {
		checkErr(err, t)
	}
	t.Logf("StepPredict.PredictRequest[%v], and Response[%v]", req, resp)

	// test rpc.StepTrain
	reqT := &pb.TrainRequest{
		TaskID:  "Test-Cluster-StepTrain",
		Algo:    pbCom.Algorithm_LINEAR_REGRESSION_VL,
		Payload: []byte("Hello-This-Is-PredictRequest-Test"),
	}
	respT, err := rpcH.StepTrain(reqT, "127.0.0.1:"+serverPort)
	if err != nil {
		checkErr(err, t)
	}
	t.Logf("StepTrain.PredictRequest[%v], and Response[%v]", reqT, respT)

	testP2P.Stop()
}

func runServer(t *testing.T) {
	var rpcOptions []grpc.ServerOption

	t.Log("start rpc server ...")
	t.Logf("port is: %s", serverPort)
	server := grpc.NewServer(rpcOptions...)
	lis, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		checkErr(err, t)
	}

	testMpc := &mpc{}
	service := NewService(testMpc)
	service.RegisterClusterServer(server)

	if err := server.Serve(lis); err != nil {
		checkErr(err, t)
	}

	t.Logf("rpc server started. server: %v", server)
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
