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

package fabric

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// AddNode adds a node to fabric
func (f *Fabric) AddNode(opt *blockchain.AddNodeOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal AddNodeOptions")
	}
	args := [][]byte{s}
	if _, err = f.InvokeContract(args, "AddNode"); err != nil {
		return err
	}
	return nil
}

// ListNodes gets all nodes from fabric
func (f *Fabric) ListNodes() (blockchain.Nodes, error) {
	var nodes blockchain.Nodes
	s, err := f.QueryContract([][]byte{}, "ListNodes")
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
func (f *Fabric) GetNode(id []byte) (node blockchain.Node, err error) {
	s, err := f.QueryContract([][]byte{id}, "GetNode")
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
func (f *Fabric) setNodeOnlineStatus(opt *blockchain.NodeOperateOptions, online bool) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal NodeOperateOptions")
	}
	var mName string
	if online {
		mName = "NodeOnline"
	} else {
		mName = "NodeOffline"
	}
	if _, err := f.InvokeContract([][]byte{s}, mName); err != nil {
		return err
	}
	return nil
}

// NodeOffline set node status on chain to offline
func (f *Fabric) NodeOffline(opt *blockchain.NodeOperateOptions) error {
	return f.setNodeOnlineStatus(opt, false)
}

// NodeOnline set node status on chain to online
func (f *Fabric) NodeOnline(opt *blockchain.NodeOperateOptions) error {
	return f.setNodeOnlineStatus(opt, true)
}

// Heartbeat updates heartbeat of node
func (f *Fabric) Heartbeat(id, sig []byte, timestamp int64) error {
	args := [][]byte{id, sig, []byte(strconv.FormatInt(timestamp, 10)), []byte(
		strconv.FormatInt(common.TodayBeginning(timestamp), 10))}
	if _, err := f.InvokeContract(args, "Heartbeat"); err != nil {
		return err
	}
	return nil
}

// GetHeartbeatNum gets heartbeat number by time
func (f *Fabric) GetHeartbeatNum(id []byte, timestamp int64) (int, error) {
	args := [][]byte{id, []byte(strconv.FormatInt(common.TodayBeginning(timestamp), 10))}
	number, err := f.QueryContract(args, "GetHeartbeatNum")
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
func (f *Fabric) GetSliceMigrateRecords(opt *blockchain.NodeSliceMigrateOptions) (string, error) {
	s, err := json.Marshal(*opt)
	if err != nil {
		return "", errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal NodeSliceMigrateOptions")
	}

	sm, err := f.QueryContract([][]byte{s}, "GetSliceMigrateRecords")
	if err != nil {
		return "", err
	}
	return string(sm), nil
}

// ListNodesExpireSlice lists expired slices from fabric
func (f *Fabric) ListNodesExpireSlice(opt *blockchain.ListNodeSliceOptions) ([]string, error) {
	var sliceL []string

	opts, err := json.Marshal(*opt)
	if err != nil {
		return sliceL, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal ListFileOptions")
	}

	s, err := f.QueryContract([][]byte{opts}, "ListNodesExpireSlice")
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
func (f *Fabric) GetNodeHealth(id []byte) (string, error) {
	now := time.Now().UnixNano()
	node, err := f.GetNode(id)
	if err != nil {
		return "", err
	}
	// get heartbeat start and end time
	start, end := common.GetHeartBeatStats(now, node.RegTime)

	// get proved challenges ratio
	numOpt := blockchain.GetChallengeNumOptions{
		TargetNode: id,
		TimeStart:  start,
		TimeEnd:    end,
	}
	all, err := f.GetChallengeNum(&numOpt)
	if err != nil {
		return "", err
	}
	provedRation := blockchain.DefaultChallProvedRate
	if all != 0 {
		numOpt.Status = blockchain.ChallengeProved
		proved, err := f.GetChallengeNum(&numOpt)
		if err != nil {
			return "", err
		}

		provedRation = float64(proved) / float64(all)
	}
	// get heartbeat rate
	heartBeatMax := common.GetHeartbeatMaxNum(start, end, node.RegTime)

	heartBeatRate := blockchain.DefaultHearBeatRate
	if heartBeatMax != 0 {
		hearBeatTotal, err := common.GetHeartBeatTotalNumByTime(f, id, start, end)
		if err != nil {
			return "", err
		}

		heartBeatRate = float64(hearBeatTotal) / float64(heartBeatMax)
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
