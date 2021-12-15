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
	"fmt"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

// loopRequest checks blockchain every some seconds to find tasks ready to execute,
// then starts Multi-Party Computation for each task.
// And checks tasks in execution pool if they're expired,
// then stops expired tasks
func (t *TaskMonitor) loopRequest(ctx context.Context) {
	logger.Info("task loop start")

	ticker := time.NewTicker(t.RequestInterval)
	defer ticker.Stop()

	t.doneLoopReqC = make(chan struct{})
	defer close(t.doneLoopReqC)

	defer logger.Info("task loop stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		//checks blockchain every some seconds to find tasks ready to execute,
		//then starts Multi-Party Computation for each task.
		if err := t.getToProcessTaskAndStart(); err != nil {
			logger.WithError(err).Error("failed to find taskToProcess task list")
		}

		//checks tasks in execution pool if they're expired,
		// then stops expired tasks
		t.MpcHandler.CheckMpcTimeOutTasks()
	}
}

// getToProcessTaskAndStart process tasks in Processing status when restarting server
//  this step is necessary because when participants abnormally exit computation process
//  the task may be in Processing stage forever
func (t *TaskMonitor) getToProcessTaskAndStart() error {
	// 1. find all ready tasks from chain
	taskList, err := t.Blockchain.ListTask(&blockchain.ListFLTaskOptions{
		PubKey:    t.PublicKey[:],
		Status:    blockchain.TaskToProcess,
		TimeStart: 0,
		TimeEnd:   time.Now().UnixNano(),
		Limit:     blockchain.TaskListMaxNum,
	})
	if err != nil {
		return errorx.Wrap(err, "failed to find ToProcess task list")
	}
	if len(taskList) == 0 {
		logger.WithField("amount", len(taskList)).Debug("no task found")
		return nil
	}

	for _, task := range taskList {
		// 2. verify whether the training or predicting task resources pool is full
		trainAvailableNum, predictAvailableNum := t.MpcHandler.GetAvailableTasksNum()
		if task.AlgoParam.TaskType == pbCom.TaskType_LEARN && trainAvailableNum == 0 {
			logger.Info("Training task resources is full")
			continue
		}
		if task.AlgoParam.TaskType == pbCom.TaskType_PREDICT && predictAvailableNum == 0 {
			logger.Info("Predicting task resources is full")
			continue
		}

		// 3. update task status
		if err := t.updateTaskExecStatus(task.ID); err != nil {
			continue
		}
		// 4. prepare resources before starting local MPC task
		startRequest, err := t.MpcHandler.TaskStartPrepare(task)
		if err != nil {
			logger.WithError(err).Errorf("error occurred when task start prepare, and taskId: %s", task.ID)
			continue
		}
		// 5. start local task
		logger.Infof("start ToProcess task of loop, taskId: %s", task.ID)
		if err := t.MpcHandler.StartLocalMpcTask(startRequest, true); err != nil {
			logger.WithError(err).Errorf("error occurred when execute task, and taskId: %s", task.ID)
			continue
		}
	}

	logger.WithFields(logrus.Fields{
		"task_len": len(taskList),
		"end_time": time.Now().Format("2006-01-02 15:04:05"),
	}).Info("tasks execution finished of each round")
	return nil
}

// RetryProcessingTask process tasks in Processing status when restarting server
//  this step is necessary because when participants abnormally exit computation process
//  the task may be in Processing stage forever
func (t *TaskMonitor) RetryProcessingTask(ctx context.Context) {
	logger.Info("processing tasks retry execution start")

	t.doneRetryReqC = make(chan struct{})
	defer close(t.doneRetryReqC)

	defer logger.Info("processing tasks retry end")

	// 1. find all processing task
	taskList, err := t.Blockchain.ListTask(&blockchain.ListFLTaskOptions{
		PubKey:    t.PublicKey[:],
		Status:    blockchain.TaskProcessing,
		TimeStart: 0,
		TimeEnd:   time.Now().UnixNano(),
	})

	if err != nil {
		logger.WithError(err).Error("failed to find TaskProcessing task list")
		return
	}
	if len(taskList) == 0 {
		logger.WithField("amount", len(taskList)).Debug("no processing task found")
		return
	}

	// start processing task
	for _, task := range taskList {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// 2. prepare resources before starting local MPC task
		startRequest, err := t.MpcHandler.TaskStartPrepare(task)
		if err != nil {
			logger.WithError(err).Errorf("error occurred when retry prepare task, and taskId: %s", task.ID)
			continue
		}
		// 3. start local mpc task
		logger.Infof("retry start Processing task, taskId: %s", task.ID)
		if err := t.MpcHandler.StartLocalMpcTask(startRequest, true); err != nil {
			logger.WithError(err).Errorf("error occurred when retry execute task, and taskId: %s", task.ID)
			continue
		}
	}

	logger.WithFields(logrus.Fields{
		"task_len": len(taskList),
		"end_time": time.Now().Format("2006-01-02 15:04:05"),
	}).Info("tasks retry execution finished")
}

// updateTaskExecStatus update an executing task status to Processing in blockchain
func (t *TaskMonitor) updateTaskExecStatus(taskId string) error {
	execTaskOptions := &blockchain.FLTaskExeStatusOptions{
		Executor:    t.PublicKey[:],
		TaskID:      taskId,
		CurrentTime: time.Now().UnixNano(),
	}
	// sign request
	signExecMes := fmt.Sprintf("%x,%s,%d", execTaskOptions.Executor, execTaskOptions.TaskID, execTaskOptions.CurrentTime)
	sig, err := ecdsa.Sign(t.PrivateKey, hash.HashUsingSha256([]byte(signExecMes)))
	if err != nil {
		logger.WithError(err).Errorf("failed to sign exec task options, taskId: %s", taskId)
		return err
	}
	execTaskOptions.Signature = sig[:]

	if err := t.Blockchain.ExecuteTask(execTaskOptions); err != nil {
		logger.WithError(err).Errorf("failed to execute task, taskID: %s", taskId)
		return err
	}
	return nil
}
