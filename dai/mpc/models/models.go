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

package models

import (
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/models/linear_reg_vl"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/models/logic_reg_vl"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

// Model was trained out by a Learner,
// and participates in the multi-parts-calculation during prediction process
// If input different parts of a sample into Models on different mpc-nodes, you'll get final predicting result after some time of multi-parts-calculation
type Model interface {
	// Advance does calculation with local parts of samples and communicates with other nodes in cluster to predict outcomes
	// payload could be resolved by Model trained out by specific algorithm and samples
	// We'd better call the method asynchronously avoid blocking the main go-routine
	Advance(payload []byte) (*pb.PredictResponse, error)
}

// RpcHandler used to request remote mpc-node
type RpcHandler interface {
	StepPredict(req *pb.PredictRequest, peerName string) (*pb.PredictResponse, error)

	// StepPredictWithRetry sends prediction message to remote mpc-node
	// retries 2 times at most
	// inteSec indicates the interval between retry requests, in seconds
	StepPredictWithRetry(req *pb.PredictRequest, peerName string, times int, inteSec int64) (*pb.PredictResponse, error)
}

// ResultHandler handles final result which is successful or failed
// Should be called when prediction finished
type ResultHandler interface {
	SaveResult(*pbCom.PredictTaskResult)
}

// NewModel returns a Model
// id is the assigned id for Model
// samplesFile is sample file content for prediction
// address indicates local mpc-node
// algo is the algorithm of model
// parties are other models who participates in MPC, assigned with mpc-node address usually
// rpc is used to request remote mpc-node
// rh handles final result which is successful or failed
// params are parameters for model
func NewModel(id string, address string, algo pbCom.Algorithm,
	params *pbCom.TrainModels, samplesFile []byte,
	parties []string, rpc RpcHandler, rh ResultHandler) (Model, error) {

	if pbCom.Algorithm_LINEAR_REGRESSION_VL == algo {
		return linear_reg_vl.NewModel(id, address, params, samplesFile,
			parties, rpc, rh)
	} else { // pbCom.Algorithm_LOGIC_REGRESSION_VL
		return logic_reg_vl.NewModel(id, address, params, samplesFile,
			parties, rpc, rh)
	}
}
