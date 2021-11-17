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

package trainer

import (
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/learners"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

var (
	logger = logrus.WithField("module", "mpc.trainer")
)

// Learner is assigned with a specific algorithm and data used for training a model
//  participates in the multi-parts-calculation during training process
type Learner interface {
	// Advance does calculation with local data and communicates with other nodes in cluster to train a model step by step
	// payload could be resolved by Learner defined by specific algorithm
	// We'd better call the method asynchronously avoid blocking the main go-routine
	Advance(payload []byte) (*pb.TrainResponse, error)
}

type TrainResponse struct {
	Resp *pb.TrainResponse
	Err  error
}

// RpcHandler performs remote procedure calls to remote cluster nodes.
// set into Trainer instance in initialization phase
type RpcHandler interface {
	StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error)
}

// Callback contains some methods would be called when finish training
// such as to save the trained models and to stop a training task
// set into Trainer instance in initialization phase
type Callback interface {
	//SaveModel to persist a model
	SaveModel(*pbCom.TrainTaskResult) error

	//StopTask to stop a training task
	// You'd better use it asynchronously to avoid deadlock
	StopTask(*pbCom.StopTaskRequest)
}

// Trainer manages Learners, such as to create or to delete a learner
// dispatches requests to different Learners by taskId,
// keeps the number of Learners in the proper range in order to avoid high memory usage
type Trainer struct {
	learnerLimit int
	learners     map[string]Learner
	rpcHandler   RpcHandler
	callback     Callback
	address      string
}

// NewLearner creates a Learner instance related to TaskId and stores it into Memory Storage
// keeps the number of Learners in the proper range in order to avoid high memory usage
func (t *Trainer) NewLearner(req *pbCom.StartTaskRequest) error {

	if t.learnerLimit <= len(t.learners) {
		err := errorx.New(errcodes.ErrCodeTooMuchTasks, "the number of tasks reached upper-limit %d", t.learnerLimit)
		return err
	}

	taskId := req.TaskID
	if _, ok := t.learnerExists(taskId); ok {
		err := errorx.New(errcodes.ErrCodeTaskExists, "task[%s] already exists ", taskId)
		return err
	}

	algo := req.GetParams().GetAlgo()
	params := req.GetParams().GetTrainParams()
	file := req.GetFile()
	hosts := req.GetHosts()
	learner, err := learners.NewLearner(taskId, t.address, algo, params, file, hosts, t.rpcHandler, t)

	if err != nil {
		return err
	}

	t.learners[taskId] = learner

	return nil
}

// DeleteLearner deletes a task from Memory Storage
func (t *Trainer) DeleteLearner(req *pbCom.StopTaskRequest) error {
	taskId := req.TaskID
	t.deleteLearner(taskId)

	logger.WithField("taskId", taskId).Info("task deleted")
	return nil
}

// Train dispatches requests to different Learners by taskId
// resC returns the result, and couldn't be set with nil
func (t *Trainer) Train(req *pb.TrainRequest, resC chan *TrainResponse) {
	setResult := func(resp *pb.TrainResponse, err error) {
		res := &TrainResponse{Resp: resp, Err: err}
		resC <- res
		close(resC)
	}

	taskId := req.TaskID
	mesg := req.GetPayload()
	if learner, ok := t.learnerExists(taskId); ok {
		go func() {
			resp, err := learner.Advance(mesg)
			setResult(resp, err)
		}()
	} else {
		err := errorx.New(errcodes.ErrCodeParam, "task[%s] not exists ", taskId)
		setResult(nil, err)
	}
}

// SaveResult saves the training result (failed status or successful status) for a Learner
// and stops related task
func (t *Trainer) SaveResult(result *pbCom.TrainTaskResult) {
	if err := t.callback.SaveModel(result); err != nil {
		logger.WithField("taskId", result.TaskID).Errorf("failed to save model[%s] and training result[%t], and error is[%s]",
			string(result.Model), result.Success, err.Error())
	}

	logger.WithField("taskId", result.TaskID).Infof("Stop training task. And delete model[%s] and training result[%t]",
		string(result.Model), result.Success)

	req := &pbCom.StopTaskRequest{
		TaskID: result.TaskID,
		Params: &pbCom.TaskParams{
			TaskType: pbCom.TaskType_LEARN,
		},
	}
	go t.callback.StopTask(req)
}

func (t *Trainer) learnerExists(taskId string) (Learner, bool) {
	if l, ok := t.learners[taskId]; ok {
		return l, ok
	} else {
		return nil, false
	}
}

func (t *Trainer) deleteLearner(taskId string) {
	delete(t.learners, taskId)
}

// NewTrainer creates a Trainer instance,
// address indicates local mpc-node address
// learnerLimit indicates the upper limit of the number of Learners
// rh indicates the handler for rpc request sending
// cb indicates the callback methods called when finish training
func NewTrainer(address string, rh RpcHandler, cb Callback, learnerLimit int) *Trainer {
	t := &Trainer{
		learnerLimit: learnerLimit,
		learners:     make(map[string]Learner, learnerLimit),
		rpcHandler:   rh,
		callback:     cb,
		address:      address,
	}

	return t
}
