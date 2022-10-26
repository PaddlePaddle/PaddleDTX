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

//参考dai下面xchain的调用

package fabric

import (
	"encoding/json"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
)

// RegisterExecutorNode registers Executor node to fabric
func (f *Fabric) RegisterExecutorNode(opt *blockchain.AddNodeOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal AddNodeOptions")
	}
	args := [][]byte{s}
	mName := "RegisterExecutorNode"

	if _, err = f.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// ListExecutorNodes gets all Executor nodes from fabric
func (f *Fabric) ListExecutorNodes() (blockchain.ExecutorNodes, error) {
	var nodes blockchain.ExecutorNodes

	mName := "ListExecutorNodes"
	s, err := f.QueryContract([][]byte{}, mName)
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
func (f *Fabric) GetExecutorNodeByID(id string) (node blockchain.ExecutorNode, err error) {
	mName := "GetExecutorNodeByID"
	s, err := f.QueryContract([][]byte{[]byte(id)}, mName)
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
func (f *Fabric) GetExecutorNodeByName(name string) (node blockchain.ExecutorNode, err error) {
	args := [][]byte{[]byte(name)}
	mName := "GetExecutorNodeByName"
	s, err := f.QueryContract(args, mName)
	if err != nil {
		return node, err
	}
	if err = json.Unmarshal(s, &node); err != nil {
		return node, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal executor node")
	}
	return node, err
}
