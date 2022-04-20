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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
)

// PublishTask publishes task
func (x *Xdata) PublishTask(ctx code.Context) code.Response {
	var opt blockchain.PublishFLTaskOptions
	// get opt
	p, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	if err := json.Unmarshal(p, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal PublishFLTaskOptions"))
	}
	// get fltask
	t := opt.FLTask
	// marshal fltask
	s, err := json.Marshal(t)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal FLTask"))
	}
	if err := x.checkSign(opt.Signature, t.Requester, s); err != nil {
		return code.Error(err)
	}

	t.Status = blockchain.TaskConfirming
	// marshal fltask
	s, err = json.Marshal(t)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal FLTask"))
	}

	// put index-fltask on xchain, judge if index exists
	index := packFlTaskIndex(t.ID)
	if _, err := ctx.GetObject([]byte(index)); err == nil {
		return code.Error(errorx.New(errorx.ErrCodeAlreadyExists,
			"duplicated taskID"))
	}
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"fail to put index-flTask on xchain"))
	}

	// put requester listIndex-fltask on xchain
	index = packFlTaskListIndex(t)
	if err := ctx.PutObject([]byte(index), []byte(t.ID)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"fail to put requester listIndex-fltask on xchain"))
	}
	// put executor listIndex-fltask on xchain
	for _, ds := range t.DataSets {
		index := packExecutorTaskListIndex(ds.Executor, t)
		if err := ctx.PutObject([]byte(index), []byte(t.ID)); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
				"fail to put executor listIndex-fltask on xchain"))
		}
	}
	return code.OK([]byte("added"))
}

// ListTask lists tasks
func (x *Xdata) ListTask(ctx code.Context) code.Response {
	// get opt
	p, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.ListFLTaskOptions
	if err := json.Unmarshal(p, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal ListFLTaskOptions"))
	}

	var tasks blockchain.FLTasks

	// get fltasks by list_prefix
	prefix := packFlTaskFilter(opt.PubKey)
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()
	for iter.Next() {
		if opt.Limit > 0 && int64(len(tasks)) >= opt.Limit {
			break
		}
		t, err := x.getTaskById(ctx, string(iter.Value()))
		if err != nil {
			return code.Error(err)
		}
		if t.PublishTime < opt.TimeStart || (opt.TimeEnd > 0 && t.PublishTime > opt.TimeEnd) ||
			(opt.Status != "" && t.Status != opt.Status) {
			continue
		}
		tasks = append(tasks, t)
	}
	// marshal tasks
	s, err := json.Marshal(tasks)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal tasks"))
	}
	return code.OK(s)
}

// GetTaskById gets task by id
func (x *Xdata) GetTaskById(ctx code.Context) code.Response {
	taskID, ok := ctx.Args()["id"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}

	// get fltask by index
	index := packFlTaskIndex(string(taskID))
	s, err := ctx.GetObject([]byte(index))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "task not found"))
	}
	return code.OK(s)
}

// ConfirmTask is called when Executor confirms task
func (x *Xdata) ConfirmTask(ctx code.Context) code.Response {
	return x.setTaskConfirmStatus(ctx, true)
}

// RejectTask is called when Executor rejects task
func (x *Xdata) RejectTask(ctx code.Context) code.Response {
	return x.setTaskConfirmStatus(ctx, false)
}

// setTaskConfirmStatus sets task status as Confirmed or Rejected
func (x *Xdata) setTaskConfirmStatus(ctx code.Context, isConfirm bool) code.Response {
	var opt blockchain.FLTaskConfirmOptions
	// get opt
	p, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	if err := json.Unmarshal(p, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal FLTaskConfirmOptions"))
	}
	t, err := x.getTaskById(ctx, opt.TaskID)
	if err != nil {
		return code.Error(err)
	}
	// executor validity check
	if ok := x.checkExecutor(opt.Pubkey, t.DataSets); !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "bad param: executor"))
	}
	// verify sig
	m := fmt.Sprintf("%x,%s,%s,%d", opt.Pubkey, opt.TaskID, opt.RejectReason, opt.CurrentTime)
	if err := x.checkSign(opt.Signature, opt.Pubkey, []byte(m)); err != nil {
		return code.Error(err)
	}

	// check status
	if t.Status != blockchain.TaskConfirming {
		return code.Error(errorx.New(errorx.ErrCodeParam,
			"confirm task error, taskStatus is not Confirming, taskId: %s, taskStatus: %s", t.ID, t.Status))
	}
	isAllConfirm := true
	for index, ds := range t.DataSets {
		if bytes.Equal(ds.Executor, opt.Pubkey) {
			// judge sample file exists
			if _, err := ctx.GetObject([]byte(ds.DataID)); err != nil {
				return code.Error(errorx.New(errorx.ErrCodeParam, "bad param:taskId, dataId not exist"))
			}
			// judge task is confirmed
			if ds.ConfirmedAt > 0 || ds.RejectedAt > 0 {
				return code.Error(errorx.New(errorx.ErrCodeAlreadyUpdate, "bad param:taskId, task already confirmed"))
			}
			if isConfirm {
				t.DataSets[index].ConfirmedAt = opt.CurrentTime
			} else {
				t.DataSets[index].RejectedAt = opt.CurrentTime
			}
		} else {
			if ds.ConfirmedAt == 0 {
				isAllConfirm = false
			}
		}
	}
	// if all executor nodes confirmed task, task status is ready
	if isAllConfirm {
		t.Status = blockchain.TaskReady
	}
	// if one of executor nodes rejected task, task status is rejected
	if !isConfirm {
		t.Status = blockchain.TaskRejected
		t.ErrMessage = opt.RejectReason
	}
	s, err := json.Marshal(t)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "fail to marshal FLTask"))
	}
	// update index-fltask on xchain
	index := packFlTaskIndex(t.ID)
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"fail to confirm index-flTask on xchain"))
	}
	return code.OK([]byte("OK"))
}

