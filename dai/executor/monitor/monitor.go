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

package monitor

import (
	"context"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	xdbchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

var (
	logger = logrus.WithField("module", "monitor.task")
)

type Blockchain interface {
	// task operation
	ListTask(opt *blockchain.ListFLTaskOptions) (blockchain.FLTasks, error)
	ExecuteTask(opt *blockchain.FLTaskExeStatusOptions) error
	ConfirmTask(opt *blockchain.FLTaskConfirmOptions) error
	RejectTask(opt *blockchain.FLTaskConfirmOptions) error
	// query the list of authorization applications
	ListFileAuthApplications(opt *xdbchain.ListFileAuthOptions) (xdbchain.FileAuthApplications, error)
	// publish sample file's authorization application
	PublishFileAuthApplication(opt *xdbchain.PublishFileAuthOptions) error
}

type MpcHandler interface {
	// TaskStartPrepare prepare resources before starting local MPC task, like parameters and sample data
	TaskStartPrepare(task blockchain.FLTask) (*pbCom.StartTaskRequest, error)
	// StartLocalMpcTask start local mpc task
	// task required parameters passed when starting local task training
	StartLocalMpcTask(task *pbCom.StartTaskRequest, isSendTaskToOthers bool) error
	// GetAvailableTasksNum get available numbers of task execution resources
	// used check how many tasks could be handled this round
	GetAvailableTasksNum() (int, int)
	// CheckMpcTimeOutTasks checks tasks in execution pool if they're expired,
	// and stops expired tasks
	CheckMpcTimeOutTasks()
}

// TaskMonitor
type TaskMonitor struct {
	ExecutionType   string // mode for downloading sample files during tasks execution
	PrivateKey      ecdsa.PrivateKey
	PublicKey       ecdsa.PublicKey
	RequestInterval time.Duration // task loop interval

	Blockchain Blockchain // task contract invoke
	MpcHandler MpcHandler

	doneLoopReqC  chan struct{} // doneLoopReqC closed when loop breaks
	doneRetryReqC chan struct{} // doneRetryReqC closed when processing task retry end
}

// StartTaskLoopRequest starts timed task which will block until receive Stop signal
func (t *TaskMonitor) StartTaskLoopRequest(ctx context.Context) {
	go t.loopRequest(ctx)
}

// StopLoopReq wait for t.StopLoopReq() to quit
func (t *TaskMonitor) StopLoopReq() {
	if t.doneLoopReqC == nil { //avoid block
		return
	}

	logger.Info("TaskLoopRequest stops ...")
	select {
	case <-t.doneLoopReqC:
		// already closed
		return
	default:
	}
	// wait for t.loopRequest() to quit
	<-t.doneLoopReqC
}

// StopRetryReq wait for t.RetryProcessingTask() to quit
func (t *TaskMonitor) StopRetryReq() {
	if t.doneRetryReqC == nil {
		return
	}

	logger.Info("TaskRetryRequest stops ...")

	select {
	case <-t.doneRetryReqC:
		return
	default:
	}

	<-t.doneRetryReqC
}
