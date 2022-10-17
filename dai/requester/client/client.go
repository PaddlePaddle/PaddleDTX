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

package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	fabricblockchain "github.com/PaddlePaddle/PaddleDTX/dai/blockchain/fabric"
	xchainblockchain "github.com/PaddlePaddle/PaddleDTX/dai/blockchain/xchain"
	"github.com/PaddlePaddle/PaddleDTX/dai/config"
	"github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common/csv"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
	xdbchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	util "github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/strings"
)

// Blockchain defines some contract methods
type Blockchain interface {
	// executor operation
	GetExecutorNodeByID(id string) (blockchain.ExecutorNode, error)
	GetExecutorNodeByName(name string) (blockchain.ExecutorNode, error)
	ListExecutorNodes() (blockchain.ExecutorNodes, error)

	// task operation
	ListTask(opt *blockchain.ListFLTaskOptions) (blockchain.FLTasks, error)
	PublishTask(opt *blockchain.PublishFLTaskOptions) error
	StartTask(opt *blockchain.StartFLTaskOptions) error
	GetTaskById(id string) (blockchain.FLTask, error)

	// file operation
	GetFileByID(id string) (xdbchain.File, error)
	ListFileAuthApplications(opt *xdbchain.ListFileAuthOptions) (xdbchain.FileAuthApplications, error)
	GetAuthApplicationByID(authID string) (xdbchain.FileAuthApplication, error)
}

// Client requester client, used to publish task and retrieve task result
type Client struct {
	chainClient Blockchain
}

func newChainClient(conf *config.ExecutorBlockchainConf) (b Blockchain, err error) {
	switch conf.Type {
	case "xchain":
		b, err = xchainblockchain.New(conf.Xchain)
	case "fabric":
		b, err = fabricblockchain.New(conf.Fabric)
	default:
		return b, errorx.New(errorx.ErrCodeConfig, "invalid blockchain type: %s", conf.Type)
	}
	return b, err
}

// GetRequestClient returns client for Requester by blockchain configuration
func GetRequestClient(configPath string) (*Client, error) {
	// check blockchain config yaml
	err := checkConfig(configPath)
	if err != nil {
		return nil, err
	}
	// clear the standard output of the chain contract invoke
	log.SetOutput(ioutil.Discard)
	chainClient, err := newChainClient(config.GetCliConf())
	if err != nil {
		return nil, err
	}
	return &Client{chainClient: chainClient}, nil
}

// PublishOptions define parameters used to publishing a task
type PublishOptions struct {
	PrivateKey  string           // requester private key
	Files       string           // file list with "," as delimiter, used for training or prediction
	Executors   string           // executor nodes with "," as delimiter, used for executing a training or prediction task
	TaskName    string           // task name, not unique for one requester
	AlgoParam   pbCom.TaskParams // parameters required for training or prediction
	PSILabels   string           // ID feature name list with "," as delimiter, used for PSI
	Description string           // task description
}

