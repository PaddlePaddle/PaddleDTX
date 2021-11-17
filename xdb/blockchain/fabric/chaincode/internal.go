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

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
)

const (
	compositeKeyNamespace = "\x00"
	minUnicodeRuneValue   = 0

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
	//return fmt.Sprintf("%s/%x", prefixNodeIndex, nodeID)
	attributes := []string{fmt.Sprintf("%x", nodeID)}
	return createCompositeKey(prefixNodeIndex, attributes)
}

func packNodeListIndex(node blockchain.Node) string {
	//return fmt.Sprintf("%s/%d/%x", prefixNodeListIndex, subByInt64Max(node.RegTime), node.ID)
	attributes := []string{fmt.Sprintf("%d", subByInt64Max(node.RegTime)), fmt.Sprintf("%x", node.ID)}
	return createCompositeKey(prefixNodeListIndex, attributes)
}

func packHeartBeatIndex(nodeID []byte, ctime int64) string {
	attr := []string{fmt.Sprintf("%x", nodeID), fmt.Sprintf("%d", ctime)}
	return createCompositeKey(prefixNodeHeartbeatIndex, attr)
}

func packNonceIndex(node []byte, nonce int64) string {
	//return fmt.Sprintf("%s/%x/%d", prefixNodeNonceIndex, node, nonce)
	return createCompositeKey(prefixNodeNonceIndex, []string{fmt.Sprintf("%x", node), fmt.Sprintf("%d", nonce)})
}

func packNodeSliceIndex(node string, f blockchain.File) string {
	//return fmt.Sprintf("%s/%s/%d/%s", prefixNodeFileSlice, node, f.ExpireTime, f.ID)
	attributes := []string{node, fmt.Sprintf("%d", f.ExpireTime), f.ID}
	return createCompositeKey(prefixNodeFileSlice, attributes)
}

func packNodeSliceFilter(target string) (string, []string) {
	return prefixNodeFileSlice, []string{target}
}

// example: string(key) = \x00 index_fslice/ 0 node_id 0 1625039335453720000 0 fileid11 0
func getNodeSliceFileId(key []byte) int64 {
	str_arr := strings.Split(string(key), string(minUnicodeRuneValue))
	expireTime, err := strconv.ParseInt(str_arr[3], 10, 64)
	if err != nil {
		return 0
	}
	return expireTime
}

func packNodeSliceMigrateIndex(target string, ctime int64) string {
	//return fmt.Sprintf("%s/%s/%d", prefixNodeSliceMigrateIndex, target, subByInt64Max(ctime))
	attributes := []string{target, fmt.Sprintf("%d", subByInt64Max(ctime))}
	return createCompositeKey(prefixNodeSliceMigrateIndex, attributes)
}

func packNodeSliceMigrateFilter(nodeID string) (string, []string) {
	//return fmt.Sprintf("%s/%s/", prefixNodeSliceMigrateIndex, target)
	return prefixNodeSliceMigrateIndex, []string{nodeID}
}

func getNodeSliceMigrateTime(key []byte) int64 {
	str_arr := strings.Split(string(key), string(minUnicodeRuneValue))
	expireTime, err := strconv.ParseInt(str_arr[3], 10, 64)
	if err != nil {
		return 0
	}
	return subByInt64Max(expireTime)
}

func packFileNameIndex(owner []byte, ns, name string) string {
	//return fmt.Sprintf("%s/%x/%s/%s", prefixFilenameIndex, owner, ns,  name)
	attributes := []string{fmt.Sprintf("%x", owner), ns, name}
	return createCompositeKey(prefixFilenameIndex, attributes)
}

func packFileNameListIndex(owner []byte, ns, name string, pubTime int64) string {
	//return fmt.Sprintf("%s/%x/%s/%d/%s", prefixFilenameListIndex, owner, ns, subByInt64Max(pubTime), name)
	attributes := []string{fmt.Sprintf("%x", owner), ns, fmt.Sprintf("%d", subByInt64Max(pubTime)), name}
	return createCompositeKey(prefixFilenameListIndex, attributes)
}

func packFileNsIndex(owner []byte, ns string) string {
	//return fmt.Sprintf("%s/%x/%s", prefixFileNsIndex, owner, ns)
	attributes := []string{fmt.Sprintf("%x", owner), ns}
	return createCompositeKey(prefixFileNsIndex, attributes)
}

func packFileNsListIndex(owner []byte, ns string, createTime int64) string {
	//return fmt.Sprintf("%s/%x/%d/%s", prefixFileNsListIndex, owner, subByInt64Max(createTime), ns)
	attributes := []string{fmt.Sprintf("%x", owner), fmt.Sprintf("%d", subByInt64Max(createTime)), ns}
	return createCompositeKey(prefixFileNsListIndex, attributes)
}

// prefixFileNsListIndex + "/" + fmt.Sprintf("%x/", owner)
func packFileNsListFilter(owner []byte) (prefix string, attr []string) {
	//return prefixFileNsListIndex + "/" + fmt.Sprintf("%x/", owner)
	return prefixFileNsListIndex, []string{fmt.Sprintf("%x", owner)}
}

func packFileNameFilter(owner []byte, ns string) (prefix string, attr []string) {
	prefix = prefixFilenameListIndex
	attr = []string{fmt.Sprintf("%x", owner)}
	if len(ns) > 0 {
		attr = append(attr, ns)
	}
	return prefix, attr
}

func packChallengeIndex(id string) string {
	//return fmt.Sprintf("%s/%s", prefixChallenge, id)
	return createCompositeKey(prefixChallenge, []string{id})
}

func packChallengeIndex4Owner(c *blockchain.Challenge) string {
	//return fmt.Sprintf("%s/%x/%x/%d/%s", prefixChallengeIndex4Owner, c.FileOwner, c.TargetNode, subByInt64Max(c.ChallengeTime), c.ID)
	attributes := []string{fmt.Sprintf("%x", c.FileOwner), fmt.Sprintf("%x", c.TargetNode), fmt.Sprintf("%d", subByInt64Max(c.ChallengeTime)), c.ID}
	return createCompositeKey(prefixChallengeIndex4Owner, attributes)
}

func packChallengeIndex4Target(c *blockchain.Challenge) string {
	//return fmt.Sprintf("%s/%x/%d/%s", prefixChallengeIndex4Target, c.TargetNode, subByInt64Max(c.ChallengeTime), c.ID)
	attributes := []string{fmt.Sprintf("%x", c.TargetNode), fmt.Sprintf("%d", subByInt64Max(c.ChallengeTime)), c.ID}
	return createCompositeKey(prefixChallengeIndex4Target, attributes)
}

func packChallengeFilter(owner, target []byte) (prefix string, attr []string) {
	prefix = prefixChallengeIndex4Owner
	if len(owner) == 0 && len(target) > 0 {
		prefix = prefixChallengeIndex4Target
	}

	if len(owner) > 0 {
		attr = []string{fmt.Sprintf("%x", owner)}
	}
	if len(target) > 0 {
		attr = append(attr, fmt.Sprintf("%x", target))
	}
	return prefix, attr
}

func createCompositeKey(objectType string, attributes []string) string {
	ck := compositeKeyNamespace + objectType + string(minUnicodeRuneValue)
	for _, att := range attributes {
		ck += att + string(minUnicodeRuneValue)
	}
	return ck
}

// return maxInt64 - N
func subByInt64Max(n int64) int64 {
	return math.MaxInt64 - n
}
