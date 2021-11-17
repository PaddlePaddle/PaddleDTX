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

package mock

import (
	"encoding/hex"
	"fmt"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
)

const (
	prefixFilenameIndex         = "index_fn"
	prefixChallengeIndex4Owner  = "index_cho"
	prefixChallengeIndex4Target = "index_cht"
	prefixNodeIndex             = "index_node"
)

func mustToHex(s string) []byte {
	bs, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return bs
}

func packNodeIndex(node blockchain.Node) string {
	return fmt.Sprintf("%s/%x", prefixNodeIndex, node.ID)
}

func packFilenameIndex(owner []byte, ns, name string) string {
	return fmt.Sprintf("%s/%x/%s/%s", prefixFilenameIndex, owner, ns, name)
}

func packChallengeIndex4Owner(c *blockchain.Challenge) string {
	return fmt.Sprintf("%s/%x/%x/%s/%x/%x", prefixChallengeIndex4Owner,
		c.FileOwner, c.TargetNode, c.FileID, c.Indices, c.Vs)
}

func packChallengeIndex4Target(c *blockchain.Challenge) string {
	return fmt.Sprintf("%s/%x/%s/%x/%x", prefixChallengeIndex4Target,
		c.TargetNode, c.FileID, c.Indices, c.Vs)
}

func packFilenameFilter(owner []byte, ns string) string {
	filter := prefixFilenameIndex
	if len(owner) > 0 {
		filter += fmt.Sprintf("/%x", owner)
		if len(ns) > 0 {
			filter += fmt.Sprintf("/%s", ns)
		}
	}

	return filter
}

func packChallengeFilter(owner, target []byte) string {
	prefix := prefixChallengeIndex4Owner // default
	if len(owner) == 0 && len(target) > 0 {
		prefix = prefixChallengeIndex4Target
	}

	if len(owner) > 0 {
		prefix += fmt.Sprintf("/%x", owner)
	}
	if len(target) > 0 {
		prefix += fmt.Sprintf("/%x", target)
	}

	return prefix
}

type indexTuple struct {
	Index string
	ID    string
}

type sortableIndexTuple []indexTuple

func (t sortableIndexTuple) Len() int { return len(t) }

func (t sortableIndexTuple) Less(i, j int) bool {
	return t[i].Index < t[j].Index
}

func (t sortableIndexTuple) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
