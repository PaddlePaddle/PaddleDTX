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
	"regexp"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	util "github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/strings"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
)

// RegisterExecutorNode registers Executor node
func (x *Xdata) RegisterExecutorNode(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var opt blockchain.AddNodeOptions
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting AddNodeOptions")
	}
	// unmarshal opt
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal AddNodeOptions").Error())
	}

	// get node
	n := opt.Node
	// marshal node
	s, err := json.Marshal(n)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Node").Error())
	}

	// check node.Name, the length of the executor node name is 4-16 characters
	// only support lowercase letters and numbers
	if ok, _ := regexp.MatchString("^[a-z0-9]{4,16}", n.Name); !ok {
		return shim.Error(errorx.New(errorx.ErrCodeParam,
			"bad param, nodeName only supports numbers and lowercase letters with a length of 4-16").Error())
	}

	// get the message to sign
	msg, err := util.GetSigMessage(opt)
	if err != nil {
		return shim.Error(errorx.Internal(err, "failed to get the message to sign").Error())
	}

	// verify sig
	if err := x.checkSign(opt.Signature, n.ID, []byte(msg)); err != nil {
		return shim.Error(err.Error())
	}

	// put index-node on xchain, judge if index exists
	index := packNodeIndex(n.ID)
	if resp := x.GetValue(stub, []string{index}); len(resp.Payload) != 0 {
		return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists,
			"duplicated nodeID").Error())
	}
	if resp := x.SetValue(stub, []string{index, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to put index-Node on chain: %s", resp.Message).Error())
	}

	// put index-nodeName on xchain
	index = packNodeNameIndex(n.Name)
	if resp := x.GetValue(stub, []string{index}); len(resp.Payload) != 0 {
		return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists,
			"duplicated nodeName").Error())
	}
	if resp := x.SetValue(stub, []string{index, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain, "failed to put index-NodeName on chain: %s",
			resp.Message).Error())
	}

	// put listIndex-node on xchain
	index = packNodeListIndex(n)
	if resp := x.SetValue(stub, []string{index, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain, "failed to put listIndex-Node on chain: %s",
			resp.Message).Error())
	}
	return shim.Success(nil)
}

// ListExecutorNodes gets all Executor nodes
func (x *Xdata) ListExecutorNodes(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var nodes blockchain.ExecutorNodes
	iterator, err := stub.GetStateByPartialCompositeKey(prefixNodeListIndex, []string{})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		var node blockchain.ExecutorNode
		if err := json.Unmarshal(queryResponse.Value, &node); err != nil {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal node").Error())
		}
		nodes = append(nodes, node)
	}

	// marshal nodes
	s, err := json.Marshal(nodes)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal nodes").Error())
	}
	return shim.Success(s)
}

// GetExecutorNodeByID gets Executor node by ID
func (x *Xdata) GetExecutorNodeByID(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting nodeID")
	}
	index := packNodeStringIndex(args[0])
	resp := x.GetValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "node not found: %s", resp.Message).Error())
	}
	return shim.Success(resp.Payload)
}

// GetExecutorNodeByName gets Executor node by NodeName
func (x *Xdata) GetExecutorNodeByName(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting name")
	}
	index := packNodeNameIndex(args[0])
	resp := x.GetValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound,
			"node not found: %s", resp.Message).Error())
	}
	return shim.Success(resp.Payload)
}
