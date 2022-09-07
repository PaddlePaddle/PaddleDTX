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
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	xdbchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/peer"
	util "github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/strings"
)

// Blockchain defines some contract methods
//  refer blockchain module for more
type Blockchain interface {
	// executor operation
	RegisterExecutorNode(opt *blockchain.AddNodeOptions) error
	GetExecutorNodeByID(id string) (blockchain.ExecutorNode, error)
	ListExecutorNodes() (blockchain.ExecutorNodes, error)

	// task operation
	ListTask(opt *blockchain.ListFLTaskOptions) (blockchain.FLTasks, error)
	PublishTask(opt *blockchain.PublishFLTaskOptions) error
	GetTaskById(id string) (blockchain.FLTask, error)
	ConfirmTask(opt *blockchain.FLTaskConfirmOptions) error
	RejectTask(opt *blockchain.FLTaskConfirmOptions) error
	ExecuteTask(opt *blockchain.FLTaskExeStatusOptions) error
	FinishTask(opt *blockchain.FLTaskExeStatusOptions) error
	// get file stored in xuperDB by id
	GetFileByID(id string) (xdbchain.File, error)
	// query the list of authorization applications
	ListFileAuthApplications(opt *xdbchain.ListFileAuthOptions) (xdbchain.FileAuthApplications, error)
	// publish sample file's authorization application
	PublishFileAuthApplication(opt *xdbchain.PublishFileAuthOptions) error
	// query the list of storage nodes
	ListNodes() (xdbchain.Nodes, error)

	Close()
}

type Node struct {
	peer.Local
	HttpAddress     string
	PaddleFLAddress string
	PaddleFLRole    int
}

// Register registers local node to blockchain
func (n *Node) Register(chain Blockchain) error {
	if err := n.autoRegister(chain); err != nil {
		logrus.WithError(err).Error("failed to register node into chain")
		return err
	}

	return nil
}

// autoRegister automatically registers executor node on blockchain when server starts
// after registration, the grpc address of the executor node can be queried from the chain
// for multi-party tasks traing or tasks prediction
func (n *Node) autoRegister(chain Blockchain) error {
	logrus.WithField("module", "handler.node")

	pubkey := ecdsa.PublicKeyFromPrivateKey(n.PrivateKey)
	if _, err := chain.GetExecutorNodeByID(pubkey.String()); err == nil {
		logrus.Info("node already registered on blockchain")
		return nil
	}
	// if executor node not exist, registers executor node into blockchain
	timestamp := time.Now().UnixNano()
	opt := blockchain.AddNodeOptions{
		Node: blockchain.ExecutorNode{
			ID:              pubkey[:],
			Name:            n.Name,
			Address:         n.Address,
			HttpAddress:     n.HttpAddress,
			PaddleFLAddress: n.PaddleFLAddress,
			PaddleFLRole:    n.PaddleFLRole,
			RegTime:         timestamp,
		},
	}
	// get the sign message
	msg, err := util.GetSigMessage(opt)
	if err != nil {
		return errorx.Internal(err, "failed to get message to sign node auto-registration")
	}
	// generate signature
	sig, err := ecdsa.Sign(n.PrivateKey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign node")
	}

	opt.Signature = sig[:]
	if err := chain.RegisterExecutorNode(&opt); err != nil {
		logrus.Error("failed to register node automatically")
		return errorx.Wrap(err, "failed to register node automatically")
	}
	logrus.WithFields(logrus.Fields{
		"node_id":   pubkey.String(),
		"online_at": timestamp,
	}).Info("success to register node on blockchain")

	return nil
}
