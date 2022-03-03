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
	"fmt"
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
	xchainblockchain "github.com/PaddlePaddle/PaddleDTX/dai/blockchain/xchain"
	"github.com/PaddlePaddle/PaddleDTX/dai/config"
	"github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common/csv"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
	util "github.com/PaddlePaddle/PaddleDTX/dai/util/strings"
	xdbchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
)

type Client struct {
	XchainClient *xchainblockchain.XChain
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

	// get blockchain client
	xchainClient, err := xchainblockchain.New(config.GetCliConf().Xchain)
	if err != nil {
		return nil, err
	}
	return &Client{XchainClient: xchainClient}, nil
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
		file, err := c.XchainClient.GetFileByID(fileID)
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
		executorNode, err := c.XchainClient.GetExecutorNodeByName(executors[index])
		if err != nil {
			return nil, errorx.Wrap(err, "failed to get executor node by node name")
		}
		dataSets = append(dataSets, &pbTask.DataForTask{
			Owner:     file.Owner,
			Executor:  executorNode.ID,
			PSILabel:  psiLabels[index],
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

// Publish publishes a task
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
	task.ID = taskUuid.String()

	// sign task info
	s, err := json.Marshal(task)
	if err != nil {
		return taskId, errorx.Wrap(err, "failed to marshal fl task")
	}
	sig, err := ecdsa.Sign(privkey, hash.HashUsingSha256(s))
	if err != nil {
		return taskId, errorx.Wrap(err, "failed to sign fl task")
	}
	pubOpt := &blockchain.PublishFLTaskOptions{
		FLTask:    &task,
		Signature: sig[:],
	}

	if err := c.XchainClient.PublishTask(pubOpt); err != nil {
		return taskId, err
	}
	return task.ID, nil
}

// GetTaskById gets task by taskID
func (c *Client) GetTaskById(id string) (t blockchain.FLTask, err error) {
	t, err = c.XchainClient.GetTaskById(id)
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
func (c *Client) ListTask(pubkeyStr, status string, start, end,
	limit int64) (tasks blockchain.FLTasks, err error) {
	pubkey, err := hex.DecodeString(pubkeyStr)
	if err != nil {
		return tasks, errorx.Wrap(err, "failed to decode public key")
	}

	tasks, err = c.XchainClient.ListTask(&blockchain.ListFLTaskOptions{
		PubKey:    pubkey[:],
		TimeStart: start,
		TimeEnd:   end,
		Status:    status,
		Limit:     limit,
	})
	return
}

// StartTask starts task by taskID
func (c *Client) StartTask(privateKey, id string) (err error) {
	pubkey, privkey, err := checkUserPrivateKey(privateKey)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("%s,%x", id, pubkey[:])
	sig, err := ecdsa.Sign(privkey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign fl task")
	}
	err = c.XchainClient.StartTask(id, sig[:])
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
	task, err := c.XchainClient.GetTaskById(taskID)
	if err != nil {
		return err
	}
	// check task type
	if task.AlgoParam.TaskType != pbCom.TaskType_PREDICT {
		return errorx.New(errorx.ErrCodeParam, "invalid task type, not a predict task")
	}
	// get training task
	modelTask, err := c.XchainClient.GetTaskById(task.AlgoParam.ModelTaskID)
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
	msg := fmt.Sprintf("%x,%s", in.PubKey, in.TaskID)
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

// ListExecutorNodes list executor nodes
func (c *Client) ListExecutorNodes() (nodes blockchain.ExecutorNodes, err error) {
	return c.XchainClient.ListExecutorNodes()
}

// GetExecutorNodeByID get executor node by nodeID
func (c *Client) GetExecutorNodeByID(pubkeyStr string) (node blockchain.ExecutorNode, err error) {
	pubkey, err := hex.DecodeString(pubkeyStr)
	if err != nil {
		return node, errorx.Wrap(err, "failed to decode executor public key")
	}
	return c.XchainClient.GetExecutorNodeByID(pubkey[:])
}

// GetExecutorNodeByName get executor node by nodeName
func (c *Client) GetExecutorNodeByName(name string) (node blockchain.ExecutorNode, err error) {
	return c.XchainClient.GetExecutorNodeByName(name)
}

// GetAuthByID get file authorization application detail by authID
func (c *Client) GetFileAuthByID(id string) (fileAuth xdbchain.FileAuthApplication, err error) {
	return c.XchainClient.GetAuthApplicationByID(id)
}

// ListFileAuthApplications query the list of authorization applications
func (c *Client) ListFileAuthApplications(opt *xdbchain.ListFileAuthOptions) (fileAuths xdbchain.FileAuthApplications, err error) {
	return c.XchainClient.ListFileAuthApplications(opt)
}
