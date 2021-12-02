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
	"context"
	"encoding/json"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	xdatachain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/peer"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
)

// Blockchain defines some contract methods
//  see more from blockchain.xchain
type Blockchain interface {
	// executor operation
	RegisterDataNode(ctx context.Context, opt *blockchain.AddNodeOptions) error
	GetDataNodeByID(ctx context.Context, id []byte) (blockchain.DataNode, error)
	ListDataNodes(ctx context.Context) (blockchain.DataNodes, error)
	// task operation
	ListTask(ctx context.Context, opt *blockchain.ListFLTaskOptions) (blockchain.FLTasks, error)
	PublishTask(ctx context.Context, opt *blockchain.PublishFLTaskOptions) error
	GetTaskById(ctx context.Context, id string) (blockchain.FLTask, error)
	ConfirmTask(ctx context.Context, opt *blockchain.FLTaskConfirmOptions) error
	RejectTask(ctx context.Context, opt *blockchain.FLTaskConfirmOptions) error
	ExecuteTask(ctx context.Context, opt *blockchain.FLTaskExeStatusOptions) error
	FinishTask(ctx context.Context, opt *blockchain.FLTaskExeStatusOptions) error
	// gets file stored in xuperDB by id
	GetFileByID(ctx context.Context, id string) (xdatachain.File, error)

	Close()
}

type Node struct {
	peer.Local
}

// Register registers local node to blockchain
func (n *Node) Register(ctx context.Context, chain Blockchain) error {
	if err := n.autoregister(ctx, chain); err != nil {
		logrus.WithError(err).Error("failed to register node into chain")
		return err
	}

	return nil
}

// autoregister automatically registers executor node on blockchain when server starts
func (n *Node) autoregister(ctx context.Context, chain Blockchain) error {
	logrus.WithField("module", "handler.node")

	pubkey := ecdsa.PublicKeyFromPrivateKey(n.PrivateKey)
	if _, err := chain.GetDataNodeByID(ctx, pubkey[:]); err == nil {
		logrus.Info("node already registered on blockchain")
		return nil
	}
	timestamp := time.Now().UnixNano()
	opt := blockchain.AddNodeOptions{
		Node: blockchain.DataNode{
			ID:      pubkey[:],
			Name:    n.Name,
			Address: n.Address,
			RegTime: timestamp,
		},
	}
	// sign node info
	s, err := json.Marshal(opt.Node)
	if err != nil {
		return errorx.Wrap(err, "failed to marshal node")
	}
	sig, err := ecdsa.Sign(n.PrivateKey, hash.HashUsingSha256(s))
	if err != nil {
		return errorx.Wrap(err, "failed to sign node")
	}

	opt.Signature = sig[:]
	if err := chain.RegisterDataNode(ctx, &opt); err != nil {
		logrus.Error("failed to register node automatically")
		return errorx.Wrap(err, "failed to register node automatically")
	}
	logrus.WithFields(logrus.Fields{
		"node_id":   pubkey.String(),
		"online_at": timestamp,
	}).Info("success to register node on blockchain")

	return nil
}
