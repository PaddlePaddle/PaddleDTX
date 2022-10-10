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

package fabric

import (
	"encoding/json"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
)

// PublishTask publishes task on xchain
func (f *Fabric) PublishTask(opt *blockchain.PublishFLTaskOptions) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal PublishFLTaskOptions")
	}

	mName := "PublishTask"
	if _, err = f.InvokeContract([][]byte{opts}, mName); err != nil {
		return err
	}
	return nil
}

// ListTask lists tasks from xchain
func (f *Fabric) ListTask(opt *blockchain.ListFLTaskOptions) (blockchain.FLTasks, error) {
	var ts blockchain.FLTasks

	opts, err := json.Marshal(*opt)
	if err != nil {
		return ts, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal ListFLTaskOptions")
	}

	mName := "ListTask"
	s, err := f.QueryContract([][]byte{opts}, mName)
	if err != nil {
		return ts, err
	}
	if err = json.Unmarshal(s, &ts); err != nil {
		return ts, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal FLTasks")
	}
	return ts, nil
}

// GetTaskById gets task by id
func (f *Fabric) GetTaskById(id string) (blockchain.FLTask, error) {
	var t blockchain.FLTask
	mName := "GetTaskById"
	s, err := f.QueryContract([][]byte{[]byte(id)}, mName)
	if err != nil {
		return t, err
	}

	if err = json.Unmarshal([]byte(s), &t); err != nil {
		return t, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal File")
	}
	return t, nil
}

// ConfirmTask is called when Executor confirms task
func (f *Fabric) ConfirmTask(opt *blockchain.FLTaskConfirmOptions) error {
	return f.setTaskConfirmStatus(opt, true)
}

// RejectTask is called when Executor rejects task
func (f *Fabric) RejectTask(opt *blockchain.FLTaskConfirmOptions) error {
	return f.setTaskConfirmStatus(opt, false)
}

// setTaskConfirmStatus is used by the Executor to confirm or reject the task
// if isConfirm is false, update the task status from 'Confirming' to 'Rejected'
func (f *Fabric) setTaskConfirmStatus(opt *blockchain.FLTaskConfirmOptions, isConfirm bool) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal FLTaskConfirmOptions")
	}
	mName := "RejectTask"
	if isConfirm {
		mName = "ConfirmTask"
	}
	if _, err := f.InvokeContract([][]byte{opts}, mName); err != nil {
		return err
	}
	return nil
}

// StartTask is called when Requester starts task after all Executors confirmed
func (f *Fabric) StartTask(opt *blockchain.StartFLTaskOptions) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal StartFLTaskOptions")
	}
	mName := "StartTask"
	if _, err := f.InvokeContract([][]byte{opts}, mName); err != nil {
		return err
	}
	return nil
}

// ExecuteTask is called when Executor run task
func (f *Fabric) ExecuteTask(opt *blockchain.FLTaskExeStatusOptions) error {
	return f.setTaskExecuteStatus(opt, false)
}

// FinishTask is called when task execution finished
func (f *Fabric) FinishTask(opt *blockchain.FLTaskExeStatusOptions) error {
	return f.setTaskExecuteStatus(opt, true)
}

// setTaskExecuteStatus updates task status when the Executor starts running the task or finished the task
// call the contract's 'ExecuteTask' or 'FinishTask' method
func (f *Fabric) setTaskExecuteStatus(opt *blockchain.FLTaskExeStatusOptions, isFinish bool) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal FLTaskExeStatusOptions")
	}
	mName := "ExecuteTask"
	if isFinish {
		mName = "FinishTask"
	}
	if _, err := f.InvokeContract([][]byte{opts}, mName); err != nil {
		return err
	}
	return nil
}
