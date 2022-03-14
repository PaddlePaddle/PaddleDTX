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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	xdbchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/handler"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
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

		// query confirming tasks and publish file authorization applications,
		// if the file authorization application has been passed, confirm the task
		if err := t.getUnconfirmedTaskAndConfirm(); err != nil {
			logger.WithError(err).Error("failed to find confirming tasks to confirm")
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

// getUnconfirmedTaskAndConfirm query confirming tasks that need to be confirmed by the
// executor node from chain, check whether the executor node has permission to use the sample file,
// if not, publishes a file authorization application, otherwise confrims or rejects the task
func (t *TaskMonitor) getUnconfirmedTaskAndConfirm() error {
	// 1. find all confirming tasks from chain
	taskList, err := t.Blockchain.ListTask(&blockchain.ListFLTaskOptions{
		PubKey:    t.PublicKey[:],
		Status:    blockchain.TaskConfirming,
		TimeStart: 0,
		TimeEnd:   time.Now().UnixNano(),
		Limit:     blockchain.TaskListMaxNum,
	})
	if err != nil {
		return errorx.Wrap(err, "failed to find Confirming task list")
	}
	if len(taskList) == 0 {
		logger.WithField("amount", len(taskList)).Debug("no Confirming task found")
		return nil
	}
	// 2. confirm tasks by the executor node's ExecutionType
	for _, task := range taskList {
		for _, ds := range task.DataSets {
			if bytes.Equal(ds.Executor, t.PublicKey[:]) {
				if err := t.confirmTaskByExecutionType(task.ID, ds); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// confirmTaskByExecutionType confrims tasks by ExecutionType.
// If t.ExecutionType is "Self", means the dataOwner node has authorized sample files to the executor
// node, the executor node can directly confirm tasks. if t.ExecutionType is "Proxy",
// the executor node confirms or rejects tasks by the file authorization application.
func (t *TaskMonitor) confirmTaskByExecutionType(taskID string, ds *pbTask.DataForTask) error {
	if t.ExecutionType == handler.SelfExecutionMode {
		if err := t.confirmTaskOnChain(taskID, "", true); err != nil {
			return errorx.Wrap(err, "confirm task failed, taskID: %s, ExecutionType: %s, Executor: %x",
				taskID, t.ExecutionType, t.PublicKey[:])
		}
	} else {
		// 1. query the list of file authorization applications
		// applier is the executor's public key, authorizer is the file owner's public key
		currentTime := time.Now().UnixNano()
		fileAuths, err := t.Blockchain.ListFileAuthApplications(&xdbchain.ListFileAuthOptions{
			Applier:    t.PublicKey[:],
			Authorizer: ds.Owner,
			FileID:     ds.DataID,
			TimeStart:  0,
			TimeEnd:    currentTime,
			Limit:      1,
		})
		if err != nil {
			return errorx.Wrap(err, "failed to find the file authorization application, fileID: %s, Applier: %x, Authorizer: %x",
				ds.DataID, t.PublicKey[:], ds.Owner)
		}
		// 2. if the authorization application has not been published or the authorization application has expired,
		// then publish the file authorization application
		if len(fileAuths) == 0 || (fileAuths[0].Status == xdbchain.FileAuthApproved &&
			fileAuths[0].ExpireTime <= currentTime) {
			if err := t.publishFileAuthApplication(ds.DataID, taskID, ds.Owner); err != nil {
				return err
			}
		} else {
			// 3. if the authorization application is rejected, rejected the task
			if fileAuths[0].Status == xdbchain.FileAuthRejected {
				rejectReason := fmt.Sprintf("File authorization application is refused, authID: %s, reason: %s",
					fileAuths[0].ID, fileAuths[0].RejectReason)
				if err := t.confirmTaskOnChain(taskID, rejectReason, false); err != nil {
					return errorx.Wrap(err, "reject task failed, taskID: %s, Executor: %x", taskID, t.PublicKey[:])
				}
			} else if fileAuths[0].Status == xdbchain.FileAuthApproved && fileAuths[0].ExpireTime > currentTime {
				// if the authorization application has been passed and has not expired, then confirm the task
				if err := t.confirmTaskOnChain(taskID, "", true); err != nil {
					return errorx.Wrap(err, "confirm task failed, taskID: %s, Executor: %x", taskID, t.PublicKey[:])
				}
			} else {
				logger.Infof("the file authorization application is Unapproved, fileAuthID: %s, taskID: %s", fileAuths[0].ID, taskID)
			}
		}
	}
	return nil
}

// confirmTask after the file owner confirms or rejects the executor's file authorization application
// then the executor node confirms or rejects the task
func (t *TaskMonitor) confirmTaskOnChain(taskID, rejectReason string, isConfirm bool) error {
	currentTime := time.Now().UnixNano()
	confirmOptions := &blockchain.FLTaskConfirmOptions{
		Pubkey:       t.PublicKey[:],
		TaskID:       taskID,
		CurrentTime:  currentTime,
		RejectReason: rejectReason,
	}
	m := fmt.Sprintf("%x,%s,%s,%d", t.PublicKey[:], taskID, rejectReason, currentTime)

	sig, err := ecdsa.Sign(t.PrivateKey, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign confirm fl task")
	}

	// invoke contract to confirm or reject the task
	confirmOptions.Signature = sig[:]
	if isConfirm {
		if err := t.Blockchain.ConfirmTask(confirmOptions); err != nil {
			if code, _ := errorx.Parse(err); code == errorx.ErrCodeAlreadyUpdate {
				logger.Debugf("task already confirmed, taskID: %s, Executor: %x", taskID, t.PublicKey[:])
				return nil
			}
			return err
		}
		logger.Infof("confrims the task successfully, taskID: %s", taskID)
	} else {
		if err := t.Blockchain.RejectTask(confirmOptions); err != nil {
			if code, _ := errorx.Parse(err); code == errorx.ErrCodeAlreadyUpdate {
				logger.Debugf("task already rejected, taskID: %s, Executor: %x", taskID, t.PublicKey[:])
				return nil
			}
			return err
		}
		logger.Infof("rejects the task successfully, taskID: %s, rejectReason: %s", taskID, rejectReason)
	}
	return nil
}

// publishFileAuthApplication the executor node publishes a file authorization application,
// when the file owner confirms the file authorization application, the executor node can confirm the task
func (t *TaskMonitor) publishFileAuthApplication(fileID, taskID string, fileOwner []byte) error {
	// generate a uuid as fileAuthID
	fileAuthUuid, err := uuid.NewRandom()
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to generate fileAuthID")
	}
	opt := xdbchain.PublishFileAuthOptions{
		FileAuthApplication: xdbchain.FileAuthApplication{
			ID:          fileAuthUuid.String(),
			FileID:      fileID,
			Name:        fmt.Sprintf("TaskID-%s", taskID),
			Description: "Used for the executor node to training or predicting task",
			CreateTime:  time.Now().UnixNano(),
			Applier:     t.PublicKey[:],
			Authorizer:  fileOwner,
		},
	}
	// sign file authorization application
	s, err := json.Marshal(opt.FileAuthApplication)
	if err != nil {
		return errorx.Wrap(err, "failed to marshal file authorization application")
	}
	sig, err := ecdsa.Sign(t.PrivateKey, hash.HashUsingSha256(s))
	if err != nil {
		return errorx.Wrap(err, "failed to sign file authorization application")
	}
	opt.Signature = sig[:]
	// publish file authorization application into chain
	if err := t.Blockchain.PublishFileAuthApplication(&opt); err != nil {
		return errorx.Wrap(err, "failed to publish file authorization application")
	}
	return nil
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
