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

package xchain

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// AddNode adds a node to xchain
func (x *XChain) AddNode(opt *blockchain.AddNodeOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal AddNodeOptions")
	}
	args := map[string]string{
		"opt": string(s),
	}
	mName := "AddNode"
	if _, err = x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// ListNodes gets all nodes from xchain
func (x *XChain) ListNodes() (blockchain.Nodes, error) {
	var nodes blockchain.Nodes
	args := map[string]string{}
	mName := "ListNodes"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(s, &nodes); err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal nodes")
	}
	return nodes, nil
}

// GetNode gets node by id
func (x *XChain) GetNode(id []byte) (node blockchain.Node, err error) {
	args := map[string]string{
		"id": string(id),
	}
	mName := "GetNode"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return node, err
	}
	if err = json.Unmarshal(s, &node); err != nil {
		return node, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal nodes")
	}
	return node, err
}

// setNodeOnlineStatus sets node status online/offline
func (x *XChain) setNodeOnlineStatus(opt *blockchain.NodeOperateOptions, online bool) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal NodeOperateOptions")
	}
	args := map[string]string{
		"opt": string(s),
	}
	var mName string
	if online {
		mName = "NodeOnline"
	} else {
		mName = "NodeOffline"
	}
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// NodeOffline set node status on chain to offline
func (x *XChain) NodeOffline(opt *blockchain.NodeOperateOptions) error {
	return x.setNodeOnlineStatus(opt, false)
}

// NodeOnline set node status on chain to online
func (x *XChain) NodeOnline(opt *blockchain.NodeOperateOptions) error {
	return x.setNodeOnlineStatus(opt, true)
}

// Heartbeat updates heartbeat of node
func (x *XChain) Heartbeat(id, sig []byte, timestamp int64) error {
	args := map[string]string{
		"id":            string(id),
		"signature":     string(sig),
		"currentTime":   strconv.FormatInt(timestamp, 10),
		"beginningTime": strconv.FormatInt(common.TodayBeginning(timestamp), 10),
	}
	mName := "Heartbeat"
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// GetHeartbeatNum gets heartbeat number by time
func (x *XChain) GetHeartbeatNum(id []byte, timestamp int64) (int, error) {
	args := map[string]string{
		"id":          string(id),
		"currentTime": strconv.FormatInt(common.TodayBeginning(timestamp), 10),
	}
	mName := "GetHeartbeatNum"
	number, err := x.QueryContract(args, mName)
	if err != nil {
		return 0, err
	}
	num, err := strconv.Atoi(string(number))
	if err != nil {
		return 0, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal heartbeat number")
	}
	return num, nil
}

// GetSliceMigrateRecords get node slice migration records
func (x *XChain) GetSliceMigrateRecords(opt *blockchain.NodeSliceMigrateOptions) (string, error) {
	s, err := json.Marshal(*opt)
	if err != nil {
		return "", errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal NodeSliceMigrateOptions")
	}
	args := map[string]string{
		"opt": string(s),
	}
	mName := "GetSliceMigrateRecords"
	sm, err := x.QueryContract(args, mName)
	if err != nil {
		return "", err
	}
	return string(sm), nil
}

// ListNodesExpireSlice lists expired slices from xchain
func (x *XChain) ListNodesExpireSlice(opt *blockchain.ListNodeSliceOptions) ([]string, error) {
	var sliceL []string

	opts, err := json.Marshal(*opt)
	if err != nil {
		return sliceL, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal ListFileOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ListNodesExpireSlice"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return sliceL, err
	}
	if err = json.Unmarshal(s, &sliceL); err != nil {
		return sliceL, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal Files")
	}
	// convert ['a,d','b,c'] to ['a','d','b','c']
	arrs := strings.Join(sliceL, ",")
	return strings.Split(arrs, ","), nil
}

// GetNodeHealth gets node health status
func (x *XChain) GetNodeHealth(id []byte) (string, error) {
	now := time.Now().UnixNano()
	node, err := x.GetNode(id)
	if err != nil {
		return "", err
	}

	start, end := common.GetHeartBeatStats(now, node.RegTime)

	// get proved challenges ratio
	numOpt := blockchain.GetChallengeNumOptions{
		TargetNode: id,
		TimeStart:  start,
		TimeEnd:    end,
	}
	all, err := x.GetChallengeNum(&numOpt)
	if err != nil {
		return "", err
	}
	provedRation := blockchain.DefaultChallProvedRate
	if all != 0 {
		numOpt.Status = blockchain.ChallengeProved
		proved, err := x.GetChallengeNum(&numOpt)
		if err != nil {
			return "", err
		}
		provedRation = float64(proved) / float64(all)
	}
	// get hearbeat max number
	heartBeaMax := common.GetHeartbeatMaxNum(start, end, node.RegTime)

	heartBeatRate := blockchain.DefaultHearBeatRate
	if heartBeaMax != 0 {
		hearBeatTotal, err := common.GetHeartBeatTotalNumByTime(x, id, start, end)
		if err != nil {
			return "", err
		}
		heartBeatRate = float64(hearBeatTotal) / float64(heartBeaMax)
	}
	// calculate health index
	health := blockchain.NodeHealthChallProp*provedRation + blockchain.NodeHealthHeartBeatProp*heartBeatRate
	if health >= blockchain.NodeHealthBoundGood {
		return blockchain.NodeHealthGood, nil
	}
	if health < blockchain.NodeHealthBoundMedium {
		return blockchain.NodeHealthBad, nil
	}
	return blockchain.NodeHealthMedium, nil
}
