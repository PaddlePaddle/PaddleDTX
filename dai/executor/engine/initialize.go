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
	"strings"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/peer"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain/xchain"
	"github.com/PaddlePaddle/PaddleDTX/dai/config"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/handler"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/monitor"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/storage/local"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/storage/xuperdb"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc"
	"github.com/PaddlePaddle/PaddleDTX/dai/p2p"
	"github.com/PaddlePaddle/PaddleDTX/dai/util/file"
)

const (
	// The number of executing task concurrently
	DefaultTrainTaskLimit   = 100
	DefaultPredictTaskLimit = 100
	DefaultRpcTimeout       = 3

	// Task default max execution time
	DefaultMpcTaskMaxExecTime = time.Hour * 2
	// Task loop default interval time
	DefaultRequestInterval = time.Second * 10
)

// initEngine initiates Engine
func initEngine(conf *config.ExecutorConf) (e *Engine, err error) {
	// get blockchain instance
	chain, err := newBlockchain(conf.Blockchain)
	if err != nil {
		return e, err
	}
	// initiate local node account
	node, err := newNode(conf)
	if err != nil {
		return e, err
	}
	// get storage instance to save model or prediction result
	storage, err := newStorage(conf.Storage)
	if err != nil {
		return e, err
	}
	// get download instance to download sample files
	download, err := newExecutionMode(conf.Mode, node.PrivateKey)
	if err != nil {
		return e, err
	}
	// get MPC instance to handle tasks
	mpcHandler, err := newMpc(conf.Mpc, node, storage, download, chain)
	if err != nil {
		return e, err
	}
	// get Monitor to handle loop request
	taskMonitor, err := newMonitor(download.Type, node.PrivateKey, chain, mpcHandler)
	if err != nil {
		return e, err
	}
	logger.Info("initiate engine successfully")

	return &Engine{
		node:       node,
		chain:      chain,
		storage:    storage,
		mpcHandler: mpcHandler,
		monitor:    taskMonitor,
	}, nil
}

// newBlockchain initiates blockchain client
func newBlockchain(conf *config.ExecutorBlockchainConf) (b handler.Blockchain, err error) {
	switch conf.Type {
	case "xchain":
		b, err = xchain.New(conf.Xchain)
	default:
		return b, errorx.New(errorx.ErrCodeConfig, "invalid blockchain type: %s", conf.Type)
	}
	return b, err
}

// newNode loads account
func newNode(conf *config.ExecutorConf) (node handler.Node, err error) {
	sk, err := ecdsa.DecodePrivateKeyFromString(conf.PrivateKey)

	if err != nil {
		return node, errorx.Wrap(err, "failed to decode private key")
	}

	pk := ecdsa.PublicKeyFromPrivateKey(sk)
	local := handler.Node{
		Local: peer.Local{
			Address:    conf.PublicAddress,
			ID:         pk[:],
			Name:       conf.Name,
			PrivateKey: sk,
		},
	}
	return local, nil
}

// newStorage initiates local storage, contains train-model and prediction-result storage
func newStorage(conf *config.ExecutorStorageConf) (fileStroage handler.FileStorage, err error) {
	// train-model could only be stored locally
	mStorage, err := local.New(conf.LocalModelStoragePath)
	if err != nil {
		return fileStroage, errorx.New(errorx.ErrCodeConfig, "invalid model storage path：%s", err)
	}

	// evaluation result could only be stored locally
	eStorage, err := local.New(conf.LocalEvaluationStoragePath)
	if err != nil {
		return fileStroage, errorx.New(errorx.ErrCodeConfig, "invalid evaluation result storage path：%s", err)
	}
	// prediction-result could stored in local path or in xuperdb
	pStroage, err := newPredictStorage(conf)
	if err != nil {
		return fileStroage, err
	}

	fileStroage = handler.FileStorage{
		ModelStorage:      mStorage,
		EvaluationStorage: eStorage,
		PredictStorage:    pStroage,
	}
	return fileStroage, nil
}

// newPredictStorage initiates prediction result store client
func newPredictStorage(conf *config.ExecutorStorageConf) (s handler.Storage, err error) {
	switch conf.Type {
	case "Local":
		s, err = local.New(conf.Local.LocalPredictStoragePath)
		if err != nil {
			return s, errorx.New(errorx.ErrCodeConfig, "invalid prediction result-path：%s", err)
		}
	case "XuperDB":
		// if conf.XuperDB.PrivateKey is empty, get the dataOwner client privateKey from conf.XuperDB.KeyPath
		if conf.XuperDB.PrivateKey == "" {
			privateKeyBytes, err := file.ReadFile(conf.XuperDB.KeyPath, file.PrivateKeyFileName)
			if err == nil && len(privateKeyBytes) != 0 {
				conf.XuperDB.PrivateKey = strings.TrimSpace(string(privateKeyBytes))
			} else {
				return s, errorx.New(errorx.ErrCodeConfig, "invalid xuperdb privateKey-path：%s", err)
			}
		}
		privateKey, err := ecdsa.DecodePrivateKeyFromString(conf.XuperDB.PrivateKey)
		if err != nil {
			return s, errorx.Wrap(err, "failed to decode xuperdb private key")
		}
		// get XuperDB instance to upload and download files
		s = xuperdb.New(conf.XuperDB.ExpireTime, conf.XuperDB.NameSpace, conf.XuperDB.Host, privateKey)
	default:
		return s, errorx.New(errorx.ErrCodeConfig, "invalid predict stroage type: %s", conf.Type)
	}
	return s, nil
}

