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

package learners

import (
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/learners/linear_reg_vl"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/learners/logic_reg_vl"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

// Learner is assigned with a specific algorithm and data used for training a model
// participates in the multi-parts-calculation during training process
type Learner interface {
	// Advance does calculation with local data and communicates with other nodes in cluster to train a model step by step
	// When we implement the method, we should pay attention to performance in case it takes a long time and blocks the client.
	// payload could be resolved by Learner defined by specific algorithm
	Advance(payload []byte) (*pb.TrainResponse, error)
}

// RpcHandler used to request remote mpc-node
type RpcHandler interface {
	StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error)
}

// ResultHandler handles final result which is successful or failed
// Should be called when learning finished
type ResultHandler interface {
	SaveResult(*pbCom.TrainTaskResult)
}

// NewLearner returns a Learner defined by algorithm and training samples
// id is the assigned id for Learner
// address indicates local mpc-node
// algo is the assigned algorithm for learner
// parties are other learners who participates in MPC, assigned with mpc-node address usually
// rpc is used to request remote mpc-node
// rh handles final result which is successful or failed
// params are parameters for training model
// samplesFile contains samples for training model
func NewLearner(id string, address string, algo pbCom.Algorithm,
	params *pbCom.TrainParams, samplesFile []byte,
	parties []string, rpc RpcHandler, rh ResultHandler) (Learner, error) {
	if pbCom.Algorithm_LINEAR_REGRESSION_VL == algo {
		return linear_reg_vl.NewLearner(id, address, params, samplesFile,
			parties, rpc, rh)
	} else { // pbCom.Algorithm_LOGIC_REGRESSION_VL
		return logic_reg_vl.NewLearner(id, address, params, samplesFile,
			parties, rpc, rh)
	}
}
