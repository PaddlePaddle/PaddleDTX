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
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/p2p"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

var (
	logger = logrus.WithField("module", "mpc.cluster")
)

// Rpc performs remote procedure calls to remote cluster nodes.
//  PredictHandler could be called during prediction
//  TrainHandler could be called during training
type Rpc interface {
	PredictHandler
	TrainHandler
}

type PredictHandler interface {
	StepPredict(req *pb.PredictRequest, peerName string) (*pb.PredictResponse, error)

	// StepPredictWithRetry sends prediction message to remote mpc-node
	// retries 2 times at most
	// inteSec indicates the interval between retry requests, in seconds
	StepPredictWithRetry(req *pb.PredictRequest, peerName string, times int, inteSec int64) (*pb.PredictResponse, error)
}

type TrainHandler interface {
	StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error)

	// StepTrainWithRetry sends training message to remote mpc-node
	// retries 2 times at most
	// inteSec indicates the interval between retry requests, in seconds
	StepTrainWithRetry(req *pb.TrainRequest, peerName string, times int, inteSec int64) (*pb.TrainResponse, error)
}

// P2P is used to get rpc connection to remote cluster nodes,
// remember to call FreePeer() when rpc requests finish
type P2P interface {
	GetPeer(address string) (*p2p.Peer, error)
	FreePeer()
}

// RpcClient implements Rpc interface,
//  performs remote procedure calls to remote cluster nodes.
type RpcClient struct {
	timeout time.Duration
	cluster P2P
}

func (rc *RpcClient) StepPredict(req *pb.PredictRequest, peerName string) (*pb.PredictResponse, error) {
	peer, err := rc.cluster.GetPeer(peerName)
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeRPCFindNoPeer, "failed to get peer %s when do rpc request: %s", peerName, err.Error())
	}
	defer rc.cluster.FreePeer()

	conn, err := peer.GetConnect()
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeRPCConnect, "failed to get connection with %s: %s", peerName, err.Error())
	}

	c := pb.NewClusterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), rc.timeout)
	defer cancel()

	stepReq := &pb.StepRequest{
		Payload: &pb.StepRequest_PredictRequest{
			PredictRequest: req,
		},
	}
	stepResp, err := c.Step(ctx, stepReq)
	if err != nil {
		logger.Warningf("Step response is error: %s", err.Error())
		return nil, err
	}
	resp := stepResp.GetPredictResponse()
	return resp, err
}

// StepPredictWithRetry sends prediction message to remote mpc-node
// retries 2 times at most
// inteSec indicates the interval between retry requests, in seconds
func (rc *RpcClient) StepPredictWithRetry(req *pb.PredictRequest, peerName string, times int, inteSec int64) (*pb.PredictResponse, error) {
	if times <= 0 {
		times = 1
	} else if times > 2 {
		times = 3
	} else {
		times += 1
	}

	var errR error
	for i := 0; i < times; i++ {
		if i > 0 {
			time.Sleep(time.Duration(inteSec) * time.Second)
		}
		resp, err := rc.StepPredict(req, peerName)
		if err == nil {
			return resp, err
		}
		errR = err
	}

	return nil, errR
}

func (rc *RpcClient) StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error) {
	peer, err := rc.cluster.GetPeer(peerName)
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeRPCFindNoPeer, "failed to get peer %s when do rpc request: %s", peerName, err.Error())
	}
	defer rc.cluster.FreePeer()

	conn, err := peer.GetConnect()
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeRPCConnect, "failed to get connection with %s: %s", peerName, err.Error())
	}

	c := pb.NewClusterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), rc.timeout)
	defer cancel()

	stepReq := &pb.StepRequest{
		Payload: &pb.StepRequest_TrainRequest{
			TrainRequest: req,
		},
	}
	stepResp, err := c.Step(ctx, stepReq)
	if err != nil {
		logger.Warningf("Step response is error: %s", err.Error())
		return nil, err
	}

	resp := stepResp.GetTrainResponse()
	return resp, err
}

// StepTrainWithRetry sends training message to remote mpc-node
// retries 2 times at most
// inteSec indicates the interval between retry requests, in seconds
func (rc *RpcClient) StepTrainWithRetry(req *pb.TrainRequest, peerName string, times int, inteSec int64) (*pb.TrainResponse, error) {
	if times <= 0 {
		times = 1
	} else if times > 2 {
		times = 3
	} else {
		times += 1
	}

	var errR error
	for i := 0; i < times; i++ {
		if i > 0 {
			time.Sleep(time.Duration(inteSec) * time.Second)
		}
		resp, err := rc.StepTrain(req, peerName)
		if err == nil {
			return resp, err
		}
		errR = err
	}

	return nil, errR
}

// NewRpcClient returns RpcClient instance
// timeout eg. 3*time.Second
// connection releases when timeout elapses
func NewRpcClient(clu P2P, timeout time.Duration) Rpc {
	rc := &RpcClient{
		cluster: clu,
		timeout: timeout,
	}
	return rc
}
