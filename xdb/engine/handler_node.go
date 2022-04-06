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
	"fmt"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// ListNodes lists storage nodes from blockchain
func (e *Engine) ListNodes() (blockchain.Nodes, error) {
	nodes, err := e.chain.ListNodes()
	if err != nil {
		return nil, errorx.Wrap(err, "failed to read blockchain")
	}
	return nodes, nil
}

// GetNode gets storage node by node id
func (e *Engine) GetNode(id []byte) (blockchain.Node, error) {
	node, err := e.chain.GetNode(id)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return node, errorx.New(errorx.ErrCodeNotFound, "node not found")
		}
		return node, errorx.Wrap(err, "failed to read blockchain")
	}
	return node, nil
}

// NodeOffline set storage node status to offline
func (e *Engine) NodeOffline(opt types.NodeOfflineOptions) error {
	if err := e.verifyUserIDIsLocalNodeID(opt.NodeID); err != nil {
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
	if err := e.chain.NodeOffline(nodeOpts); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return errorx.New(errorx.ErrCodeNotFound, "node not found")
		}
		return errorx.Wrap(err, "failed to read blockchain")
	}
	return nil
}

// NodeOnline set storage node status to online
func (e *Engine) NodeOnline(opt types.NodeOnlineOptions) error {
	if err := e.verifyUserIDIsLocalNodeID(opt.NodeID); err != nil {
		return err
	}

	m := fmt.Sprintf("%s,%d", opt.NodeID, opt.Nonce)
	h := hash.HashUsingSha256([]byte(m))
	if err := verifyUserToken(opt.NodeID, opt.Token, h); err != nil {
		return err
	}
	sig, err := ecdsa.Sign(e.monitor.challengingMonitor.PrivateKey, h)
	if err != nil {
		return errorx.Wrap(err, "failed to sign node")
	}

	nodeOpts := &blockchain.NodeOperateOptions{
		NodeID: []byte(opt.NodeID),
		Nonce:  opt.Nonce,
		Sig:    sig[:],
	}
	if err := e.chain.NodeOnline(nodeOpts); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return errorx.New(errorx.ErrCodeNotFound, "node not found")
		}
		return errorx.Wrap(err, "failed to read blockchain")
	}
	return nil
}

// GetNodeHealth gets storage node health status by node id
func (e *Engine) GetNodeHealth(id []byte) (string, error) {
	status, err := e.chain.GetNodeHealth(id)
	if err != nil {
		return status, errorx.Wrap(err, "failed to get node health status")
	}
	return status, nil
}

// GetSliceMigrateRecords gets storage node slice migration record
func (e *Engine) GetSliceMigrateRecords(opt *blockchain.NodeSliceMigrateOptions) (string, error) {
	if _, err := e.chain.GetNode(opt.Target); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return "", errorx.New(errorx.ErrCodeNotFound, "node not found")
		}
		return "", errorx.Wrap(err, "failed to read blockchain")
	}
	result, err := e.chain.GetSliceMigrateRecords(opt)
	if err != nil {
		return result, errorx.Wrap(err, "failed to get node migrate records")
	}
	return result, nil
}

// GetHeartbeatNum gets storage node heartbeats number of given time
// The total number of heartbeats is obtained from the blockchain and
// the maximum number of heartbeats is estimated by given time,
// node's heartbeat healthy rate is calculated from the total number and maximum number
func (e *Engine) GetHeartbeatNum(id []byte, ctime int64) (int, int, error) {
	node, err := e.chain.GetNode(id)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return 0, 0, errorx.New(errorx.ErrCodeNotFound, "node not found")
		}
		return 0, 0, errorx.Wrap(err, "failed to read blockchain")
	}
	if ctime != 0 && ctime < node.RegTime {
		return 0, 0, errorx.New(errorx.ErrCodeNotFound, "invalid time, must greater than node register time")
	}
	now := time.Now().UnixNano()
	start, end := common.GetHeartBeatStats(now, node.RegTime)
	heartBeatMax := common.GetHeartbeatMaxNum(start, end, node.RegTime)

	// get heartbeat num of ctime
	if ctime != 0 && ctime >= node.RegTime {
		hearBeatDayNum, err := e.chain.GetHeartbeatNum(id, common.TodayBeginning(ctime))
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
		heartBeatTotal, err = common.GetHeartBeatTotalNumByTime(e.chain, id, start, end)
		if err != nil {
			return 0, 0, err
		}
	}
	return heartBeatTotal, heartBeatMax, nil
}
