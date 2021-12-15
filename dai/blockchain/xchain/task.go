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

package xchain

import (
	"encoding/json"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
)

// PublishTask publishes task on xchain
func (x *XChain) PublishTask(opt *blockchain.PublishFLTaskOptions) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal PublishFLTaskOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "PublishTask"
	if _, err = x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// ListTask lists tasks from xchain
func (x *XChain) ListTask(opt *blockchain.ListFLTaskOptions) (blockchain.FLTasks, error) {
	var ts blockchain.FLTasks

	opts, err := json.Marshal(*opt)
	if err != nil {
		return ts, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal ListFLTaskOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ListTask"
	s, err := x.QueryContract(args, mName)
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
func (x *XChain) GetTaskById(id string) (blockchain.FLTask, error) {
	var t blockchain.FLTask
	args := map[string]string{
		"id": id,
	}
	mName := "GetTaskById"
	s, err := x.QueryContract(args, mName)
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
func (x *XChain) ConfirmTask(opt *blockchain.FLTaskConfirmOptions) error {
	return x.setTaskConfirmStatus(opt, true)
}

// RejectTask is called when Executor rejects task
func (x *XChain) RejectTask(opt *blockchain.FLTaskConfirmOptions) error {
	return x.setTaskConfirmStatus(opt, false)
}

func (x *XChain) setTaskConfirmStatus(opt *blockchain.FLTaskConfirmOptions, isConfirm bool) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal FLTaskConfirmOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "RejectTask"
	if isConfirm {
		mName = "ConfirmTask"
	}
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// StartTask is called when Requester starts task after all Executors confirmed
func (x *XChain) StartTask(id string, sig []byte) error {
	args := map[string]string{
		"taskId":    id,
		"signature": string(sig),
	}
	mName := "StartTask"
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// ExecuteTask is called when Executor run task
func (x *XChain) ExecuteTask(opt *blockchain.FLTaskExeStatusOptions) error {
	return x.setTaskExecuteStatus(opt, false)
}

// FinishTask is called when task execution finished
func (x *XChain) FinishTask(opt *blockchain.FLTaskExeStatusOptions) error {
	return x.setTaskExecuteStatus(opt, true)
}

func (x *XChain) setTaskExecuteStatus(opt *blockchain.FLTaskExeStatusOptions, isFinish bool) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal FLTaskExeStatusOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ExecuteTask"
	if isFinish {
		mName = "FinishTask"
	}
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}