// checkPublishTaskOptions checks params for publishing task
func (c *Client) checkPublishTaskOptions(opt PublishOptions) ([]*pbTask.DataForTask, error) {
	if opt.TaskName == "" {
		return nil, errorx.New(errorx.ErrCodeParam, "taskName can not be empty")
	}
	// 1. check taskID for predict task
	if opt.AlgoParam.TaskType == pbCom.TaskType_PREDICT {
		if opt.AlgoParam.ModelTaskID == "" {
			return nil, errorx.New(errorx.ErrCodeParam, "taskID can not empty for predict task")
		}
		task, err := c.GetTaskById(opt.AlgoParam.ModelTaskID)
		if err != nil || task.Status != blockchain.TaskFinished {
			return nil, errorx.New(errorx.ErrCodeParam, "failed to get task or task status is not finished")
		}
	} else {
		if opt.AlgoParam.TrainParams.Label == "" {
			return nil, errorx.New(errorx.ErrCodeParam, "label can not empty for train task")
		}
		if opt.AlgoParam.Algo == pbCom.Algorithm_LOGIC_REGRESSION_VL && opt.AlgoParam.TrainParams.LabelName == "" {
			return nil, errorx.New(errorx.ErrCodeParam, "labelName can not be empty for logistic-vl")
		}
	}

	// 2. check data sets number and executor nodes number, at least two parties
	fileIDs := strings.Split(strings.TrimSpace(opt.Files), ",")
	executors := strings.Split(strings.TrimSpace(opt.Executors), ",")
	if len(fileIDs) < 2 {
		return nil, errorx.New(errorx.ErrCodeParam, "not enough data set numbers, got: %d", len(fileIDs))
	}
	if util.IsContainDuplicateItems(fileIDs) {
		return nil, errorx.New(errorx.ErrCodeParam, "sample file IDs cannot be the same")
	}
	// the number of files and the number of nodes must be the same
	// when task is executed, the executor node take the sample file with the same index position
	if len(fileIDs) != len(executors) {
		return nil, errorx.New(errorx.ErrCodeParam, "sample files num not match executor nodes num, got: %d", len(executors))
	}
	if util.IsContainDuplicateItems(executors) {
		return nil, errorx.New(errorx.ErrCodeParam, "executor node names cannot be the same")
	}

	// 3. check if algorithm exists
	psiLabels := strings.Split(strings.TrimSpace(opt.PSILabels), ",")
	if _, ok := blockchain.VlAlgorithmListValue[opt.AlgoParam.Algo]; ok {
		if opt.PSILabels == "" {
			return nil, errorx.New(errorx.ErrCodeParam, "PSILabel cannot be empty for vertical train task")
		}
		if len(fileIDs) != len(psiLabels) {
			return nil, errorx.New(errorx.ErrCodeParam, "sample file num not match psi label num")
		}
	}

	// 4. check if dataset and specified label exist
	var dataSets []*pbTask.DataForTask
	var isTagPart bool
	isLabelExist := 0
	for index, fileID := range fileIDs {
		file, err := c.chainClient.GetFileByID(fileID)
		if err != nil {
			return nil, err
		}

		fileExtra := blockchain.FLInfo{}
		if err := json.Unmarshal(file.Ext, &fileExtra); err != nil {
			return nil, errorx.New(errorx.ErrCodeInternal, "failed to get file extra info: %v", err)
		}
		fileFeatures := strings.Split(fileExtra.Features, ",")
		// check if psiLabel exists in feature list
		if !util.IsContain(fileFeatures, psiLabels[index]) {
			return nil, errorx.New(errorx.ErrCodeParam, "features of file does not contain psiLabel")
		}

		// check if label exists in one of the datasets
		if util.IsContain(fileFeatures, opt.AlgoParam.TrainParams.Label) {
			isLabelExist += 1
			isTagPart = true
		}
		// only one party is allowed to have label
		if isLabelExist > 1 {
			return nil, errorx.New(errorx.ErrCodeParam, "invalid fileIDs, only one sample file is allowed to have label")
		}

		// get dataID address
		executorNode, err := c.chainClient.GetExecutorNodeByName(executors[index])
		if err != nil {
			return nil, errorx.Wrap(err, "failed to get executor node by node name")
		}
		dataSets = append(dataSets, &pbTask.DataForTask{
			Owner:     file.Owner,
			Executor:  executorNode.ID,
			PsiLabel:  psiLabels[index],
			DataID:    fileID,
			Address:   executorNode.Address,
			IsTagPart: isTagPart,
		})
	}
	if opt.AlgoParam.TaskType == pbCom.TaskType_LEARN && isLabelExist < 1 {
		return nil, errorx.New(errorx.ErrCodeParam, "invalid label, dataSets label doest not exist")
	}
	return dataSets, nil
}

// Publish publishes a task, returns taskID
func (c *Client) Publish(opt PublishOptions) (taskId string, err error) {
	pubkey, privkey, err := checkUserPrivateKey(opt.PrivateKey)
	if err != nil {
		return taskId, err
	}
	dataSets, err := c.checkPublishTaskOptions(opt)
	if err != nil {
		return taskId, err
	}

	task := pbTask.FLTask{
		Name:        opt.TaskName,
		Description: opt.Description,
		Requester:   pubkey[:],
		AlgoParam:   &opt.AlgoParam,
		PublishTime: time.Now().UnixNano(),
		DataSets:    dataSets,
	}

	// generate a uuid as taskId
	taskUuid, err := uuid.NewRandom()
	if err != nil {
		return taskId, errorx.Internal(err, "failed to get uuid")
	}
	task.TaskID = taskUuid.String()

	// sign task info
	m, err := util.GetSigMessage(task)
	if err != nil {
		return taskId, errorx.Internal(err, "failed to get fl task signature message")
	}
	sig, err := ecdsa.Sign(privkey, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return taskId, errorx.Wrap(err, "failed to sign fl task")
	}
	pubOpt := &blockchain.PublishFLTaskOptions{
		FLTask:    &task,
		Signature: sig[:],
	}
	if err := c.chainClient.PublishTask(pubOpt); err != nil {
		return taskId, err
	}
	return task.TaskID, nil
}

// GetTaskById gets task by taskID
func (c *Client) GetTaskById(id string) (t blockchain.FLTask, err error) {
	t, err = c.chainClient.GetTaskById(id)
	if err != nil {
		return t, err
	}
	return t, nil
}

