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

package blockchain

import (
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
)

const (
	/* Define Task Status stored in Contract */
	TaskConfirming = "Confirming" // waiting for Executors to confirm
	TaskReady      = "Ready"      // has been confirmed by all Executors, and ready to start
	TaskToProcess  = "ToProcess"  // has started, and waiting to be precessed
	TaskProcessing = "Processing" // under process, that's during training or predicting
	TaskFinished   = "Finished"   // task finished
	TaskFailed     = "Failed"     // task failed
	TaskRejected   = "Rejected"   // task rejected by one of the Executors

	/* Define Task Type stored in Contract */
	TaskTypeTrain   = "train"   // training task
	TaskTypePredict = "predict" // prediction task

	/* Define Algorithms stored in Contract */
	AlgorithmVLine = "linear-vl"       // linear regression with multiple variables in vertical federated learning
	AlgorithmVLog  = "logistic-vl"     // logistic regression with multiple variables in vertical federated learning
	AlgorithmVDnn  = "dnn-paddlefl-vl" // dnn implemented using paddlefl

	/* Define Regularization stored in Contract */
	RegModeL1 = "l1" // L1-norm
	RegModeL2 = "l2" // L2-norm

	/* Define the maximum number of task list query */
	TaskListMaxNum = 100
)

// VlAlgorithmListName the mapping of vertical algorithm name and value
var VlAlgorithmListName = map[string]pbCom.Algorithm{
	AlgorithmVLine: pbCom.Algorithm_LINEAR_REGRESSION_VL,
	AlgorithmVLog:  pbCom.Algorithm_LOGIC_REGRESSION_VL,
	AlgorithmVDnn:  pbCom.Algorithm_DNN_PADDLEFL_VL,
}

// VlAlgorithmListValue the mapping of vertical algorithm value and name
// key is the int value of the algorithm
var VlAlgorithmListValue = map[pbCom.Algorithm]string{
	pbCom.Algorithm_LINEAR_REGRESSION_VL: AlgorithmVLine,
	pbCom.Algorithm_LOGIC_REGRESSION_VL:  AlgorithmVLog,
	pbCom.Algorithm_DNN_PADDLEFL_VL:      AlgorithmVDnn,
}

// TaskTypeListName the mapping of train task type name and value
// key is the task type name of the training task or prediction task
var TaskTypeListName = map[string]pbCom.TaskType{
	TaskTypeTrain:   pbCom.TaskType_LEARN,
	TaskTypePredict: pbCom.TaskType_PREDICT,
}

// TaskTypeListValue the mapping of train task type value and name
// key is the int value of the training task or prediction task
var TaskTypeListValue = map[pbCom.TaskType]string{
	pbCom.TaskType_LEARN:   TaskTypeTrain,
	pbCom.TaskType_PREDICT: TaskTypePredict,
}

// RegModeListName the mapping of train regMode name and value
var RegModeListName = map[string]pbCom.RegMode{
	RegModeL1: pbCom.RegMode_Reg_Lasso,
	RegModeL2: pbCom.RegMode_Reg_Ridge,
}

// RegModeListValue the mapping of train regMode value and name
var RegModeListValue = map[pbCom.RegMode]string{
	pbCom.RegMode_Reg_Lasso: RegModeL1,
	pbCom.RegMode_Reg_Ridge: RegModeL2,
}

// FLInfo used to parse the content contained in the extra field of the file on the chain,
// only files that can be parsed can be used for task training or prediction
type FLInfo struct {
	FileType  string `json:"fileType"`  // file type, only supports "csv"
	Features  string `json:"features"`  // feature list
	TotalRows int64  `json:"totalRows"` // total number of samples
}

// ExecutorNode has access to samples with which to train models or to predict,
//  and starts task that multi parties execute synchronically
type ExecutorNode struct {
	ID              []byte `json:"id"`
	Name            string `json:"name"`
	Address         string `json:"address"`     // local grpc host
	HttpAddress     string `json:"httpAddress"` // local http host
	PaddleFLAddress string `json:"paddleFLAddress"`
	PaddleFLRole    int    `json:"paddleFLRole"`
	RegTime         int64  `json:"regTime"` // node registering time
}

type ExecutorNodes []ExecutorNode

// FLTask defines Federated Learning Task based on MPC
type FLTask *pbTask.FLTask

type FLTasks []*pbTask.FLTask

// PublishFLTaskOptions contains parameters for publishing tasks
type PublishFLTaskOptions struct {
	FLTask    FLTask `json:"fLTask"`
	Signature []byte `json:"signature"`
}

// StartFLTaskOptions contains parameters for the requester to start tasks
type StartFLTaskOptions struct {
	TaskID    string `json:"taskID"`
	Signature []byte `json:"signature"`
}

// ListFLTaskOptions contains parameters for listing tasks
// support listing tasks a requester published or tasks an executor involved
type ListFLTaskOptions struct {
	PubKey    []byte `json:"pubKey"`    // requester or executor's public key
	Status    string `json:"status"`    // task status
	TimeStart int64  `json:"timeStart"` // task publish time period, only task published after TimeStart and before TimeEnd will be listed
	TimeEnd   int64  `json:"timeEnd"`
	Limit     int64  `json:"limit"` // limit number of tasks in list request, default 'all'
}

// FLTaskConfirmOptions contains parameters for confirming task
type FLTaskConfirmOptions struct {
	Pubkey       []byte `json:"pubkey"` // one of the task executor's public key
	TaskID       string `json:"taskID"`
	RejectReason string `json:"rejectReason"` // reason of the rejected task
	CurrentTime  int64  `json:"currentTime"`  // time when confirming task

	Signature []byte `json:"signature"` // executor's signature
}

// FLTaskExeStatusOptions contains parameters for updating executing task
type FLTaskExeStatusOptions struct {
	Executor    []byte `json:"executor"`
	TaskID      string `json:"taskID"`
	CurrentTime int64  `json:"currentTime"` // task execute start time or finish time

	Signature []byte `json:"signature"`

	ErrMessage string `json:"errMessage"` // for failed task
	Result     string `json:"result"`     // for finished task
}

// AddNodeOptions contains parameters for adding node of Executor
type AddNodeOptions struct {
	Node      ExecutorNode `json:"node"`
	Signature []byte       `json:"signature"`
}
