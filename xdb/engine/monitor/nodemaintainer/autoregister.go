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

package nodemaintainer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/hash"
)

// autoregister storage-node automatically register in blockchain
func (m *NodeMaintainer) autoregister(ctx context.Context) error {
	logrus.WithField("module", "autoregister")

	pubkey := ecdsa.PublicKeyFromPrivateKey(m.localNode.PrivateKey)
	node, err := m.blockchain.GetNode(ctx, []byte(pubkey.String()))
	if err == nil && node.Online {
		logrus.Info("node already registered on blockchain")
		return nil
	} else if err == nil && !node.Online {
		nonce := time.Now().UnixNano()
		mes := fmt.Sprintf("%s,%d", pubkey.String(), nonce)
		sig, err := ecdsa.Sign(m.localNode.PrivateKey, hash.Hash([]byte(mes)))
		if err != nil {
			return errorx.Wrap(err, "failed to sign File")
		}
		nodeOpts := &blockchain.NodeOperateOptions{
			NodeID: []byte(pubkey.String()),
			Nonce:  nonce,
			Sig:    sig[:],
		}
		err = m.blockchain.NodeOnline(ctx, nodeOpts)
		if err != nil {
			logrus.Error("node failed online on  blockchain")
		}
		logrus.Info("node online")
		return err
	} else if errorx.Is(err, errorx.ErrCodeNotFound) {
		timestamp := time.Now().UnixNano()
		opt := blockchain.AddNodeOptions{
			Node: blockchain.Node{
				ID:       []byte(pubkey.String()),
				Name:     m.localNode.Name,
				Address:  m.localNode.Address,
				Online:   true,
				RegTime:  timestamp,
				UpdateAt: timestamp,
			},
		}
		// sign node info
		s, err := json.Marshal(opt.Node)
		if err != nil {
			return errorx.Wrap(err, "failed to marshal node")
		}
		sig, err := ecdsa.Sign(m.localNode.PrivateKey, hash.Hash(s))
		if err != nil {
			return errorx.Wrap(err, "failed to sign node")
		}
		opt.Signature = sig[:]
		if err := m.blockchain.AddNode(ctx, &opt); err != nil {
			logrus.Error("failed to register node automatically")
			return errorx.Wrap(err, "failed to register node automatically")
		}
		logrus.WithFields(logrus.Fields{
			"node_id":   pubkey.String(),
			"online_at": timestamp,
		}).Info("success to register node on blockchain")

		return nil
	} else {
		logrus.Errorf("failed to read blockchain: %v", err)
		return errorx.Wrap(err, "failed to read blockchain")
	}
}
