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

package common

import (
	"sync"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// ToNodesMap make a map from node id to node
func ToNodesMap(nodes blockchain.Nodes) map[string]blockchain.Node {
	nodesMap := make(map[string]blockchain.Node)
	for _, n := range nodes {
		if n.Online {
			nodesMap[string(n.ID)] = n
		}
	}

	return nodesMap
}

// ToNodeHsMap make a map from node id to node with health status
func ToNodeHsMap(nodes blockchain.NodeHs) map[string]blockchain.Node {
	nodesMap := make(map[string]blockchain.Node)
	for _, n := range nodes {
		if n.Node.Online {
			nodesMap[string(n.Node.ID)] = n.Node
		}
	}

	return nodesMap
}

// GetHeartbeatNum get heart beat number of a storage node from blockchain
func GetHeartbeatNum(chain CommonChain, id []byte, ts []int64) (int, error) {
	hearBeatTotal := 0
	wg := sync.WaitGroup{}
	wg.Add(len(ts))

	var hErr error
	cts := make(chan int, len(ts))
	for _, t := range ts {
		// concurrent to get heartbeat num
		go func(t int64) {
			defer wg.Done()
			n, err := chain.GetHeartbeatNum(id, t)
			if err != nil && !errorx.Is(err, errorx.ErrCodeNotFound) {
				hErr = err
				return
			}
			cts <- n
		}(t)
	}
	wg.Wait()
	close(cts)
	for i := range cts {
		hearBeatTotal += i
	}
	if hErr != nil {
		return hearBeatTotal, hErr
	}
	return hearBeatTotal, nil
}

// TodayBeginning convert a timestamp to 00:00:00 of this day
func TodayBeginning(timestamp int64) int64 {
	t := time.Unix(0, timestamp)
	dayTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return dayTime.UnixNano()
}

// GetHeartBeatStats get heart beat statistics during start and end time
func GetHeartBeatStats(now, regTime int64) (int64, int64) {
	// yesterday
	end := TodayBeginning(now) - int64(24*time.Hour)
	// seven days ago
	start := end - int64((blockchain.NodeHealthTimeDur-1)*24*time.Hour)
	if start < regTime {
		start = regTime
		end = now
	}
	return start, end
}

// GetHeartBeatTotalNumByTime get total heart beat number of a storage node during given time period
func GetHeartBeatTotalNumByTime(chain CommonChain, id []byte, start, end int64) (int, error) {
	t := start

	var ts []int64
	for t <= end {
		t = TodayBeginning(t)
		ts = append(ts, t)
		t += int64(24 * time.Hour)
	}
	heartBeatTotal, err := GetHeartbeatNum(chain, id, ts)
	if err != nil && !errorx.Is(err, errorx.ErrCodeNotFound) {
		return 0, err
	}
	return heartBeatTotal, nil
}

// GetHeartbeatMaxNum calculate the max possible heart beat number given a time period
func GetHeartbeatMaxNum(start, end, regTime int64) int {
	// get heartbeat max
	heartBeatMax := blockchain.NodeHealthTimeDur * blockchain.HeartBeatPerDay
	// if register time is not enough 7 days, max heart beat num is from reg to now
	if start == regTime {
		heartBeatMax = int((end - start) / int64(blockchain.HeartBeatFreq))
	}
	return heartBeatMax
}
