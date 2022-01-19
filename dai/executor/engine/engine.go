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

package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/dai/config"
	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/handler"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/monitor"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/cluster"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
)

var (
	logger = logrus.WithField("module", "engine")
)

// Engine task processing engine
type Engine struct {
	chain      handler.Blockchain
	node       handler.Node
	storage    handler.FileStorage
	mpcHandler handler.MpcHandler
	monitor    *monitor.TaskMonitor
}

// NewEngine initiates Engine
func NewEngine(conf *config.ExecutorConf) (e *Engine, err error) {
	return initEngine(conf)
}

// Start registers local node to blockchain and starts Monitor
func (e *Engine) Start(ctx context.Context) error {
	// register node
	if err := e.node.Register(e.chain); err != nil {
		return err
	}
	// re-execute tasks in Processing status
	go e.monitor.RetryProcessingTask(ctx)

	// start timed task to find out tasks ready to execute,
	// then starts Multi-Party Computation for each task
	e.monitor.StartTaskLoopRequest(ctx)

	return nil
}

// GetMpcService returns mpc service to be registered to grpcServer
func (e *Engine) GetMpcService() *cluster.Service {
	return e.mpcHandler.GetMpcClusterService()
}

// ListTask lists tasks from blockchain by requester or executor's Public Key
func (e *Engine) ListTask(ctx context.Context, in *pbTask.ListTaskRequest) (*pbTask.FLTasks, error) {
	listOptions := &blockchain.ListFLTaskOptions{
		PubKey:    in.PubKey,
		Status:    in.Status,
		TimeStart: in.TimeStart,
		TimeEnd:   in.TimeEnd,
		Limit:     in.Limit,
	}
	// invoke contract to list tasks
	fts, err := e.chain.ListTask(listOptions)
	if err != nil {
		return &pbTask.FLTasks{}, errorx.Wrap(err, "failed list task")
	}
	// traverse tasks
	resp := &pbTask.FLTasks{}
	for _, ft := range fts {
		resp.FLTasks = append(resp.FLTasks, ft)
	}
	return resp, nil
}

// GetTaskById queries task details by taskID
func (e *Engine) GetTaskById(ctx context.Context, in *pbTask.GetTaskRequest) (*pbTask.FLTask, error) {
	// get task detail
	task, err := e.chain.GetTaskById(in.TaskID)
	if err != nil {
		return &pbTask.FLTask{}, errorx.Wrap(err, "failed get task by id")
	}
	return task, nil
}

// GetPredictResult checks task's initiator and gets prediction result from Xuper db.
//  in.PubKey must matches task.Requester, only task.Requester can get prediction result.
func (e *Engine) GetPredictResult(ctx context.Context, in *pbTask.TaskRequest) (*pbTask.PredictResponse, error) {
	// get task detail
	task, err := e.chain.GetTaskById(in.TaskID)
	if err != nil {
		return &pbTask.PredictResponse{}, errorx.Wrap(err, "failed get predict result")
	}
	// check task type
	if task.AlgoParam.TaskType != pbCom.TaskType_PREDICT {
		return &pbTask.PredictResponse{}, errorx.New(errorx.ErrCodeParam, "illegal taskId, not a predict task")
	}
	if !bytes.Equal(task.Requester, in.PubKey) {
		return &pbTask.PredictResponse{}, errorx.New(errorx.ErrCodeParam, "public key is invalid")
	}
	// check signature
	m := fmt.Sprintf("%x,%s", in.PubKey, in.TaskID)
	if err := e.checkSign(in.Signature, in.PubKey, []byte(m)); err != nil {
		return &pbTask.PredictResponse{}, errorx.Wrap(err, "get predict result failed")
	}

	// get prediction result from XuperDB or LocalPath, if the prediction result is stored
	// on XuperDB, task.Result is fileId, else get the file from LocalPath by prediction task's ID.
	predictFileName := task.Result
	if predictFileName == "" {
		predictFileName = in.TaskID
	}
	r, err := e.storage.PredictStorage.Read(predictFileName)
	if err != nil {
		return &pbTask.PredictResponse{}, errorx.Wrap(err, "failed to get reader from xuperdb")
	}
	text, err := ioutil.ReadAll(r)
	if err != nil {
		return &pbTask.PredictResponse{}, errorx.Wrap(err, "failed to read file from xuperdb")
	}
	defer r.Close()

	// format result
	rows, err := vl_common.PredictResultFromBytes(text)
	if err != nil {
		return &pbTask.PredictResponse{}, errorx.Wrap(err, "failed to read file from xuperdb")
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return &pbTask.PredictResponse{}, errorx.New(errorx.ErrCodeParam, "illegal taskId, not a predict task")
	}

	return &pbTask.PredictResponse{
		TaskID:  task.ID,
		Payload: payload,
	}, nil
}

// StartTask starts mpc-training or mpc-prediction after received "task starting" message from remote executor
func (e *Engine) StartTask(ctx context.Context, in *pbTask.TaskRequest) (*pbTask.TaskResponse, error) {
	logger.Debugf("got StartTaskRequest: %v", in)
	// get task detail
	task, err := e.chain.GetTaskById(in.TaskID)
	if err != nil {
		return &pbTask.TaskResponse{}, errorx.Wrap(err, "get task from chain error")
	}
	// check task status
	if task.Status != blockchain.TaskProcessing {
		return &pbTask.TaskResponse{}, errorx.Wrap(err, "illegal task status")
	}

	// check sign
	m := fmt.Sprintf("%x,%s", in.PubKey, in.TaskID)
	if err := e.checkSign(in.Signature, in.PubKey, []byte(m)); err != nil {
		return &pbTask.TaskResponse{}, errorx.Wrap(err, "start task failed, signature error")
	}
	// make sure the request must come from executor who confirmed the task
	isExecutorNodeExist := false
	for _, ds := range task.DataSets {
		if bytes.Equal(ds.Executor, in.PubKey) {
			isExecutorNodeExist = true
			break
		}
	}
	if !isExecutorNodeExist {
		return &pbTask.TaskResponse{}, errorx.New(errcodes.ErrCodeParam, "wrong request source[%x]", in.PubKey)
	}

	// prepare resources before start mpc
	startRequest, err := e.mpcHandler.TaskStartPrepare(task)
	if err != nil {
		if code, _ := errorx.Parse(err); code == errcodes.ErrCodeTaskExists {
			logger.Info("Local mpc task already start")
			return &pbTask.TaskResponse{
				TaskID: in.TaskID,
			}, nil
		}
		logger.WithError(err).Error("failed to start local mpc, task start preparation error")
		return &pbTask.TaskResponse{}, errorx.Wrap(err, "task start prepare error")
	}

	// start local mpc
	go func() {
		if err := e.mpcHandler.StartLocalMpcTask(startRequest, false); err != nil {
			logger.WithError(err).Errorf("failed to start local mpc , taskId: %s", task.ID)
		}
	}()

	logger.Info("Start local mpc successfully after receive task starting signal")

	return &pbTask.TaskResponse{
		TaskID: in.TaskID,
	}, nil
}

// checkSign checks signature
func (e *Engine) checkSign(sign, owner, mes []byte) (err error) {
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

// Close waits until all inner services stop
func (e *Engine) Close() {
	if e.monitor != nil {
		e.monitor.StopLoopReq()
		e.monitor.StopRetryReq()
	}
	if e.mpcHandler != nil {
		e.mpcHandler.Close()
	}
	if e.chain != nil {
		e.chain.Close()
	}
}
