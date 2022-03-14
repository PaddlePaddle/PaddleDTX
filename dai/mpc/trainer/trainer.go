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
	"strings"
	"sync"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/evaluator"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/learners"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/livaluator"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

var (
	logger = logrus.WithField("module", "mpc.trainer")
)

// Learner is assigned with a specific algorithm and data used for training a model,
//  and participates in the multi-parts-calculation during training process
type Learner interface {
	// Advance does calculation with local data and communicates with other nodes in cluster to train a model step by step
	// payload could be resolved by Learner defined by specific algorithm
	// We'd better call the method asynchronously avoid blocking the main go-routine
	Advance(payload []byte) (*pb.TrainResponse, error)
}

// Evaluator performs model evaluation, supports cross-validation, LOO, validation by proportional random division.
// The basic steps of evaluation:
//  Divide the dataset in some way
//  Train the model
//  Validate
//  Calculate the evaluation metric scores with prediction result obtained on the validation set
//  Calculate the average scores for each metric
type Evaluator interface {
	// Start starts model evaluation, segment the training set according to a certain strategy (cross validation, proportional random division),
	//  then starts the training-validation process.
	// fileRows is returned by psi.IntersectParts after sample alignment.
	Start(fileRows [][]string) error

	// Stop deletes all the leaners created by Evaluator as well as other objects
	Stop()

	// SaveModel collects the results of the training in the evaluation phase, that is, the model.
	// If the model is successfully trained,
	// it will trigger the local creation of a Model instance for validation.
	SaveModel(*pbCom.TrainTaskResult) error

	// SavePredictOut collects the prediction results in the evaluation phase.
	// If the prediction result is obtained, it will check how many prediction results have been obtained so far,
	//  and determine whether to start calculating the average scores for each metric.
	SavePredictOut(*pbCom.PredictTaskResult) error
}

// LiveEvaluator performs staged evaluation during training.
// The basic steps of LiveEvaluator:
//  Divide the dataset in the way of proportional random division.
//  Initiate a learner for evaluation with training part.
//  Train the model, and pause training when the pause round is reached,
//  and instantiate the staged model for validation,
//  then, calculate the evaluation metric scores with prediction result obtained on the validation set.
//  Repeat Train-Pause-validate until the stop signal is received.
type LiveEvaluator interface {
	// Trigger triggers model evaluation.
	// The parameter contains two types of messages.
	// One is to set the learner for evaluation with training set and start it.
	// The other is to drive the learner to continue training. When the conditions are met(reaching pause round),
	// stop training and instantiate the model for validation.
	Trigger(*pb.LiveEvaluationTriggerMsg) error

	// Stop deletes all the leaners created by LiveEvaluator as well as other objects
	Stop()

	// SaveModel collects the results of the training in the evaluation phase,
	// that is, the model, for LiveEvaluation of Model.
	// If the model is successfully trained,
	// it will trigger the local creation of a Model instance for validation.
	SaveModel(*pbCom.TrainTaskResult) error

	// SavePredictOut collects the prediction results in the evaluation phase.
	// If the prediction result is obtained, it will start calculating metric scores,
	// then report the results to visualization system.
	SavePredictOut(*pbCom.PredictTaskResult) error
}

type TrainResponse struct {
	Resp *pb.TrainResponse
	Err  error
}

// RpcHandler performs remote procedure calls to remote cluster nodes.
// set into Trainer instance in initialization phase
type RpcHandler interface {
	StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error)

	// StepTrainWithRetry sends training message to remote mpc-node
	// retries 2 times at most
	// inteSec indicates the interval between retry requests, in seconds
	StepTrainWithRetry(req *pb.TrainRequest, peerName string, times int, inteSec int64) (*pb.TrainResponse, error)
}

