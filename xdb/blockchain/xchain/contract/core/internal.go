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

package core

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
)

const (
	prefixFilenameIndex         = "index_fn"
	prefixFilenameListIndex     = "index_fn_list"
	prefixFileNsIndex           = "index_fns"
	prefixFileNsListIndex       = "index_fns_list"
	prefixChallenge             = "index_chal"
	prefixChallengeIndex4Owner  = "index_cho"
	prefixChallengeIndex4Target = "index_cht"
	prefixNodeIndex             = "index_node"
	prefixNodeListIndex         = "index_node_list"
	prefixNodeHeartbeatIndex    = "index_hbnode"
	prefixNodeSliceMigrateIndex = "index_slicemigrate"
	prefixNodeFileSlice         = "index_fslice"
	prefixNodeNonceIndex        = "index_ndnonce"
)

func packNodeIndex(nodeID []byte) string {
	return fmt.Sprintf("%s/%x", prefixNodeIndex, nodeID)
}

func packNodeListIndex(node blockchain.Node) string {
	return fmt.Sprintf("%s/%d/%x", prefixNodeListIndex, subByInt64Max(node.RegTime), node.ID)
}

func packNodeHeartBeatIndex(nodeID []byte, ctime int64) string {
	return fmt.Sprintf("%s/%x/%d", prefixNodeHeartbeatIndex, nodeID, ctime)
}

func packNonceIndex(node []byte, nonce int64) string {
	return fmt.Sprintf("%s/%x/%d", prefixNodeNonceIndex, node, nonce)
}

func packNodeSliceIndex(node string, f blockchain.File) string {
	return fmt.Sprintf("%s/%s/%d/%s", prefixNodeFileSlice, node, f.ExpireTime, f.ID)
}

func packNodeSliceFilter(target string) string {
	return fmt.Sprintf("%s/%s/", prefixNodeFileSlice, target)
}

//  example: string(key) = index_fslice/7935e6d162b2eed64/1625039335453720000/fileid11
func getNodeSliceFileId(key []byte) (string, int64) {
	str_arr := strings.Split(string(key), "/")
	expireTime, err := strconv.ParseInt(str_arr[2], 10, 64)
	if err != nil {
		return "", 0
	}
	return str_arr[len(str_arr)-1], expireTime
}

func packNodeSliceMigrateIndex(target string, ctime int64) string {
	return fmt.Sprintf("%s/%s/%d", prefixNodeSliceMigrateIndex, target, subByInt64Max(ctime))
}

func packNodeSliceMigrateFilter(target string) string {
	return fmt.Sprintf("%s/%s/", prefixNodeSliceMigrateIndex, target)
}

// example: string(key) = index_slicemirgate/7935e6d162b2eed64/1625039335453720000
func getNodeSliceMigrateTime(key []byte) int64 {
	str_arr := strings.Split(string(key), "/")
	expireTime, err := strconv.ParseInt(str_arr[2], 10, 64)
	if err != nil {
		return 0
	}
	return subByInt64Max(expireTime)
}

func packFileNameIndex(owner []byte, ns, name string) string {
	return fmt.Sprintf("%s/%x/%s/%s", prefixFilenameIndex, owner, ns, name)
}

func packFileNameListIndex(owner []byte, ns, name string, pubTime int64) string {
	return fmt.Sprintf("%s/%x/%s/%d/%s", prefixFilenameListIndex, owner, ns, subByInt64Max(pubTime), name)
}

func packFileNsIndex(owner []byte, ns string) string {
	return fmt.Sprintf("%s/%x/%s", prefixFileNsIndex, owner, ns)
}

func packFileNsListIndex(owner []byte, ns string, createTime int64) string {
	return fmt.Sprintf("%s/%x/%d/%s", prefixFileNsListIndex, owner, subByInt64Max(createTime), ns)
}

func packFileNsListFilter(owner []byte) string {
	return prefixFileNsListIndex + "/" + fmt.Sprintf("%x/", owner)
}

func packFileNameFilter(owner []byte, ns string) string {
	filter := prefixFilenameListIndex + "/" + fmt.Sprintf("%x/", owner)
	if len(ns) > 0 {
		filter += fmt.Sprintf("%s/", ns)
	}
	return filter
}

func packChallengeIndex(id string) string {
	return fmt.Sprintf("%s/%s", prefixChallenge, id)
}

func packChallengeIndex4Owner(c *blockchain.Challenge) string {
	return fmt.Sprintf("%s/%x/%x/%d/%s", prefixChallengeIndex4Owner, c.FileOwner, c.TargetNode, subByInt64Max(c.ChallengeTime), c.ID)
}

func packChallengeIndex4Target(c *blockchain.Challenge) string {
	return fmt.Sprintf("%s/%x/%d/%s", prefixChallengeIndex4Target, c.TargetNode, subByInt64Max(c.ChallengeTime), c.ID)
}

func packChallengeFilter(owner, target []byte) string {
	prefix := prefixChallengeIndex4Owner + "/" // default

	if len(owner) == 0 && len(target) > 0 {
		prefix = prefixChallengeIndex4Target + "/"
	}

	if len(owner) > 0 {
		prefix += fmt.Sprintf("%x/", owner)
	}
	if len(target) > 0 {
		prefix += fmt.Sprintf("%x/", target)
	}

	return prefix
}

// return maxInt64 - N
func subByInt64Max(n int64) int64 {
	return math.MaxInt64 - n
}
