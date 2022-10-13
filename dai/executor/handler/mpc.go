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

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	reModel "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/cluster"
	"github.com/PaddlePaddle/PaddleDTX/dai/p2p"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
	util "github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/strings"
)

var (
	logger = logrus.WithField("module", "handler.mpc")
)

// MpcHandler starts mpc-training or mpc-prediction when gets task from blockchain,
//  persists the trained models and prediction outcomes.
type MpcHandler interface {
	// SaveModel persists a model
	// called by MPC
	SaveModel(*pbCom.TrainTaskResult) error

	// SavePredictOut persists predicting outcomes
	// outcomes will be zero-value if the holder does not have target feature
	// called by MPC
	SavePredictOut(*pbCom.PredictTaskResult) error

	// GetMpcClusterService returns mpc cluster service server
	GetMpcClusterService() *cluster.Service

	// TaskStartPrepare prepares resources needed by task, and adds task to execution pool.
	TaskStartPrepare(task blockchain.FLTask) (*pbCom.StartTaskRequest, error)

	// StartLocalMpcTask executes task
	StartLocalMpcTask(task *pbCom.StartTaskRequest, isSendTaskToOthers bool) error

	// GetAvailableTasksNum returns left number of tasks could be executed
	GetAvailableTasksNum() (int, int)

	// CheckMpcTimeOutTasks checks tasks in execution pool if they're expired,
	// and stops expired tasks
	CheckMpcTimeOutTasks()

	//Close closes all inner services
	Close()
}

// FlTask include task details and task execution expired time
type FlTask struct {
	// task detail info on chain
	pbTask.FLTask
	// timeout for task execution
	ExpiredTime int64
}

// MpcModelHandler handler for mpc training or prediction tasks
type MpcModelHandler struct {
	Config             mpc.Config
	Node               Node          // executor node information
	Storage            FileStorage   // handler for computing results storage
	Download           FileDownload  // handler for file download, 'proxy' or 'self'
	Chain              Blockchain    // handler for blockchain operation
	MpcTaskMaxExecTime time.Duration // maximum execution time for mpc task
	Mpc                mpc.Mpc
	ClusterP2p         *p2p.P2P
	// store execution mpc tasks
	MpcTasks map[string]*FlTask
	sync.RWMutex
}

// ParticipantParams local parameters required for task execution
type ParticipantParams struct {
	otherParts    []string // other execution parties
	fileText      []byte   // sample file content
	isTagPart     bool     // if local party contains label feature
	psiLabel      string   // feature name for psi
	PaddleFLRole  int
	PaddleFLNodes [3]string
}

// GetMpcClusterService returns mpc cluster service
func (m *MpcModelHandler) GetMpcClusterService() *cluster.Service {
	return cluster.NewService(m.Mpc)
}

// GetAvailableTasksNum returns left number of tasks could be executed
// Returns the number of tasks that can participate in training or prediction
func (m *MpcModelHandler) GetAvailableTasksNum() (tNum int, pNum int) {
	trainTaskNum := 0
	predictTaskNum := 0
	m.RLock()
	// get training or predicting tasks number
	for _, task := range m.MpcTasks {
		if task.AlgoParam.TaskType == pbCom.TaskType_LEARN {
			trainTaskNum += 1
		} else {
			predictTaskNum += 1
		}
	}
	m.RUnlock()
	if trainTaskNum >= m.Config.TrainTaskLimit {
		tNum = 0
	} else {
		tNum = m.Config.TrainTaskLimit - trainTaskNum
	}
	if predictTaskNum >= m.Config.PredictTaskLimit {
		pNum = 0
	} else {
		pNum = m.Config.PredictTaskLimit - predictTaskNum
	}
	return tNum, pNum
}

