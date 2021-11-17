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
	"context"
	"fmt"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

func (mc *MockChain) ListNodes(ctx context.Context) (blockchain.Nodes, error) {
	var nodes blockchain.Nodes
	for _, n := range mc.Nodes {
		nodes = append(nodes, n)
	}

	return nodes, nil
}

func (mc *MockChain) AddNode(ctx context.Context, opt *blockchain.AddNodeOptions) error {
	index := packNodeIndex(opt.Node)
	mc.Nodes[index] = opt.Node
	return nil
}

func (mc *MockChain) GetNode(ctx context.Context, id []byte) (node blockchain.Node, err error) {
	index := fmt.Sprintf("%s/%x", prefixNodeIndex, id)
	node, ok := mc.Nodes[index]
	if !ok {
		return node, errorx.New(errorx.ErrCodeNotFound, "node not found")
	}
	return node, nil
}

func (mc *MockChain) NodeOffline(ctx context.Context, opt *blockchain.NodeOperateOptions) error {
	index := fmt.Sprintf("%s/%x", prefixNodeIndex, opt.NodeID)
	node, ok := mc.Nodes[index]
	if !ok {
		return errorx.New(errorx.ErrCodeNotFound, "node not found")
	}
	node.Online = false
	mc.Nodes[index] = node
	return nil
}

func (mc *MockChain) NodeOnline(ctx context.Context, opt *blockchain.NodeOperateOptions) error {
	index := fmt.Sprintf("%s/%x", prefixNodeIndex, opt.NodeID)
	node, ok := mc.Nodes[index]
	if !ok {
		return errorx.New(errorx.ErrCodeNotFound, "node not found")
	}
	node.Online = true
	mc.Nodes[index] = node
	return nil
}

func (mc *MockChain) Heartbeat(ctx context.Context, id, sig []byte, timestamp int64) error {
	index := fmt.Sprintf("%s/%x", prefixNodeIndex, id)
	node, ok := mc.Nodes[index]
	if !ok {
		return errorx.New(errorx.ErrCodeNotFound, "node not found")
	}
	node.UpdateAt = timestamp
	mc.Nodes[index] = node
	return nil
}

func (mc *MockChain) GetSliceMigrateRecords(ctx context.Context, opt *blockchain.NodeSliceMigrateOptions) (string, error) {
	return "", nil
}

// 获取最新的区块高度
func (mc *MockChain) GetRootAndLatestBlockIdInChain() ([]byte, int64, error) {
	return nil, 0, errorx.New(errorx.ErrCodeInternal, "can't find blockid")
}

// GetNodeHealth get node health status
func (mc *MockChain) GetNodeHealth(ctx context.Context, id []byte) (string, error) {
	return blockchain.NodeHealthGood, nil
}

func (mc *MockChain) ListNodesExpireSlice(ctx context.Context, opt *blockchain.ListNodeSliceOptions) ([]string, error) {
	return nil, errorx.New(errorx.ErrCodeInternal, "not implemented")
}

func (mc *MockChain) GetHeartbeatNum(ctx context.Context, id []byte, timestamp int64) (int, error) {
	return 0, nil
}