// ListTask lists tasks by requester or executor's public key hex string
// support listing tasks a requester published or tasks an executor involved
// status is task status to search
// only task published after "start" before "end" will be listed
// limit is the maximum number of tasks to response
func (c *Client) ListTask(rPubkeyStr, ePubkeyStr, status string, start, end,
	limit int64) (tasks blockchain.FLTasks, err error) {
	rPubkey, err := hex.DecodeString(rPubkeyStr)
	if err != nil {
		return tasks, errorx.Wrap(err, "failed to decode requester public key")
	}
	ePubkey, err := hex.DecodeString(ePubkeyStr)
	if err != nil {
		return tasks, errorx.Wrap(err, "failed to decode executor public key")
	}
	tasks, err = c.chainClient.ListTask(&blockchain.ListFLTaskOptions{
		PubKey:     rPubkey[:],
		ExecPubKey: ePubkey[:],
		TimeStart:  start,
		TimeEnd:    end,
		Status:     status,
		Limit:      limit,
	})
	return
}

// StartTask starts task by taskID, only task with 'Ready' status can be started
func (c *Client) StartTask(privateKey, id string) (err error) {
	_, privkey, err := checkUserPrivateKey(privateKey)
	if err != nil {
		return err
	}
	sParams := blockchain.StartFLTaskOptions{TaskID: id}
	msg, err := util.GetSigMessage(sParams)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign for start task")
	}
	sig, err := ecdsa.Sign(privkey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign fl task")
	}
	sParams.Signature = sig[:]
	err = c.chainClient.StartTask(&sParams)
	return err
}

// GetPredictResult gets predict result by taskID
// output is the path to save predict result
func (c *Client) GetPredictResult(privateKey, taskID, output string) (err error) {
	pubkey, privkey, err := checkUserPrivateKey(privateKey)
	if err != nil {
		return err
	}

	// get prediction task
	task, err := c.chainClient.GetTaskById(taskID)
	if err != nil {
		return err
	}
	// check task type
	if task.AlgoParam.TaskType != pbCom.TaskType_PREDICT {
		return errorx.New(errorx.ErrCodeParam, "invalid task type, not a predict task")
	}
	// get training task
	modelTask, err := c.chainClient.GetTaskById(task.AlgoParam.ModelTaskID)
	if err != nil {
		return err
	}
	var executorHost string
	for _, dataset := range modelTask.DataSets {
		if dataset.IsTagPart {
			executorHost = dataset.Address
			break
		}
	}

	// connect to result owner
	conn, err := grpc.Dial(executorHost, grpc.WithInsecure())
	if err != nil {
		return errorx.New(errorx.ErrCodeInternal, "CAN_NOT_CONNECT_EXECUTOR_SERVER: %v", err)
	}
	defer conn.Close()
	taskClient := pbTask.NewTaskClient(conn)

	in := &pbTask.TaskRequest{
		PubKey: pubkey[:],
		TaskID: taskID,
	}

	// verify signature
	msg, err := util.GetSigMessage(in)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign for download prediction result")
	}
	sig, err := ecdsa.Sign(privkey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign predict task")
	}
	in.Signature = sig[:]

	// request data node to download predict file
	out, err := taskClient.GetPredictResult(context.Background(), in)
	if err != nil {
		return err
	}
	var rows [][]string
	if err := json.Unmarshal(out.Payload, &rows); err != nil {
		return errorx.Wrap(err, "failed to unmarshal result to rows")
	}
	// save result to csv file
	if err := csv.WriteRowsToFile(rows, output); err != nil {
		return errorx.Wrap(err, "failed to unmarshal result to rows")
	}

	return nil
}

// ListExecutorNodes list all executor nodes
func (c *Client) ListExecutorNodes() (nodes blockchain.ExecutorNodes, err error) {
	return c.chainClient.ListExecutorNodes()
}

// GetExecutorNodeByID get the executor node by nodeID, which is the public key of the node
func (c *Client) GetExecutorNodeByID(pubkeyStr string) (node blockchain.ExecutorNode, err error) {
	_, err = hex.DecodeString(pubkeyStr)
	if err != nil {
		return node, errorx.Wrap(err, "failed to decode executor public key")
	}
	return c.chainClient.GetExecutorNodeByID(pubkeyStr)
}

// GetExecutorNodeByName get executor node by nodeName
func (c *Client) GetExecutorNodeByName(name string) (node blockchain.ExecutorNode, err error) {
	return c.chainClient.GetExecutorNodeByName(name)
}

// GetFileAuthByID get file authorization application detail by authID
func (c *Client) GetFileAuthByID(id string) (fileAuth xdbchain.FileAuthApplication, err error) {
	return c.chainClient.GetAuthApplicationByID(id)
}

// ListFileAuthApplications query the list of authorization applications
func (c *Client) ListFileAuthApplications(opt *xdbchain.ListFileAuthOptions) (fileAuths xdbchain.FileAuthApplications, err error) {
	return c.chainClient.ListFileAuthApplications(opt)
}