// Callback contains some methods would be called when finish training,
// such as to save the trained models and to stop a training task.
// On the other hand, it also contains some other methods would be called during the evaluation phase,
// such as to start a specific task of training or prediction and to train out a model.
// It will be set into Trainer instance in initialization phase.
type Callback interface {
	//SaveModel to persist a model
	SaveModel(*pbCom.TrainTaskResult) error

	// StartTask starts a specific task of training or prediction
	StartTask(*pbCom.StartTaskRequest) error

	//StopTask to stop a training task
	// You'd better use it asynchronously to avoid deadlock
	StopTask(*pbCom.StopTaskRequest) error

	// Train to train out a model
	Train(*pb.TrainRequest) (*pb.TrainResponse, error)
}

// Trainer manages Learners, such as to create or to delete a learner
// dispatches requests to different Learners by taskId,
// keeps the number of Learners in the proper range in order to avoid high memory usage
type Trainer struct {
	learnerLimit   int
	learners       map[string]Learner
	evaluators     sync.Map
	liveEvaluators sync.Map
	trainResults   sync.Map
	rpcHandler     RpcHandler
	callback       Callback
	address        string
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

	// create a learner without samples if the request contains no file, and such request always comes from LiveEvaluator
	// create a common learner with samples if the request contains file, and such request always comes from a user (or an Evaluator)

	var learner Learner
	var errL error
	if len(file) > 0 {
		le := t.newLiveEvaluator(req)
		learner, errL = learners.NewLearner(taskId, t.address, algo, params, file, hosts, t.rpcHandler, t, le)
	} else {
		learner, errL = learners.NewLearnerWithoutSamples(taskId, t.address, algo, params, hosts, t.rpcHandler, t)
	}
	if errL != nil {
		return errL
	}

	t.learners[taskId] = learner
	t.newEvaluator(req)
	logger.WithField("taskId", taskId).Infof("task stored")

	return nil
}

