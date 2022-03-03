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
	AlgorithmVLine = "linear-vl"   // linear regression with multiple variables in vertical federated learning
	AlgorithmVLog  = "logistic-vl" // logistic regression with multiple variables in vertical federated learning

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
}

var VlAlgorithmListValue = map[pbCom.Algorithm]string{
	pbCom.Algorithm_LINEAR_REGRESSION_VL: AlgorithmVLine,
	pbCom.Algorithm_LOGIC_REGRESSION_VL:  AlgorithmVLog,
}

// TaskTypeListName the mapping of train task type name and value
var TaskTypeListName = map[string]pbCom.TaskType{
	TaskTypeTrain:   pbCom.TaskType_LEARN,
	TaskTypePredict: pbCom.TaskType_PREDICT,
}

var TaskTypeListValue = map[pbCom.TaskType]string{
	pbCom.TaskType_LEARN:   TaskTypeTrain,
	pbCom.TaskType_PREDICT: TaskTypePredict,
}

// RegModeListName the mapping of train regMode name and value
var RegModeListName = map[string]pbCom.RegMode{
	RegModeL1: pbCom.RegMode_Reg_Lasso,
	RegModeL2: pbCom.RegMode_Reg_Ridge,
}

var RegModeListValue = map[pbCom.RegMode]string{
	pbCom.RegMode_Reg_Lasso: RegModeL1,
	pbCom.RegMode_Reg_Ridge: RegModeL2,
}

type FLInfo struct {
	FileType  string // file type, only supports "csv"
	Features  string // feature list
	TotalRows int64 // total number of samples
}

// ExecutorNode has access to samples with which to train models or to predict,
//  and starts task that multi parties execute synchronically
type ExecutorNode struct {
	ID      []byte
	Name    string
	Address string // local host
	RegTime int64  // node registering time
}

type ExecutorNodes []ExecutorNode

// FLTask defines Federated Learning Task based on MPC
type FLTask *pbTask.FLTask

type FLTasks []*pbTask.FLTask

// PublishFLTaskOptions contains parameters for publishing tasks
type PublishFLTaskOptions struct {
	FLTask    FLTask
	Signature []byte
}

// ListFLTaskOptions contains parameters for listing tasks
// support listing tasks a requester published or tasks an executor involved
type ListFLTaskOptions struct {
	PubKey    []byte // requester or executor's public key
	Status    string // task status
	TimeStart int64  // task publish time period, only task published after TimeStart and before TimeEnd will be listed
	TimeEnd   int64
	Limit     int64 // limit number of tasks in list request, default 'all'
}

// FLTaskConfirmOptions contains parameters for confirming task
type FLTaskConfirmOptions struct {
	Pubkey       []byte // one of the task executor's public key
	TaskID       string
	RejectReason string // reason of the rejected task
	CurrentTime  int64  // time when confirming task

	Signature []byte // executor's signature
}

// FLTaskExeStatusOptions contains parameters for updating executing task
type FLTaskExeStatusOptions struct {
	Executor    []byte
	TaskID      string
	CurrentTime int64 // task execute start time or finish time

	Signature []byte

	ErrMessage string // for failed task
	Result     string // for finished task
}

// AddNodeOptions contains parameters for adding node of Executor
type AddNodeOptions struct {
	Node      ExecutorNode
	Signature []byte
}
