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
	"encoding/json"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
)

// RegisterDataNode registers Executor node
func (x *Xdata) RegisterDataNode(ctx code.Context) code.Response {
	var opt blockchain.AddNodeOptions
	// get opt
	p, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:node"))
	}
	// unmarshal opt
	if err := json.Unmarshal(p, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal AddNodeOptions"))
	}
	// get node
	node := opt.Node
	// marshal node
	s, err := json.Marshal(node)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal Node"))
	}
	// verify sig
	if err := x.checkSign(opt.Signature, node.ID, s); err != nil {
		return code.Error(err)
	}

	// put index-node on xchain, judge if index exists
	index := packNodeIndex(node.ID)
	if _, err := ctx.GetObject([]byte(index)); err == nil {
		return code.Error(errorx.New(errorx.ErrCodeAlreadyExists,
			"duplicated nodeID"))
	}
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"fail to put index-Node on xchain"))
	}

	// put listIndex-node on xchain
	index = packNodeListIndex(node)
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"fail to put listIndex-Node on xchain"))
	}
	return code.OK([]byte("added"))
}

// ListDataNodes gets all Executor nodes
func (x *Xdata) ListDataNodes(ctx code.Context) code.Response {
	var nodes blockchain.DataNodes

	// get data nodes by list_prefix
	prefix := prefixNodeListIndex
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	for iter.Next() {
		var node blockchain.DataNode
		if err := json.Unmarshal(iter.Value(), &node); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"fail to unmarshal node"))
		}
		nodes = append(nodes, node)
	}
	// marshal nodes
	s, err := json.Marshal(nodes)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal nodes"))
	}
	return code.OK(s)
}

// GetDataNodeByID gets Executor node by ID
func (x *Xdata) GetDataNodeByID(ctx code.Context) code.Response {
	// get id
	nodeID, ok := ctx.Args()["id"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}

	// get node by index
	index := packNodeIndex(nodeID)
	s, err := ctx.GetObject([]byte(index))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "node not found"))
	}
	return code.OK(s)
}