// addTaskIntoMpcHandler add task into execution pool
// first count the number of current training or prediction task,
// if the tasks number reaches the limit, it is not allowed to add task into execution pool
func (m *MpcModelHandler) addTaskIntoMpcHandler(task blockchain.FLTask) error {
	trainTaskNum, predictTaskNum := m.GetAvailableTasksNum()
	if task.AlgoParam.TaskType == pbCom.TaskType_LEARN && trainTaskNum == 0 {
		return errorx.New(errcodes.ErrCodeTooMuchTasks, "Insufficient computing train resources, add task into mpc handler error")
	}
	if task.AlgoParam.TaskType == pbCom.TaskType_PREDICT && predictTaskNum == 0 {
		return errorx.New(errcodes.ErrCodeTooMuchTasks, "Insufficient computing predict resources, add task into mpc handler error")
	}
	m.Lock()
	defer m.Unlock()
	if _, ok := m.MpcTasks[task.TaskID]; ok {
		return errorx.New(errcodes.ErrCodeTaskExists, "task already exists, taskId: %s", task.TaskID)
	}
	m.MpcTasks[task.TaskID] = &FlTask{
		FLTask:      *task,
		ExpiredTime: time.Now().UnixNano() + m.MpcTaskMaxExecTime.Nanoseconds(),
	}
	return nil
}

// TaskStartPrepare prepares resources needed by task, and adds task to execution pool.
func (m *MpcModelHandler) TaskStartPrepare(task blockchain.FLTask) (*pbCom.StartTaskRequest, error) {
	// 1. add task into mpc handler
	if err := m.addTaskIntoMpcHandler(task); err != nil {
		logger.WithError(err).Error("failed to add task into mpc tasks pool")
		return nil, err
	}

	// 2. get task start parameters
	startRequest, err := m.getMpcStartTaskParam(task)
	if err != nil {
		m.updateTaskStatusAndStopLocalMpc(task.TaskID, err.Error(), "")
		return nil, err
	}
	return startRequest, err
}

// StartLocalMpcTask executes task
func (m *MpcModelHandler) StartLocalMpcTask(startRequest *pbCom.StartTaskRequest, isSendTaskToOthers bool) error {
	// 1. if executor is task initiator, send the start signal to other parties
	if isSendTaskToOthers {
		// send task start request to others
		if err := m.sendTaskStartRequestToOthers(startRequest.Hosts, startRequest.TaskID); err != nil {
			m.updateTaskStatusAndStopLocalMpc(startRequest.TaskID, err.Error(), "")
			return err
		}
	}
	// 2. start train or predict task
	if mcpTaskError := m.Mpc.StartTask(startRequest); mcpTaskError != nil {
		m.updateTaskStatusAndStopLocalMpc(startRequest.TaskID, mcpTaskError.Error(), "")
		logger.WithError(mcpTaskError).Errorf("start mpc task error, taskId: %s", startRequest.TaskID)
		return mcpTaskError
	}
	return nil
}

// CheckMpcTimeOutTasks checks tasks in execution pool if they're expired,
// and stops expired tasks
func (m *MpcModelHandler) CheckMpcTimeOutTasks() {
	var timeOutTaskList []string
	m.RLock()
	for _, task := range m.MpcTasks {
		if task.ExpiredTime <= time.Now().UnixNano() {
			timeOutTaskList = append(timeOutTaskList, task.TaskID)
		}
	}
	m.RUnlock()

	// stop expired tasks and update status in blockchain
	for _, taskID := range timeOutTaskList {
		m.updateTaskStatusAndStopLocalMpc(taskID, "task execute time out", "")
	}
}

// updateTaskStatusAndStopLocalMpc used update task status and execute result into chain and stop local mpc task
// executeErr indicates whether task is successfully executed or failed
// executeResult is task result, only for prediction task
func (m *MpcModelHandler) updateTaskStatusAndStopLocalMpc(taskID, executeErr, executeResult string) {
	if err := m.UpdateTaskFinishStatus(taskID, executeErr, executeResult); err != nil {
		logger.WithError(err).Errorf("fail update task status into chain error, taskId: %s", taskID)
	} else {
		logger.Infof("success update task status into chain, taskId: %s", taskID)
	}
	m.stopLocalMpcTask(taskID)
}

// stopLocalMpcTask stops mpc task
func (m *MpcModelHandler) stopLocalMpcTask(taskId string) {
	m.Lock()
	delete(m.MpcTasks, taskId)
	m.Unlock()
	//notify MPC of stop
	if err := m.Mpc.StopTask(&pbCom.StopTaskRequest{TaskID: taskId}); err != nil {
		logger.WithError(err).Errorf("failed to stop mpc handler task, taskId: %s", taskId)
	} else {
		logger.Debugf("stop mpc task, taskId: %s", taskId)
	}
}

