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
}

type TrainHandler interface {
	StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error)
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