// StartTask is called when Requester starts task after Executors confirmed
func (x *Xdata) StartTask(ctx code.Context) code.Response {
	// get taskId
	taskId, ok := ctx.Args()["taskId"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:taskId"))
	}
	t, err := x.getTaskById(ctx, string(taskId))
	if err != nil {
		return code.Error(err)
	}
	// get signature
	signature, ok := ctx.Args()["signature"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:signature"))
	}
	mes := fmt.Sprintf("%s,%x", string(taskId), t.Requester)
	if err := x.checkSign(signature, t.Requester, []byte(mes)); err != nil {
		return code.Error(err)
	}
	if t.Status != blockchain.TaskReady && t.Status != blockchain.TaskFailed {
		return code.Error(errorx.New(errorx.ErrCodeParam,
			"start task error, task status is not Ready or Failed, taskId: %s, taskStatus: %s", t.ID, t.Status))
	}
	// update task status
	t.Status = blockchain.TaskToProcess
	s, err := json.Marshal(t)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "fail to marshal FLTask"))
	}
	// update index-fltask on xchain
	index := packFlTaskIndex(t.ID)
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"fail to start index-flTask on xchain"))
	}
	return code.OK([]byte("OK"))
}

// ExecuteTask is called when Executor run task
func (x *Xdata) ExecuteTask(ctx code.Context) code.Response {
	return x.setTaskExecuteStatus(ctx, false)
}

// FinishTask is called when task execution finished
func (x *Xdata) FinishTask(ctx code.Context) code.Response {
	return x.setTaskExecuteStatus(ctx, true)
}

func (x *Xdata) setTaskExecuteStatus(ctx code.Context, isFinish bool) code.Response {
	var opt blockchain.FLTaskExeStatusOptions
	// get opt
	p, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	if err := json.Unmarshal(p, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal FLTaskExeStatusOptions"))
	}
	t, err := x.getTaskById(ctx, opt.TaskID)
	if err != nil {
		return code.Error(err)
	}
	// executor validity check
	if ok := x.checkExecutor(opt.Executor, t.DataSets); !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "bad param:executor"))
	}
	// verify sig
	m := fmt.Sprintf("%x,%s,%d", opt.Executor, opt.TaskID, opt.CurrentTime)
	if isFinish {
		m += fmt.Sprintf("%s,%x", opt.ErrMessage, opt.Result)
	}
	if err := x.checkSign(opt.Signature, opt.Executor, []byte(m)); err != nil {
		return code.Error(err)
	}

	if isFinish {
		if t.Status != blockchain.TaskProcessing {
			return code.Error(errorx.New(errorx.ErrCodeParam,
				"finish task error, task status is not Processing, taskId: %s, taskStatus: %s", t.ID, t.Status))
		}

		t.Status = blockchain.TaskFinished
		t.EndTime = opt.CurrentTime
		t.Result = opt.Result

		if opt.ErrMessage != "" {
			t.Status = blockchain.TaskFailed
			t.ErrMessage = opt.ErrMessage
		}
	} else {
		if t.Status != blockchain.TaskToProcess {
			return code.Error(errorx.New(errorx.ErrCodeParam,
				"execute task error, task status is not ToProcess, taskId: %s, taskStatus: %s", t.ID, t.Status))
		}
		t.Status = blockchain.TaskProcessing
		t.StartTime = opt.CurrentTime
	}

	// update task status
	s, err := json.Marshal(t)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "fail to marshal FLTask"))
	}
	// update index-fltask on xchain
	index := packFlTaskIndex(t.ID)
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"fail to set task execute status on xchain"))
	}
	return code.OK([]byte("OK"))
}

func (x *Xdata) getTaskById(ctx code.Context, taskID string) (t blockchain.FLTask, err error) {
	index := packFlTaskIndex(taskID)
	s, err := ctx.GetObject([]byte(index))
	if err != nil {
		return t, errorx.NewCode(err, errorx.ErrCodeNotFound,
			"the task[%s] not found", taskID)
	}

	if err = json.Unmarshal([]byte(s), &t); err != nil {
		return t, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal FlTask")
	}
	return t, nil
}

func (x *Xdata) checkSign(sign, owner, mes []byte) (err error) {
	// verify sig
	if len(sign) != ecdsa.SignatureLength {
		return errorx.New(errorx.ErrCodeParam, "bad param:signature")
	}
	var pubkey [ecdsa.PublicKeyLength]byte
	var sig [ecdsa.SignatureLength]byte
	copy(pubkey[:], owner)
	copy(sig[:], sign)
	if err := ecdsa.Verify(pubkey, hash.HashUsingSha256(mes), sig); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeBadSignature, "failed to verify signature")
	}
	return nil
}

func (x *Xdata) checkExecutor(executor []byte, dataSets []*pbTask.DataForTask) bool {
	for _, ds := range dataSets {
		if bytes.Equal(ds.Executor, executor) {
			return true
		}
	}
	return false
}
