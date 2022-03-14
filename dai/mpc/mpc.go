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

package mpc

import (
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/cluster"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/predictor"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/trainer"
	"github.com/PaddlePaddle/PaddleDTX/dai/p2p"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

type Mpc interface {
	// Train to train out a model
	Train(*pb.TrainRequest) (*pb.TrainResponse, error)

	// Predict to do prediction
	Predict(*pb.PredictRequest) (*pb.PredictResponse, error)

	// StartTask starts a specific task of training or prediction
	StartTask(*pbCom.StartTaskRequest) error

	// StopTask stops a specific task of training or prediction
	StopTask(*pbCom.StopTaskRequest) error

	// Validate writes the prediction results to the Evaluator or LiveEvaluator,
	//  then trigger the subsequent verification process.
	Validate(*pb.ValidateRequest) error

	// Stop performs any necessary termination of the node
	Stop()
}

// P2P used by local Learners to communicate with remote ones on other nodes
// set into mpc instance in initialization phase
type P2P interface {
	GetPeer(address string) (*p2p.Peer, error)
	FreePeer()
}

// Trainer manages Learners, such as to create or to delete a learner
// dispatches requests to different Learners by taskId,
// keeps the number of Learners in the proper range in order to avoid high memory usage
type Trainer interface {
	// NewLearner creates a Learner related to TaskId
	NewLearner(*pbCom.StartTaskRequest) error

	// DeleteLearner deletes a Learner
	DeleteLearner(*pbCom.StopTaskRequest) error

	// Train to dispatch requests to different Learners by taskId during training processes
	// Response channel returns the result, and couldn't be set with nil
	Train(*pb.TrainRequest, chan *trainer.TrainResponse)

	// Validate saves the prediction results to the Evaluator or LiveEvaluator,
	// then trigger the subsequent verification process.
	Validate(*pb.ValidateRequest, chan *trainer.TrainResponse)
}

// Predictor manages Models, such as to create or to delete a model
// dispatches requests to different Models by taskId,
// keeps the number of Models in the proper range in order to avoid high memory usage
type Predictor interface {
	// NewModel creates a Model instance related to TaskId
	NewModel(*pbCom.StartTaskRequest) error
	// DeleteModel deletes a model
	DeleteModel(*pbCom.StopTaskRequest) error

	// Predict dispatches requests to different Models by taskId during prediction processes
	// Response channel returns the result, and couldn't be set with nil
	Predict(*pb.PredictRequest, chan *predictor.PredictResponse)
}

type trainRequest struct {
	startRequest *pbCom.StartTaskRequest // request to start a new task
	stopRequest  *pbCom.StopTaskRequest  // request to end a task
	trainRequest *pb.TrainRequest        // request during training process
	validRequest *pb.ValidateRequest     // request to trigger validation
	responseC    chan *trainer.TrainResponse
}

type predictRequest struct {
	startRequest   *pbCom.StartTaskRequest // request to start a new task
	stopRequest    *pbCom.StopTaskRequest  // request to end a task
	predictRequest *pb.PredictRequest      // request during prediction process
	responseC      chan *predictor.PredictResponse
}

// mpc is the implementation of the Mpc interface
type mpc struct {
	stopC     chan struct{}       // Signal to goroutines that the mpc-node is halting
	doneC     chan struct{}       // Closes when the mpc-node is stopped
	trainC    chan trainRequest   // Signal to handle training request
	predictC  chan predictRequest // Signal to handle prediction request
	trainer   Trainer
	predictor Predictor
}

// Train handles all kinds of message from another node in the cluster during training process
// to train out a model
func (m *mpc) Train(req *pb.TrainRequest) (*pb.TrainResponse, error) {
	if err := m.isRunning(); err != nil {
		return nil, err
	}

	var tarinResp *trainer.TrainResponse
	respC := make(chan *trainer.TrainResponse, 1)
	select {
	case m.trainC <- trainRequest{trainRequest: req, responseC: respC}:
		tarinResp = <-respC
		if tarinResp != nil && tarinResp.Err != nil {
			return nil, tarinResp.Err
		}
	case <-m.doneC:
		return nil, errorx.New(errcodes.ErrCodeNotFound, "mpc is stopped")
	}

	if tarinResp == nil || tarinResp.Resp == nil {
		return &pb.TrainResponse{TaskID: req.TaskID}, nil
	}

	if "" == tarinResp.Resp.TaskID {
		tarinResp.Resp.TaskID = req.TaskID
	}
	return tarinResp.Resp, nil
}

// Predict handles all kinds of message from another node in the cluster during predicting process
// to make out a final result
func (m *mpc) Predict(req *pb.PredictRequest) (*pb.PredictResponse, error) {
	if err := m.isRunning(); err != nil {
		return nil, err
	}

	var predicResp *predictor.PredictResponse
	respC := make(chan *predictor.PredictResponse, 1)
	select {
	case m.predictC <- predictRequest{predictRequest: req, responseC: respC}:
		predicResp = <-respC
		if predicResp != nil && predicResp.Err != nil {
			return nil, predicResp.Err
		}
	case <-m.doneC:
		return nil, errorx.New(errcodes.ErrCodeNotFound, "mpc is stopped")
	}

	if predicResp == nil {
		return &pb.PredictResponse{TaskID: req.TaskID}, nil
	}

	if "" == predicResp.Resp.TaskID {
		predicResp.Resp.TaskID = req.TaskID
	}
	return predicResp.Resp, nil

}

// Validate 保存预测结果触发验证流程
func (m *mpc) Validate(req *pb.ValidateRequest) error {
	if err := m.isRunning(); err != nil {
		return err
	}

	var tarinResp *trainer.TrainResponse
	respC := make(chan *trainer.TrainResponse, 1)
	select {
	case m.trainC <- trainRequest{validRequest: req, responseC: respC}:
		tarinResp = <-respC
		if tarinResp != nil && tarinResp.Err != nil {
			return tarinResp.Err
		}
	case <-m.doneC:
		return errorx.New(errcodes.ErrCodeNotFound, "mpc is stopped")
	}
	return nil
}

// Stop performs any necessary termination of the Mpc-node.
func (m *mpc) Stop() {
	select {
	case m.stopC <- struct{}{}:
		// not stopped yet, so trigger it
	case <-m.doneC:
		// Mpc-node has already been stopped - no need to do anything
		return
	}
	// Block until the stop has been acknowledged by run()
	<-m.doneC
}

// StartTask starts a specific task of training or prediction
func (m *mpc) StartTask(req *pbCom.StartTaskRequest) error {
	if err := m.isRunning(); err != nil {
		return err
	}

	tType := req.GetParams().GetTaskType()
	if pbCom.TaskType_LEARN == tType {
		respC := make(chan *trainer.TrainResponse, 1)
		select {
		case m.trainC <- trainRequest{startRequest: req, responseC: respC}:
			trainResp := <-respC
			if trainResp != nil && trainResp.Err != nil {
				return trainResp.Err
			}
		case <-m.doneC:
			return errorx.New(errcodes.ErrCodeNotFound, "mpc is stopped")
		}
	} else if pbCom.TaskType_PREDICT == tType {
		respC := make(chan *predictor.PredictResponse, 1)
		select {
		case m.predictC <- predictRequest{startRequest: req, responseC: respC}:
			predictResp := <-respC
			if predictResp != nil && predictResp.Err != nil {
				return predictResp.Err
			}
		case <-m.doneC:
			return errorx.New(errcodes.ErrCodeNotFound, "mpc is stopped")
		}

	} else {
		return errorx.New(errcodes.ErrCodeParam, "invalid TaskType %s", tType)
	}
	return nil
}

// StopTask stops a specific task of training or prediction.
func (m *mpc) StopTask(req *pbCom.StopTaskRequest) error {
	if err := m.isRunning(); err != nil {
		return err
	}
	tType := req.GetParams().GetTaskType()
	if pbCom.TaskType_LEARN == tType {
		var trainResp *trainer.TrainResponse
		respC := make(chan *trainer.TrainResponse, 1)
		select {
		case m.trainC <- trainRequest{stopRequest: req, responseC: respC}:
			trainResp = <-respC
			if trainResp != nil && trainResp.Err != nil {
				return trainResp.Err
			}
		case <-m.doneC:
			return errorx.New(errcodes.ErrCodeNotFound, "mpc is stopped")
		}
	} else if pbCom.TaskType_PREDICT == tType {
		respC := make(chan *predictor.PredictResponse, 1)
		select {
		case m.predictC <- predictRequest{stopRequest: req, responseC: respC}:
			predicResp := <-respC
			if predicResp != nil && predicResp.Err != nil {
				return predicResp.Err
			}
		case <-m.doneC:
			return errorx.New(errcodes.ErrCodeNotFound, "mpc is stopped")
		}
	} else {
		return errorx.New(errcodes.ErrCodeParam, "invalid TaskType %s", tType)
	}
	return nil
}

// run listens message and processes it
func (m *mpc) run() {
	for {
		select {
		case tReq := <-m.trainC:
			if tReq.startRequest != nil { // to start a new task
				err := m.trainer.NewLearner(tReq.startRequest)
				if tReq.responseC != nil { // avoid blocking
					tReq.responseC <- &trainer.TrainResponse{Err: err}
					close(tReq.responseC)
				}
			} else if tReq.stopRequest != nil { // to end a task
				err := m.trainer.DeleteLearner(tReq.stopRequest)
				if tReq.responseC != nil { // avoid blocking
					tReq.responseC <- &trainer.TrainResponse{Err: err}
					close(tReq.responseC)
				}
			} else if tReq.validRequest != nil { // to trigger validation
				m.trainer.Validate(tReq.validRequest, tReq.responseC)
			} else { // to train a model
				m.trainer.Train(tReq.trainRequest, tReq.responseC)
			}
		case pReq := <-m.predictC:
			if pReq.startRequest != nil { // to start a new task
				err := m.predictor.NewModel(pReq.startRequest)
				if pReq.responseC != nil { // avoid blocking
					pReq.responseC <- &predictor.PredictResponse{Err: err}
					close(pReq.responseC)
				}
			} else if pReq.stopRequest != nil { // to end a task
				err := m.predictor.DeleteModel(pReq.stopRequest)
				if pReq.responseC != nil { // avoid blocking
					pReq.responseC <- &predictor.PredictResponse{Err: err}
					close(pReq.responseC)
				}
			} else { // to do local prediction with outcomes from remote node
				m.predictor.Predict(pReq.predictRequest, pReq.responseC)
			}
		case <-m.stopC:
			close(m.doneC)
			return
		}
	}
}

func (m *mpc) isRunning() error {
	select {
	case <-m.doneC:
		return errorx.New(errcodes.ErrCodeNotFound, "mpc is stopped")
	default:
	}

	return nil
}

// ModelHolder to save the trained models and prediction outcomes,
// set into Trainer and Predictor in initialization phase
type ModelHolder interface {

	// SaveModel to persist a model
	SaveModel(*pbCom.TrainTaskResult) error

	// SavePredictOut to persist predicting outcomes
	//  outcomes will be zero-value if the holder has't target tag
	SavePredictOut(*pbCom.PredictTaskResult) error
}

// TrainCallBack contains some methods that would be called when finish training
type TrainCallBack struct {
	//modelHolder ModelHolder
	ModelHolder
	//taskHandler Mpc
	Mpc
}

// PredictCallBack contains some methods would be called when finish prediction
type PredictCallBack struct {
	//modelHolder
	ModelHolder
	//taskHandler Mpc
	Mpc
}

// Config is used when start mpc
type Config struct {
	Address          string        // local address, like ip:port
	TrainTaskLimit   int           // indicates the upper limit of the number of training task
	PredictTaskLimit int           // indicates the upper limit of the number of prediction task
	RpcTimeout       time.Duration // rpc connection releases when timeout elapses. eg. 3 means 3*time.Second
}

func newMpc(mh ModelHolder, p2p P2P, conf Config) *mpc {
	rpcHandler := cluster.NewRpcClient(p2p, conf.RpcTimeout*time.Second)

	m := &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}
	trainCallback := TrainCallBack{ModelHolder: mh, Mpc: m}
	m.trainer = trainer.NewTrainer(conf.Address, rpcHandler, &trainCallback, conf.TrainTaskLimit*21+600)
	//there will be 10 more leaners running in parallel if one 10-fold cross validation is invoked
	//there will be 20 more leaners running in parallel if one 10-fold cross validation is invoked together with live evaluation
	//there will be 1 more leaners running in parallel if one live evaluation is invoked
	//and, reserve 600 positions for LOO(one way to evaluate model)

	predictCallBack := PredictCallBack{ModelHolder: mh, Mpc: m}
	m.predictor = predictor.NewPredictor(conf.Address, rpcHandler, &predictCallBack, conf.PredictTaskLimit+conf.TrainTaskLimit*21+600)
	//there will be 10 more models running in parallel if one 10-fold cross validation is invoked
	//there will be 20 more models running in parallel if one 10-fold cross validation is invoked together with live evaluation
	//there will be 1 more model running in parallel if one live evaluation is invoked
	//and, reserve 600 positions for LOO(one way to evaluate model)

	return m
}

// StartMpc creates a mpc instance and run it
func StartMpc(mh ModelHolder, p2p P2P, conf Config) Mpc {
	m := newMpc(mh, p2p, conf)
	go m.run()
	return m
}