// sendTaskStartRequestToOthers sends "start task" request to other Executors
func (m *MpcModelHandler) sendTaskStartRequestToOthers(otherParts []string, taskID string) error {
	for _, participant := range otherParts {
		err := m.sendTaskStartRequest(participant, taskID)
		if err != nil {
			logger.WithError(err).Errorf("failed to start other participants task, taskId: %s", taskID)
			return err
		}
	}

	logger.Infof("success send task request to others, taskId: %s ", taskID)
	return nil
}

// sendTaskStartRequest sends "start task" signal to other Executor
// if task.AlgoParam.Algo is "dnn-paddlefl-vl", the model will be trained by three parties
func (m *MpcModelHandler) sendTaskStartRequest(executorHost, taskID string) (err error) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(m.Node.PrivateKey)
	in := &pbTask.TaskRequest{
		PubKey: pubkey[:],
		TaskID: taskID,
	}
	msg, err := util.GetSigMessage(in)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign for send task start request")
	}

	sig, err := ecdsa.Sign(m.Node.PrivateKey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign fl start task")
	}
	in.Signature = sig[:]
	// send message to remote Executor
	// reuse gRpc connection

	// may StartTask needs more time
	ctx, cancel := context.WithTimeout(context.Background(), m.Config.RpcTimeout*3*time.Second)
	defer cancel()

	peer, err := m.ClusterP2p.GetPeer(executorHost)
	if err != nil {
		return errorx.New(errcodes.ErrCodeRPCFindNoPeer, "failed to get peer %s when do rpc request: %s", executorHost, err.Error())
	}
	defer m.ClusterP2p.FreePeer()
	conn, err := peer.GetConnect()
	if err != nil {
		return errorx.New(errcodes.ErrCodeRPCConnect, "failed to get connection with %s: %s", executorHost, err.Error())
	}
	taskClient := pbTask.NewTaskClient(conn)

	if _, err := taskClient.StartTask(ctx, in); err != nil {
		return errorx.Wrap(err, "failed to send task to others")
	}
	return nil
}

// UpdateTaskFinishStatus updates task status in blockchain when task finished
func (m *MpcModelHandler) UpdateTaskFinishStatus(taskId, taskErr, taskResult string) error {
	// get task details from chain
	task, err := m.Chain.GetTaskById(taskId)
	if err != nil {
		return err
	}

	// check task status, no need to repeatedly update task
	if task.Status == blockchain.TaskFinished || task.Status == blockchain.TaskFailed {
		logger.Infof("task status already update, taskId: %s, task.status: %s", taskId, task.Status)
		return nil
	}
	if task.Status != blockchain.TaskProcessing {
		return errorx.New(errorx.ErrCodeInternal, "update task status error, task status is not processing")
	}
	// invoke the contract to update the task's status
	pubkey := ecdsa.PublicKeyFromPrivateKey(m.Node.PrivateKey)
	execTaskOptions := &blockchain.FLTaskExeStatusOptions{
		Executor:    pubkey[:],
		TaskID:      taskId,
		CurrentTime: time.Now().UnixNano(),
		ErrMessage:  taskErr,
		Result:      taskResult,
	}
	msg, err := util.GetSigMessage(execTaskOptions)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign for update mpc task")
	}

	sig, err := ecdsa.Sign(m.Node.PrivateKey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return err
	}
	execTaskOptions.Signature = sig[:]
	if err := m.Chain.FinishTask(execTaskOptions); err != nil {
		return err
	}
	return nil
}

