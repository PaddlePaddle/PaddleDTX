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
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/hash"
)

// AddNode adds storage node into blockchain
func (e *Engine) AddNode(ctx context.Context, opt types.AddNodeOptions) (err error) {
	if err := e.verifyUserID(opt.NodeID); err != nil {
		return err
	}

	msg := fmt.Sprintf("%s:%s", opt.Name, opt.Address)
	h := hash.Hash([]byte(msg))
	if err := verifyUserToken(opt.NodeID, opt.Token, h); err != nil {
		return err
	}

	timestamp := time.Now().UnixNano()
	addopt := blockchain.AddNodeOptions{
		Node: blockchain.Node{
			ID:       []byte(opt.NodeID),
			Address:  opt.Address,
			Name:     opt.Name,
			Online:   opt.Online,
			RegTime:  timestamp,
			UpdateAt: timestamp,
		},
	}
	// sign node info
	s, err := json.Marshal(addopt.Node)
	if err != nil {
		return errorx.Wrap(err, "failed to marshal node")
	}
	sig, err := ecdsa.Sign(e.monitor.challengingMonitor.PrivateKey, hash.Hash(s))
	if err != nil {
		return errorx.Wrap(err, "failed to sign node")
	}
	addopt.Signature = sig[:]
	if err := e.chain.AddNode(ctx, &addopt); err != nil {
		if errorx.Is(err, errorx.ErrCodeAlreadyExists) {
			return errorx.New(errorx.ErrCodeAlreadyExists, "duplicated node")
		} else {
			return errorx.Wrap(err, "failed to read blockchain")
		}
	}
	return nil
}

// ListNodes lists storage nodes from blockchain
func (e *Engine) ListNodes(ctx context.Context) (blockchain.Nodes, error) {
	nodes, err := e.chain.ListNodes(ctx)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to read blockchain")
	}
	return nodes, nil
}

// GetNode gets storage node by node id
func (e *Engine) GetNode(ctx context.Context, id []byte) (blockchain.Node, error) {
	node, err := e.chain.GetNode(ctx, id)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return node, errorx.New(errorx.ErrCodeNotFound, "node not found")
		} else {
			return node, errorx.Wrap(err, "failed to read blockchain")
		}
	}
	return node, nil
}

// NodeOffline set storage node status to offline
func (e *Engine) NodeOffline(ctx context.Context, opt types.NodeOfflineOptions) error {
	if err := e.verifyUserID(opt.NodeID); err != nil {
		return err
	}
	sig, err := ecdsa.DecodeSignatureFromString(opt.Token)
	if err != nil {
		return errorx.Wrap(err, "failed to decode signature")
	}

	nodeOpts := &blockchain.NodeOperateOptions{
		NodeID: []byte(opt.NodeID),
		Nonce:  opt.Nonce,
		Sig:    sig[:],
	}
	if err := e.chain.NodeOffline(ctx, nodeOpts); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return errorx.New(errorx.ErrCodeNotFound, "node not found")
		} else {
			return errorx.Wrap(err, "failed to read blockchain")
		}
	}
	return nil
}

// NodeOnline set storage node status to online
func (e *Engine) NodeOnline(ctx context.Context, opt types.NodeOnlineOptions) error {
	if err := e.verifyUserID(opt.NodeID); err != nil {
		return err
	}
	sig, err := ecdsa.DecodeSignatureFromString(opt.Token)
	if err != nil {
		return errorx.Wrap(err, "failed to decode signature")
	}

	nodeOpts := &blockchain.NodeOperateOptions{
		NodeID: []byte(opt.NodeID),
		Nonce:  opt.Nonce,
		Sig:    sig[:],
	}
	if err := e.chain.NodeOnline(ctx, nodeOpts); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return errorx.New(errorx.ErrCodeNotFound, "node not found")
		} else {
			return errorx.Wrap(err, "failed to read blockchain")
		}
	}
	return nil
}

// GetNodeHealth gets storage node health status by node id
func (e *Engine) GetNodeHealth(ctx context.Context, id []byte) (string, error) {
	status, err := e.chain.GetNodeHealth(ctx, id)
	if err != nil {
		return status, errorx.Wrap(err, "failed to get node health status")
	}
	return status, nil
}

// GetSliceMigrateRecords gets storage node slice migration record
func (e *Engine) GetSliceMigrateRecords(ctx context.Context, opt *blockchain.NodeSliceMigrateOptions) (string, error) {
	if _, err := e.chain.GetNode(ctx, opt.Target); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return "", errorx.New(errorx.ErrCodeNotFound, "node not found")
		} else {
			return "", errorx.Wrap(err, "failed to read blockchain")
		}
	}
	result, err := e.chain.GetSliceMigrateRecords(ctx, opt)
	if err != nil {
		return result, errorx.Wrap(err, "failed to get node migrate records")
	}
	return result, nil
}

// GetHeartbeatNum gets storage node heartbeats number of given time
func (e *Engine) GetHeartbeatNum(ctx context.Context, id []byte, ctime int64) (int, int, error) {
	node, err := e.chain.GetNode(ctx, id)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return 0, 0, errorx.New(errorx.ErrCodeNotFound, "node not found")
		} else {
			return 0, 0, errorx.Wrap(err, "failed to read blockchain")
		}
	}
	if ctime != 0 && ctime < node.RegTime {
		return 0, 0, errorx.New(errorx.ErrCodeNotFound, "invalid time, must greater than node register time")
	}
	now := time.Now().UnixNano()
	start, end := common.GetHeartBeatStats(now, node.RegTime)
	heartBeatMax := common.GetHeartbeatMaxNum(start, end, node.RegTime)

	// get heartbeat num of ctime
	if ctime != 0 && ctime >= node.RegTime {
		hearBeatDayNum, err := e.chain.GetHeartbeatNum(ctx, id, common.TodayBeginning(ctime))
		if err != nil && !errorx.Is(err, errorx.ErrCodeNotFound) {
			return 0, 0, err
		}
		if common.TodayBeginning(ctime) == common.TodayBeginning(node.RegTime) {
			start = node.RegTime
		} else {
			start = common.TodayBeginning(ctime)
		}
		if common.TodayBeginning(ctime) == common.TodayBeginning(now) {
			end = now
		} else {
			end = common.TodayBeginning(ctime) + 24*time.Hour.Nanoseconds()
		}
		heartBeatMax = int((end - start) / int64(blockchain.HeartBeatFreq))
		return hearBeatDayNum, heartBeatMax, nil
	}
	// get a series day of heartbeat number total
	heartBeatTotal := 0
	if heartBeatMax != 0 {
		heartBeatTotal, err = common.GetHeartBeatTotalNumByTime(ctx, e.chain, id, start, end)
		if err != nil {
			return 0, 0, err
		}
	}
	return heartBeatTotal, heartBeatMax, nil
}