// newExecutionMode initiates sample file download client.
// The sample file download client also represents the task execution type, such as proxy-execution or self-execution.
// If type is 'Proxy', during the task execution, the executor node publishes a file authorization application
// to the dataOwner node, only the sample file authorized to the executor node can be used for task training.
// If type is 'Self', it means that the executor node already has permission to use the sample file.
func newExecutionMode(conf *config.ExecutorModeConf, nodePrivateKey ecdsa.PrivateKey) (fileDownload handler.FileDownload, err error) {
	switch conf.Type {
	case handler.ProxyExecutionMode:
		fileDownload.Type = conf.Type
	case handler.SelfExecutionMode:
		// if conf.Self.PrivateKey is empty, get the dataOwner client privateKey from conf.Self.KeyPath
		if conf.Self.PrivateKey == "" {
			privateKeyBytes, err := file.ReadFile(conf.Self.KeyPath, file.PrivateKeyFileName)
			if err == nil && len(privateKeyBytes) != 0 {
				conf.Self.PrivateKey = strings.TrimSpace(string(privateKeyBytes))
			} else {
				return fileDownload, errorx.New(errorx.ErrCodeConfig, "invalid dataOwner privateKey-path of executor mode：%s", err)
			}
		}
		fileDownload.PrivateKey, err = ecdsa.DecodePrivateKeyFromString(conf.Self.PrivateKey)
		if err != nil {
			return fileDownload, errorx.Wrap(err, "failed to decode dataOwner private key of executor mode")
		}
		// if executer mode type is DataOwner, the dataOwner node's Host can not be empty
		if conf.Self.Host == "" {
			return fileDownload, errorx.Wrap(err, "invalid dataOwner conf of executor mode, host can not be empty")
		}
		fileDownload.Type = conf.Type
		fileDownload.Host = conf.Self.Host
	default:
		return fileDownload, errorx.New(errorx.ErrCodeConfig, "invalid executor mode type: %s", conf.Type)
	}
	fileDownload.NodePrivateKey = nodePrivateKey
	return fileDownload, nil
}

// newMpc starts MPC handler to do MPC-Training and MPC-Prediction tasks
func newMpc(conf *config.ExecutorMpcConf, node handler.Node, fstorage handler.FileStorage,
	fdownload handler.FileDownload, chain handler.Blockchain) (handler.MpcHandler, error) {

	rpcTimeout := time.Duration(conf.RpcTimeout)
	if rpcTimeout == 0 {
		rpcTimeout = DefaultRpcTimeout
	}
	taskLimitTime := time.Duration(conf.TaskLimitTime) * time.Second
	if taskLimitTime == 0 {
		taskLimitTime = DefaultMpcTaskMaxExecTime
	}
	mpcHandler := &handler.MpcModelHandler{
		Config: mpc.Config{
			Address:          node.Address,
			TrainTaskLimit:   conf.TrainTaskLimit,
			PredictTaskLimit: conf.PredictTaskLimit,
			RpcTimeout:       rpcTimeout,
		},
		Storage:            fstorage,
		Download:           fdownload,
		Node:               node,
		Chain:              chain,
		MpcTaskMaxExecTime: taskLimitTime,
		MpcTasks:           make(map[string]*handler.FlTask),
	}

	clusterP2p := p2p.NewP2P()
	mpcServer := mpc.StartMpc(mpcHandler, clusterP2p, mpcHandler.Config)
	mpcHandler.Mpc = mpcServer
	mpcHandler.ClusterP2p = clusterP2p

	return mpcHandler, nil
}

// newMonitor returns Monitor whose works are mainly monitoring status of tasks
// and starting Mpc-Training and Mpc-Prediction tasks
func newMonitor(fileDownloadType string, privateKey ecdsa.PrivateKey, chain handler.Blockchain,
	mpcHandler handler.MpcHandler) (*monitor.TaskMonitor, error) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(privateKey)
	return &monitor.TaskMonitor{
		ExecutionType:   fileDownloadType,
		PrivateKey:      privateKey,
		PublicKey:       pubkey,
		RequestInterval: DefaultRequestInterval,

		Blockchain: chain,
		MpcHandler: mpcHandler,
	}, nil
}
