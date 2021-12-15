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
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/peer"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain/xchain"
	"github.com/PaddlePaddle/PaddleDTX/dai/config"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/handler"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/monitor"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/storage"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc"
	"github.com/PaddlePaddle/PaddleDTX/dai/p2p"
)

const (
	// Maximum default time for saving predict file results
	DefaultFileRetentionTime = time.Hour * 72
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
	// get storage instance to save model
	localStorage, err := newStorage(conf.Storage)
	if err != nil {
		return e, err
	}
	// get XuperDB instance to upload and download files
	xuperDB, err := newXuperDB(conf.Storage.XuperDB, node.PrivateKey)
	if err != nil {
		return e, err
	}
	// get MPC instance to handle tasks
	mpcHandler, err := newMpc(conf.Mpc, node, localStorage, xuperDB, chain)
	if err != nil {
		return e, err
	}
	// get Monitor to handle loop request
	taskMonitor, err := newMonitor(node.PrivateKey, chain, mpcHandler)
	if err != nil {
		return e, err
	}
	logger.Info("initiate engine successfully")

	return &Engine{
		node:       node,
		chain:      chain,
		storage:    localStorage,
		xuperDB:    xuperDB,
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

// newStorage initiates local storage
func newStorage(conf *config.ExecutorStorageConf) (t *storage.Storage, err error) {
	t, err = storage.New(conf.LocalStoragePath)
	if err != nil {
		return t, errorx.New(errorx.ErrCodeConfig, "invalid train model-path：%s", err)
	}
	return t, nil
}

// newXuperDB initiates xuperDB client used to fetch samples
func newXuperDB(conf *config.XuperDBConf, privateKey ecdsa.PrivateKey) (d handler.XuperDB, err error) {
	expiretime := time.Duration(conf.ExpireTime) * time.Hour
	if expiretime == 0 {
		expiretime = DefaultFileRetentionTime
	}
	d = handler.XuperDB{
		PrivateKey: privateKey,
		Address:    conf.Host,
		Ns:         conf.NameSpace,
		ExpireTime: time.Now().UnixNano() + expiretime.Nanoseconds(),
	}
	return d, nil
}

// newMpc starts MPC handler to do MPC-Training and MPC-Prediction tasks
func newMpc(conf *config.ExecutorMpcConf, node handler.Node, estorage *storage.Storage,
	xuperDB handler.XuperDB, chain handler.Blockchain) (handler.MpcHandler, error) {

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
		Storage:            *estorage,
		XuperDB:            xuperDB,
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
func newMonitor(privateKey ecdsa.PrivateKey, chain handler.Blockchain,
	mpcHandler handler.MpcHandler) (*monitor.TaskMonitor, error) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(privateKey)
	return &monitor.TaskMonitor{
		PrivateKey:      privateKey,
		PublicKey:       pubkey,
		RequestInterval: DefaultRequestInterval,

		Blockchain: chain,
		MpcHandler: mpcHandler,
	}, nil
}
