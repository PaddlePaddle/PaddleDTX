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

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
)

// RegisterExecutorNode registers Executor node to xchain
func (x *XChain) RegisterExecutorNode(opt *blockchain.AddNodeOptions) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal AddNodeOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "RegisterExecutorNode"
	if _, err = x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// ListExecutorNodes gets all Executor nodes from xchain
func (x *XChain) ListExecutorNodes() (blockchain.ExecutorNodes, error) {
	var nodes blockchain.ExecutorNodes
	args := map[string]string{}
	mName := "ListExecutorNodes"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(s, &nodes); err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal executor nodes")
	}
	return nodes, nil
}

// GetExecutorNodeByID gets Executor node by ID
func (x *XChain) GetExecutorNodeByID(id []byte) (node blockchain.ExecutorNode, err error) {
	args := map[string]string{
		"id": string(id),
	}
	mName := "GetExecutorNodeByID"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return node, err
	}
	if err = json.Unmarshal(s, &node); err != nil {
		return node, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal executor node")
	}
	return node, err
}

// GetExecutorNodeByName gets Executor node by name
func (x *XChain) GetExecutorNodeByName(name string) (node blockchain.ExecutorNode, err error) {
	args := map[string]string{
		"name": name,
	}
	mName := "GetExecutorNodeByName"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return node, err
	}
	if err = json.Unmarshal(s, &node); err != nil {
		return node, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal executor node")
	}
	return node, err
}
