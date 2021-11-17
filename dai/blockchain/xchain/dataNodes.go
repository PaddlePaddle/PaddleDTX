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
	"context"
	"encoding/json"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
)

// RegisterDataNode registers Executor node to xchain
func (x *XChain) RegisterDataNode(ctx context.Context,
	opt *blockchain.AddNodeOptions) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal AddNodeOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "RegisterDataNode"
	if _, err = x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// ListDataNodes gets all Executor nodes from from xchain
func (x *XChain) ListDataNodes(ctx context.Context) (blockchain.DataNodes, error) {
	var nodes blockchain.DataNodes
	args := map[string]string{}
	mName := "ListDataNodes"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(s, &nodes); err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal data nodes")
	}
	return nodes, nil
}

// GetDataNodeByID gets Executor node by ID
func (x *XChain) GetDataNodeByID(ctx context.Context, id []byte) (node blockchain.DataNode, err error) {
	args := map[string]string{
		"id": string(id),
	}
	mName := "GetDataNodeByID"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return node, err
	}
	if err = json.Unmarshal(s, &node); err != nil {
		return node, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal data nodes")
	}
	return node, err
}
