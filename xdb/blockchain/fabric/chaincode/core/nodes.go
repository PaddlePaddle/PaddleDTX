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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// AddNode adds a node to fabric
func (x *Xdata) AddNode(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting AddNodeOptions")
	}

	// unmarshal opt
	var opt blockchain.AddNodeOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal AddNodeOptions").Error())
	}
	// get node
	n := opt.Node
	s, err := json.Marshal(n)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Node").Error())
	}

	// verify sig
	npk, err := hex.DecodeString(string(n.ID))
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeParam, "failed to decode nodeID").Error())
	}
	if err := x.checkSign(opt.Signature, npk, s); err != nil {
		return shim.Error(err.Error())
	}

	// put index-node on chain, judge if index exists
	index := packNodeIndex(n.ID)
	if resp := x.getValue(stub, []string{index}); len(resp.Payload) != 0 {
		return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists,
			"duplicated nodeID").Error())
	}
	if resp := x.setValue(stub, []string{index, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain, "failed to put index-Node on chain: %s",
			resp.Message).Error())
	}

	// put listIndex-node on chain
	index = packNodeListIndex(n)
	if resp := x.setValue(stub, []string{index, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain, "failed to put listIndex-Node on chain: %s",
			resp.Message).Error())
	}
	return shim.Success(nil)
}

// ListNodes gets all nodes from fabric
func (x *Xdata) ListNodes(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var nodes blockchain.Nodes

	// get nodes by prefix
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

		var node blockchain.Node
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

// GetNode gets node by id
func (x *Xdata) GetNode(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting nodeID")
	}

	// get index
	index := packNodeIndex([]byte(args[0]))
	// get node by index
	resp := x.getValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "node not found: %s", resp.Message).Error())
	}
	return shim.Success(resp.Payload)
}

// NodeOffline args = {id, sig}
func (x *Xdata) NodeOffline(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting nodeID and signature")
	}
	return x.setNodeOnlineStatus(stub, args, false)
}

// NodeOnline args = {id, sig}
func (x *Xdata) NodeOnline(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting nodeID and signature")
	}
	return x.setNodeOnlineStatus(stub, args, true)
}

// setNodeOnlineStatus sets node status
func (x *Xdata) setNodeOnlineStatus(stub shim.ChaincodeStubInterface, args []string, online bool) pb.Response {
	// unmarshal opt
	var opt blockchain.NodeOperateOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal NodeOperateOptions").Error())
	}
	// check nonce
	if err := x.checkAndSetNonce(stub, opt.NodeID, opt.Nonce); err != nil {
		return shim.Error(err.Error())
	}

	// verify sig
	npk, err := hex.DecodeString(string(opt.NodeID))
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decode nodeID").Error())
	}
	m := fmt.Sprintf("%s,%d", string(opt.NodeID), opt.Nonce)
	if err := x.checkSign(opt.Sig, npk, []byte(m)); err != nil {
		return shim.Error(err.Error())
	}

	// get node
	index := packNodeIndex(opt.NodeID)
	resp := x.getValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "node not found: %s", resp.Message).Error())
	}
	// unmarshal node
	var node blockchain.Node
	if err := json.Unmarshal(resp.Payload, &node); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal node").Error())
	}

	if node.Online != online {
		// change status
		node.Online = online
		// marshal new node
		newn, err := json.Marshal(node)
		if err != nil {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to marshal node").Error())
		}
		// put index-node on fabric
		if resp := x.setValue(stub, []string{index, string(newn)}); resp.Status == shim.ERROR {
			return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
				"failed to put index-Node on chain: %s", resp.Message).Error())
		}
		// put listIndex-node on chain
		index = packNodeListIndex(node)
		if resp := x.setValue(stub, []string{index, string(newn)}); resp.Status == shim.ERROR {
			return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
				"failed to put listIndex-Node on chain: %s", resp.Message).Error())
		}
	}
	return shim.Success([]byte("OK"))
}