// SaveModel persists a model
// called by MPC
func (m *MpcModelHandler) SaveModel(result *pbCom.TrainTaskResult) error {
	m.RLock()
	if _, ok := m.MpcTasks[result.TaskID]; !ok {
		m.RUnlock()
		logger.Debugf("train task already execution complete, taskId: %s", result.TaskID)
		return nil
	}
	m.RUnlock()

	// if the model training fails, update task status from 'Processing' to 'Failed'
	if !result.Success {
		m.updateTaskStatusAndStopLocalMpc(result.TaskID, result.ErrMsg, "")
		return nil
	}

	// store model
	r := bytes.NewReader(result.Model)
	if _, err := m.Storage.ModelStorage.Write(r, result.TaskID); err != nil {
		err := errorx.New(errorx.ErrCodeInternal, "failed to locally save task model")
		m.updateTaskStatusAndStopLocalMpc(result.TaskID, err.Error(), "")
		return err
	}
	// store evaluation result,
	// and keep going forward even if some errors happen
	if result.EvalMetricScores != nil {
		textEvalMetricScores, err := json.Marshal(result.EvalMetricScores)
		if err == nil {
			r := bytes.NewReader(textEvalMetricScores)
			if _, errS := m.Storage.EvaluationStorage.Write(r, result.TaskID); errS != nil {
				logger.Warnf("failed to locally save evaluation result: %s, taskId: %s, error: %s", string(textEvalMetricScores), result.TaskID, errS.Error())
			}
		} else {
			logger.Warnf("failed to jsonMarshal evaluation resul, taskId: %s, error: %s", result.TaskID, err.Error())
		}

	}
	logger.Debugf("successfully saved model, taskId: %s", result.TaskID)
	m.updateTaskStatusAndStopLocalMpc(result.TaskID, "", "")
	return nil
}

// SavePredictOut persists predicting outcomes
// Outcomes will be zero-value if the holder does not have target feature
// called by MPC
func (m *MpcModelHandler) SavePredictOut(result *pbCom.PredictTaskResult) error {
	m.RLock()
	if _, ok := m.MpcTasks[result.TaskID]; !ok {
		m.RUnlock()
		logger.Debugf("predict task already execution complete, taskId: %s", result.TaskID)
		return nil
	}
	m.RUnlock()

	if !result.Success {
		m.updateTaskStatusAndStopLocalMpc(result.TaskID, result.ErrMsg, "")
		return nil
	}

	if len(result.Outcomes) == 0 {
		// predict successfully, but local node has no outcomes because its samples have no Label
		logger.Debugf("no label parties do not need to store predict result")
		m.stopLocalMpcTask(result.TaskID)
		return nil
	}

	// save prediction result
	r := bytes.NewReader(result.Outcomes)
	// if the storage type of the prediction result is xuperdb, sResult is fileID, otherwise sResult is empty
	psResult, err := m.Storage.PredictStorage.Write(r, result.TaskID)
	if err != nil {
		err := errorx.Wrap(err, "failed to save task predict result, taskId: %s", result.TaskID)
		m.updateTaskStatusAndStopLocalMpc(result.TaskID, err.Error(), "")
		return err
	}
	logger.Debugf("success save predict out, taskId: %s, psResult: %s", result.TaskID, psResult)
	m.updateTaskStatusAndStopLocalMpc(result.TaskID, "", psResult)
	return nil
}

// getMpcStartTaskParam get the parameters required for task startup
func (m *MpcModelHandler) getMpcStartTaskParam(task blockchain.FLTask) (*pbCom.StartTaskRequest, error) {
	partParam, err := m.getTaskParticipantParam(task)
	if err != nil {
		return nil, err
	}

	// train params
	trainParam := task.AlgoParam.TrainParams
	trainParam.IdName = partParam.psiLabel
	trainParam.IsTagPart = partParam.isTagPart

	modeParam := &pbCom.TrainModels{}
	// set task params
	startTaskReqs := &pbCom.StartTaskRequest{
		TaskID: task.TaskID,
		File:   partParam.fileText,
		Hosts:  partParam.otherParts,
		Params: &pbCom.TaskParams{
			Algo:        task.AlgoParam.Algo,
			TaskType:    task.AlgoParam.TaskType,
			TrainParams: trainParam,
			ModelParams: modeParam,
			EvalParams:  task.AlgoParam.EvalParams,
			LivalParams: task.AlgoParam.LivalParams,
		},
		PaddleFLParams: &pbCom.PaddleFLParams{
			Role:  int32(partParam.PaddleFLRole),
			Nodes: partParam.PaddleFLNodes[:],
		},
	}

	// for predict task, model is required
	if task.AlgoParam.TaskType == pbCom.TaskType_PREDICT {
		// get the model locally and convert it to the model parameters
		model, err := m.getTaskModel(task.AlgoParam.ModelTaskID)
		if err != nil {
			return nil, err
		}
		startTaskReqs.Params.ModelParams = model
		startTaskReqs.Params.ModelParams.IdName = partParam.psiLabel
	}
	logger.Infof("get mpc task start param success, taskId: %s, param is: %+v, otherParts: %+v",
		task.TaskID, startTaskReqs, partParam.otherParts)

	return startTaskReqs, nil
}