// DeleteLearner deletes a task from Memory Storage
func (t *Trainer) DeleteLearner(req *pbCom.StopTaskRequest) error {
	taskId := req.TaskID
	t.deleteLearner(taskId)

	// If the training task came from LiveEvaluator,
	// didn't create Evaluator or LiveEvaluator for it,
	// and there was no need to store TrainResult for it either.
	fromEvaluator, fromLiveEvaluator, _ := t.checkOrigin(taskId)
	if !fromLiveEvaluator { // not from LiveEvaluator, that's to say, from a user or an Evaluator
		if !fromEvaluator { // neither from any LiveEvaluator nor Evaluator, that's to say, from a user
			// and only the leaner from a user can create Evaluator and store TrainResult
			if e, ok := t.evaluatorExists(taskId); ok {
				e.Stop()
				t.deleteEvaluator(taskId)
			}
			t.deleteTrainResult(taskId)
		}

		if e, ok := t.liveEvaluatorExists(taskId); ok {
			e.Stop()
			t.deleteLiveEvaluator(taskId)
		}
	}

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

// Validate saves the prediction results to the Evaluator or LiveEvaluator,
// then trigger the subsequent verification process.
func (t *Trainer) Validate(req *pb.ValidateRequest, resC chan *TrainResponse) {
	setResult := func(err error) {
		res := &TrainResponse{Err: err}
		resC <- res
		close(resC)
	}

	taskId := req.TaskID
	// check whether prediction is from Evaluator or LiveEvaluator,
	// then save the prediction result
	if req.From == pb.Evaluator_NORMAL {
		if eva, ok := t.evaluatorExists(taskId); ok {
			go func() {
				err := eva.SavePredictOut(req.PredictResult)
				setResult(err)
			}()
		} else {
			err := errorx.New(errcodes.ErrCodeParam, "evaluator related to task [%s] not exists ", taskId)
			setResult(err)
		}
	} else {
		if liveEva, ok := t.liveEvaluatorExists(taskId); ok {
			go func() {
				err := liveEva.SavePredictOut(req.PredictResult)
				setResult(err)
			}()
		} else {
			err := errorx.New(errcodes.ErrCodeParam, "evaluator related to task [%s] not exists ", taskId)
			setResult(err)
		}
	}
}

// SaveResult saves the training result (failed status or successful status) for a Learner
// and stops related task.
// Analyze the TaskID to determine whether the training task is a common task from user
// or a task from Evaluator.
// If the former, and user didn't ask for evaluation, persist the prediction results locally,
//  otherwise call Evaluator.Start() to start evaluation process.
// If the latter, call Evaluator.SaveModel().
func (t *Trainer) SaveResult(result *pbCom.TrainTaskResult) {
	// Only when user requests model evaluation, a corresponding Evaluator will be created,
	// and the Evaluator will not be created again for the training tasks created by Evaluator.
	// So if find a created Evaluator related to the training task, start the evaluation process.
	if eva, ok := t.evaluatorExists(result.TaskID); ok && result.Success {
		// store training result for further use when evaluation is finished
		logger.WithField("taskId", result.TaskID).Info("Start evaluation")
		t.storeTrainResult(result.TaskID, result)
		go func() {
			var ts [][]string
			for _, r := range result.TrainSet {
				ts = append(ts, r.GetRow())
			}
			err := eva.Start(ts)
			// the training set will not be used in subsequent processes,
			// and delete it from training result
			result.TrainSet = []*pbCom.TrainTaskResult_FileRow{}

			// evaluation failed, and only save the training result
			if err != nil {
				logger.WithField("taskId", result.TaskID).Errorf("failed to start evaluation, and error is[%s]", err.Error())
				t.SavePredictAndEvaluatResult(result)
			}
		}()

		return
	}

	stopTask := func() {
		logger.WithField("taskId", result.TaskID).Infof("Stop training task. And delete model[%s] and training result[%t]",
			string(result.Model), result.Success)

		req := &pbCom.StopTaskRequest{
			TaskID: result.TaskID,
			Params: &pbCom.TaskParams{
				TaskType: pbCom.TaskType_LEARN,
			},
		}

		t.callback.StopTask(req)
	}

	fromEvaluator, fromLiveEvaluator, sourceTaskId := t.checkOrigin(result.TaskID)
	if fromEvaluator {
		// If training task is from Evaluator,
		// save training result to Evaluator.

		if eva, ok := t.evaluatorExists(sourceTaskId); ok {
			go func() {
				eva.SaveModel(result)
			}()
		} else {
			logger.WithField("taskId", result.TaskID).Errorf("failed to find evaluator[%s]", sourceTaskId)
		}

		// For training task from Evaluator, it's better to stop the task immediately for resource sake.
		go stopTask()

	} else if fromLiveEvaluator {
		// If training task is from LiveEvaluator,
		// save training result to LiveEvaluator.
		if eva, ok := t.liveEvaluatorExists(sourceTaskId); ok {
			go func() {
				eva.SaveModel(result)
			}()
		} else {
			logger.WithField("taskId", result.TaskID).Errorf("failed to find evaluator[%s]", sourceTaskId)
		}
		// For training task from LiveEvaluator, the result is a staged model,
		// and the task should be stopped after the source task finishing.
	} else {
		// If training task is from user and isn't asked for being evaluated,
		// persist the prediction results locally.

		if err := t.callback.SaveModel(result); err != nil {
			logger.WithField("taskId", result.TaskID).Errorf("failed to save model[%s] and training result[%t], and error is[%s]",
				string(result.Model), result.Success, err.Error())
		}

		// For training task from user, the result has been saved, stop the task.
		go stopTask()
	}
}

// SavePredictAndEvaluatResult saves the training result and evaluation result for a Learner
// and stops related task.
// Called only by Evaluator.
func (t *Trainer) SavePredictAndEvaluatResult(result *pbCom.TrainTaskResult) {
	// add back the previously cached training results,
	// and then store the entire result locally and persistently.
	if trainResStored, ok := t.trainResultExists(result.TaskID); ok {
		result.Model = trainResStored.Model
		result.Success = true

		if err := t.callback.SaveModel(result); err != nil {
			logger.WithField("taskId", result.TaskID).Errorf("failed to save model[%s] and training result[%t], and error is[%s]",
				string(result.Model), result.Success, err.Error())
		}
	} else {
		logger.WithField("taskId", result.TaskID).Error("failed to get stored model.")
	}

	// The result has been saved (but maybe failed), stop the task.
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

// checkOrigin analyzes the TaskID to determine whether the training task is a common task from user
// or a task from Evaluator or LiveEvaluator.
// Return true together with source task id if the training task is from Evaluator or LiveEvaluator, otherwise return false.
func (t *Trainer) checkOrigin(taskId string) (fromEvaluator bool, fromLiveEvaluator bool, sourceTaskId string) {
	// if the training task is from Evaluator, the TaskID conforms such form like `{uuid}_{k}_train_Eva`,
	// and if from LiveEvaluator, the TaskID conforms such form like `{uuid}_{k}_train_LEv`(also could be `{uuid}_{k}_train_Eva_0_train_LEv`),
	// so we just need to check the last 4 letters to determine where the task comes from.
	l := len(taskId)
	if l <= 4 {
		return
	}

	suffix := taskId[len(taskId)-4:]
	if suffix == "_Eva" {
		ss := strings.SplitN(taskId, "_", 2)
		fromEvaluator = true
		sourceTaskId = ss[0]
		return
	} else if suffix == "_LEv" {
		fromLiveEvaluator = true
		sourceTaskId = taskId[0 : len(taskId)-len("_0_train_LEv")]
		return
	}

	return
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

func (t *Trainer) evaluatorExists(taskId string) (Evaluator, bool) {

	if e, ok := t.evaluators.Load(taskId); ok {
		return e.(Evaluator), ok
	} else {
		return nil, false
	}
}

// newEvaluator create an Evaluator if user wants to evaluate the model.
func (t *Trainer) newEvaluator(req *pbCom.StartTaskRequest) {
	taskId := req.TaskID
	if req.Params != nil && req.Params.EvalParams != nil && req.Params.EvalParams.Enable {
		ev, err := evaluator.NewEvaluator(req, t.callback, t)
		if err == nil {
			t.evaluators.LoadOrStore(taskId, ev)
		} else {
			logger.WithField("taskId", taskId).Errorf("failed to create evaluator, and error is[%s]",
				err.Error())
		}
	}
}

// newLiveEvaluator create a LiveEvaluator if user wants to do live evaluation.
func (t *Trainer) newLiveEvaluator(req *pbCom.StartTaskRequest) LiveEvaluator {
	taskId := req.TaskID
	if req.Params != nil && req.Params.LivalParams != nil && req.Params.LivalParams.Enable {
		lev, err := livaluator.NewLiveEvaluator(req, t.callback)
		if err == nil {
			t.liveEvaluators.LoadOrStore(taskId, lev)
			return lev
		} else {
			logger.WithField("taskId", taskId).Errorf("failed to create live evaluator, and error is[%s]",
				err.Error())
			return nil
		}
	}
	return nil
}

func (t *Trainer) deleteEvaluator(taskId string) {
	t.evaluators.Delete(taskId)
}

func (t *Trainer) liveEvaluatorExists(taskId string) (LiveEvaluator, bool) {
	if le, ok := t.liveEvaluators.Load(taskId); ok {
		return le.(LiveEvaluator), ok
	} else {
		return nil, false
	}
}

func (t *Trainer) deleteLiveEvaluator(taskId string) {
	t.liveEvaluators.Delete(taskId)
}

func (t *Trainer) storeTrainResult(taskId string, result *pbCom.TrainTaskResult) {
	t.trainResults.LoadOrStore(taskId, result)
}

func (t *Trainer) trainResultExists(taskId string) (*pbCom.TrainTaskResult, bool) {
	if r, ok := t.trainResults.Load(taskId); ok {
		return r.(*pbCom.TrainTaskResult), ok
	} else {
		return nil, false
	}
}

func (t *Trainer) deleteTrainResult(taskId string) {
	t.trainResults.Delete(taskId)
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