// Heartbeat updates heartbeat of node
func (x *Xdata) Heartbeat(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("invalid arguments. expecting nodeID, signature and timestamp")
	}

	// get id
	nodeID := []byte(args[0])
	// get currentTime
	ctime, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parseInt currentTime").Error())
	}

	// get beginningTime
	btime, err := strconv.ParseInt(args[3], 10, 64)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parseInt beginningTime").Error())
	}

	// verify sig
	nodePK, err := hex.DecodeString(string(nodeID))
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decode nodeID").Error())
	}
	msg := fmt.Sprintf("%s,%d", string(nodeID), ctime)
	if err := x.checkSign([]byte(args[1]), nodePK, []byte(msg)); err != nil {
		return shim.Error(err.Error())
	}

	// update node heartbeat number
	hindex := packHeartBeatIndex(nodeID, btime)
	resp := x.getValue(stub, []string{hindex})
	var hb []int64
	if len(resp.Payload) == 0 {
		hb = append(hb, ctime)
	} else {
		if err := json.Unmarshal(resp.Payload, &hb); err != nil {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal heartbeat").Error())
		}
		for _, v := range hb {
			if v == ctime {
				return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "heartbeat has been checked of ctime").Error())
			}
		}
		hb = append(hb, ctime)
	}

	hbc, err := json.Marshal(hb)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal heartbeat number").Error())
	}
	if resp := x.setValue(stub, []string{hindex, string(hbc)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index-heartbeat on chain: %s", resp.Message).Error())
	}

	// update node updateAt after heartbeat success
	index := packNodeIndex(nodeID)
	resp = x.getValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "node not found: %s", resp.Message).Error())
	}
	var node blockchain.Node
	if err := json.Unmarshal(resp.Payload, &node); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal node").Error())
	}
	// update node heartbeat time
	node.UpdateAt = ctime
	newNode, err := json.Marshal(node)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal node").Error())
	}
	// put index-node on fabric
	if resp := x.setValue(stub, []string{index, string(newNode)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to put index-Node on chain: %s", resp.Message).Error())
	}
	// put listIndex-node on chain
	index = packNodeListIndex(node)
	if resp := x.setValue(stub, []string{index, string(newNode)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to put listIndex-Node on chain: %s", resp.Message).Error())
	}
	return shim.Success(nil)
}

// GetHeartbeatNum gets heartbeat by time
func (x *Xdata) GetHeartbeatNum(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("invalid arguments. expecting nodeID and timestamp")
	}

	// get id
	nodeID := []byte(args[0])
	ctime, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parseInt currentTime").Error())
	}

	// get heart beat by index
	hindex := packHeartBeatIndex(nodeID, ctime)
	resp := x.getValue(stub, []string{hindex})

	var hb []int64
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "heartbeat not found: %s", resp.Message).Error())
	}
	if err := json.Unmarshal(resp.Payload, &hb); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal node").Error())
	}
	return shim.Success([]byte(strconv.Itoa(len(hb))))
}

// ListNodesExpireSlice lists expired slices from fabric
func (x *Xdata) ListNodesExpireSlice(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("incorrect arguments. expecting ListNodeSliceOptions")
	}

	// unmarshal opt
	var opt blockchain.ListNodeSliceOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListNodeSlice").Error())
	}
	pubkey, err := hex.DecodeString(string(opt.Target))
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to decode nodeID").Error())
	}
	if len(pubkey) != ecdsa.PublicKeyLength ||
		opt.StartTime < 0 || opt.EndTime <= 0 || opt.EndTime <= opt.StartTime {
		return shim.Error(errorx.New(errorx.ErrCodeParam, "bad param").Error())
	}

	// pack prefix
	prefix, attr := packNodeSliceFilter(string(opt.Target))
	// iterate
	iterator, err := stub.GetStateByPartialCompositeKey(prefix, attr)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	var sl []string
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		expireTime := getNodeSliceFileID([]byte(queryResponse.Key))
		if (opt.Limit > 0 && int64(len(sl)) >= opt.Limit) || expireTime == 0 {
			break
		}
		if expireTime < opt.StartTime || expireTime > opt.EndTime {
			continue
		}

		sl = append(sl, string(queryResponse.Value))
	}

	rs, err := json.Marshal(sl)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal slices").Error())
	}
	return shim.Success(rs)
}

// GetSliceMigrateRecords is used to query node slice migration records
func (x *Xdata) GetSliceMigrateRecords(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("incorrect arguments. expecting NodeSliceMigrateOptions")
	}

	// unmarshal opt
	var opt blockchain.NodeSliceMigrateOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal NodeSliceMigrateOptions").Error())
	}
	pubkey, err := hex.DecodeString(string(opt.Target))
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to decode nodeID").Error())
	}
	if len(pubkey) != ecdsa.PublicKeyLength ||
		opt.StartTime < 0 || opt.EndTime <= 0 || opt.EndTime <= opt.StartTime {
		return shim.Error(errorx.New(errorx.ErrCodeParam, "bad param").Error())
	}

	// pack prefix
	prefix, attr := packNodeSliceMigrateFilter(string(opt.Target))
	// iterate
	iterator, err := stub.GetStateByPartialCompositeKey(prefix, attr)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	var sl []map[string]interface{}
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		mTime := getNodeSliceMigrateTime([]byte(queryResponse.Key))
		if (opt.Limit > 0 && int64(len(sl)) >= opt.Limit) || mTime == 0 {
			break
		}
		if mTime < opt.StartTime || mTime > opt.EndTime {
			continue
		}
		// unmarshal node
		var mr map[string]interface{}
		if err := json.Unmarshal(queryResponse.Value, &mr); err != nil {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal migrate records").Error())
		}
		mr["ctime"] = mTime
		sl = append(sl, mr)
	}
	rs, err := json.Marshal(sl)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal migrate records").Error())
	}
	return shim.Success(rs)
}

func (x *Xdata) checkAndSetNonce(stub shim.ChaincodeStubInterface, nodeID []byte, nonce int64) error {
	nonceIndex := packNonceIndex(nodeID, nonce)
	resp := x.getValue(stub, []string{nonceIndex})
	if len(resp.Payload) != 0 {
		return errorx.New(errorx.ErrCodeAlreadyExists, "duplicated node nonceIndex")
	}
	// put index-node-nonce on fabric
	if resp := x.setValue(stub, []string{nonceIndex, strconv.FormatInt(nonce, 10)}); resp.Status == shim.ERROR {
		return errorx.New(errorx.ErrCodeWriteBlockchain, "failed to set nonce on chain: %s", resp.Message)
	}
	return nil
}