// getTaskParticipantParam get parameters for task execution
func (m *MpcModelHandler) getTaskParticipantParam(task blockchain.FLTask) (partParam ParticipantParams, err error) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(m.Node.PrivateKey)
	var otherParts []string

	for _, dataset := range task.DataSets {
		//  download sample file
		if bytes.Equal(dataset.Executor, pubkey[:]) {
			isTagPart, err := m.getTargetPart(dataset.DataID, task.AlgoParam.TrainParams.Label)
			if err != nil {
				return partParam, err
			}
			reader, err := m.Download.GetSampleFile(dataset.DataID, m.Chain)
			if err != nil {
				logger.Debugf("get sample file error, taskId: %s, err: %v", task.TaskID, err)
				return partParam, err
			}
			fileText, err := m.getTextByReader(reader)
			reader.Close()

			if err != nil {
				return partParam, err
			}
			partParam.isTagPart = isTagPart
			partParam.fileText = fileText
			partParam.psiLabel = dataset.PsiLabel
		} else {
			otherParts = append(otherParts, dataset.Address)
			logger.Infof("got one other party: %s", dataset.Address)
		}
	}
	partParam.otherParts = otherParts

	// if task's execution use paddlefl
	paddleFLNodes := [3]string{}
	if task.AlgoParam.Algo == pbCom.Algorithm_DNN_PADDLEFL_VL {
		nodes, err := m.Chain.ListExecutorNodes()
		if err != nil {
			return partParam, err
		}
		for _, v := range nodes {
			paddleFLNodes[v.PaddleFLRole] = v.PaddleFLAddress
			if bytes.Equal(v.ID, pubkey[:]) {
				partParam.PaddleFLRole = v.PaddleFLRole
			}
		}
	}
	partParam.PaddleFLNodes = paddleFLNodes

	return partParam, nil
}

// getTargetPart determine whether local sample has label by parsing file extra information
func (m *MpcModelHandler) getTargetPart(fileID, labelName string) (bool, error) {
	sampleFile, err := m.Chain.GetFileByID(fileID)
	if err != nil {
		return false, errorx.New(errorx.ErrCodeInternal, "failed to get file")
	}
	// parse file struct
	fileExtra := blockchain.FLInfo{}
	if err := json.Unmarshal(sampleFile.Ext, &fileExtra); err != nil {
		return false, errorx.New(errorx.ErrCodeInternal, "failed to get file extra info")
	}
	fileFeatures := strings.Split(fileExtra.Features, ",")
	isTagPart := util.IsContain(fileFeatures, labelName)

	return isTagPart, nil
}

// getTextByReader get file content from io reader
func (m *MpcModelHandler) getTextByReader(reader io.ReadCloser) ([]byte, error) {
	text, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "failed to get text by reader")
	}
	return text, nil
}

// getTaskModel get model for prediction task
func (m *MpcModelHandler) getTaskModel(taskId string) (*pbCom.TrainModels, error) {
	model, err := m.Storage.ModelStorage.Read(taskId)
	if err != nil {
		return nil, err
	}
	defer model.Close()
	trainModel, err := m.getTextByReader(model)
	if err != nil {
		return nil, err
	}
	// retrieve train models from bytes
	reTrainModel, err := reModel.TrainModelsFromBytes(trainModel)
	if err != nil {
		return nil, err
	}
	return reTrainModel, nil
}

// Close waits until all inner services stop
func (m *MpcModelHandler) Close() {
	m.Mpc.Stop()
	m.ClusterP2p.Stop()
	logger.Infof("mpc handler stop")
}
