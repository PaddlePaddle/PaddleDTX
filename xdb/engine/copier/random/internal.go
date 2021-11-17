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

package random

import (
	"math/rand"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
)

// getSliceOptimalNodes get healthy node to store a slice, green node first
func getSliceOptimalNodes(nodeHs blockchain.NodeHs, replica int) (nodes blockchain.Nodes) {
	var yellowNodeList blockchain.Nodes

	for _, n := range nodeHs {
		if n.Health == blockchain.NodeHealthGood {
			nodes = append(nodes, n.Node)
		}
		if n.Health == blockchain.NodeHealthMedium {
			yellowNodeList = append(yellowNodeList, n.Node)
		}
	}
	// if green nodes > replica
	if len(nodes) >= replica {
		return nodes
	}
	// if (green nodes + yellow nodes) < replica, merge green and yellow node
	// if green nodes is 0, nodes is all yellow
	if len(nodes) == 0 || len(nodes)+len(yellowNodeList) <= replica {
		nodes = append(nodes, yellowNodeList...)
	} else {
		// if (green nodes + yellow nodes) > replica, select all green node and random select different yellow node
		syNodes := replica - len(nodes)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		ymn := make(map[string]struct{})
		for {
			index := r.Intn(len(yellowNodeList))
			if len(ymn) == syNodes {
				break
			}
			if _, exist := ymn[string(yellowNodeList[index].ID)]; exist {
				continue
			}
			ymn[string(yellowNodeList[index].ID)] = struct{}{}
			nodes = append(nodes, yellowNodeList[index])
		}
	}
	return nodes
}

// bSearch must exist
func bSearch(nodes []nodeWeight, target uint64) []byte {
	i, j := 0, len(nodes)
	for i <= j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		if nodes[h].start <= target && nodes[h].end >= target {
			return nodes[h].nodeID
		}

		if nodes[h].end < target {
			i = h + 1
		} else {
			j = h
		}
	}

	panic("should not happen")
}
