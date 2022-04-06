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
	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// AddNode adds a node to xchain
func (x *Xdata) AddNode(ctx code.Context) code.Response {
	var opt blockchain.AddNodeOptions
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:node"))
	}
	//unmarshal opt
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal AddNodeOptions"))
	}
	// get node
	node := opt.Node
	// marshal node
	s, err := json.Marshal(node)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Node"))
	}
	// verify sig
	npk, err := hex.DecodeString(string(node.ID))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decode nodeID"))
	}
	if err := x.checkSign(opt.Signature, npk, s); err != nil {
		return code.Error(err)
	}

	// put index-node on xchain, judge if index exists
	index := packNodeIndex(node.ID)
	if _, err := ctx.GetObject([]byte(index)); err == nil {
		return code.Error(errorx.New(errorx.ErrCodeAlreadyExists,
			"duplicated nodeid"))
	}
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"failed to put index-Node on xchain"))
	}

	// put listIndex-node on xchain
	index = packNodeListIndex(node)
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"failed to put listIndex-Node on xchain"))
	}
	return code.OK([]byte("added"))
}

// ListNodes gets all nodes from xchain
func (x *Xdata) ListNodes(ctx code.Context) code.Response {
	var nodes blockchain.Nodes

	// get nodes by list_prefix
	prefix := prefixNodeListIndex
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	for iter.Next() {
		var node blockchain.Node
		if err := json.Unmarshal(iter.Value(), &node); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal node"))
		}
		nodes = append(nodes, node)
	}
	// marshal nodes
	s, err := json.Marshal(nodes)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal nodes"))
	}
	return code.OK(s)
}

// GetNode gets node by id
func (x *Xdata) GetNode(ctx code.Context) code.Response {
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

// NodeOffline gets node offline
func (x *Xdata) NodeOffline(ctx code.Context) code.Response {
	return x.setNodeOnlineStatus(ctx, false)
}

// NodeOnline gets node online
func (x *Xdata) NodeOnline(ctx code.Context) code.Response {
	return x.setNodeOnlineStatus(ctx, true)
}

// setNodeOnlineStatus sets node status
func (x *Xdata) setNodeOnlineStatus(ctx code.Context, online bool) code.Response {
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	var opt blockchain.NodeOperateOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal NodeOperateOptions"))
	}
	// check nonce
	if err := checkAndSetNonce(ctx, opt.NodeID, opt.Nonce); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to check nonce"))
	}
	// verify sig
	npk, err := hex.DecodeString(string(opt.NodeID))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decode nodeID"))
	}
	m := fmt.Sprintf("%s,%d", string(opt.NodeID), opt.Nonce)
	if err := x.checkSign(opt.Sig, npk, []byte(m)); err != nil {
		return code.Error(err)
	}

	// get node by index
	index := packNodeIndex(opt.NodeID)
	oldn, err := ctx.GetObject([]byte(index))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "node not found"))
	}

	// unmarshal node
	var node blockchain.Node
	if err := json.Unmarshal(oldn, &node); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal node"))
	}

	if node.Online != online {
		// change status
		node.Online = online
		// marshal new node
		newn, err := json.Marshal(node)
		if err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to marshal node"))
		}
		// put index-node on xchain
		if err := ctx.PutObject([]byte(index), newn); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
				"failed to put index-Node on xchain"))
		}
		// put listIndex-node on xchain
		index = packNodeListIndex(node)
		if err := ctx.PutObject([]byte(index), newn); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
				"failed to put listIndex-Node on xchain"))
		}
	}
	return code.OK([]byte("OK"))
}

// Heartbeat updates heartbeat of node
func (x *Xdata) Heartbeat(ctx code.Context) code.Response {
	// get id
	nodeID, ok := ctx.Args()["id"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}
	// get currentTime
	h, ok := ctx.Args()["currentTime"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:currentTime"))
	}
	ctime, err := strconv.ParseInt(string(h), 10, 64)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parseInt currentTime"))
	}
	// get beginningTime
	bt, ok := ctx.Args()["beginningTime"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:beginningTime"))
	}
	btime, err := strconv.ParseInt(string(bt), 10, 64)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parseInt beginningTime"))
	}
	//get signature
	signature, ok := ctx.Args()["signature"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:signature"))
	}
	// verify sig
	if len(signature) != ecdsa.SignatureLength {
		return code.Error(errorx.New(errorx.ErrCodeParam, "bad param:signature"))
	}
	nodePK, err := hex.DecodeString(string(nodeID))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decode nodeID"))
	}
	msg := fmt.Sprintf("%s,%d", string(nodeID), ctime)
	if err := x.checkSign(signature, nodePK, []byte(msg)); err != nil {
		return code.Error(err)
	}

	// update node heartbeat number
	hindex := packNodeHeartBeatIndex(nodeID, btime)
	hl, err := ctx.GetObject([]byte(hindex))
	var hb []int64
	if err != nil {
		hb = append(hb, ctime)
	} else {
		if err := json.Unmarshal(hl, &hb); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal heartbeat"))
		}
		for _, v := range hb {
			if v == ctime {
				return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "heartbeat has been checked of ctime"))
			}
		}
		hb = append(hb, ctime)
	}
	hbc, err := json.Marshal(hb)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal heartbeat number"))
	}
	if err := ctx.PutObject([]byte(hindex), hbc); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set index-heartbeat on chain"))
	}

	// update node updateAt after heartbeat success
	index := packNodeIndex(nodeID)
	oldn, err := ctx.GetObject([]byte(index))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "node not found"))
	}
	var node blockchain.Node
	if err := json.Unmarshal(oldn, &node); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal node"))
	}

	// update node heartbeat time
	node.UpdateAt = ctime
	newn, err := json.Marshal(node)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal node"))
	}
	// put index-node on xchain
	if err := ctx.PutObject([]byte(index), newn); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to put index-Node on xchain"))
	}
	// put listIndex-node on xchain
	index = packNodeListIndex(node)
	if err := ctx.PutObject([]byte(index), newn); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to put listIndex-Node on xchain"))
	}
	return code.OK(nil)
}

// GetHeartbeatNum gets heartbeat by time
func (x *Xdata) GetHeartbeatNum(ctx code.Context) code.Response {
	// get id
	nodeID, ok := ctx.Args()["id"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}
	c, ok := ctx.Args()["currentTime"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:currentTime"))
	}
	ctime, err := strconv.ParseInt(string(c), 10, 64)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parseInt currentTime"))
	}

	hindex := packNodeHeartBeatIndex(nodeID, ctime)
	// get node by index
	n, err := ctx.GetObject([]byte(hindex))
	var hb []int64
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "heartbeat not found"))
	}
	if err := json.Unmarshal(n, &hb); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal node"))
	}
	return code.OK([]byte(strconv.Itoa(len(hb))))
}

// ListNodesExpireSlice lists expired slices from xchain
func (x *Xdata) ListNodesExpireSlice(ctx code.Context) code.Response {
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.ListNodeSliceOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListNodeSlice"))
	}
	pubkey, err := hex.DecodeString(string(opt.Target))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeParam, "wrong target node"))
	}
	if len(pubkey) != ecdsa.PublicKeyLength ||
		opt.StartTime < 0 || opt.EndTime <= 0 || opt.EndTime <= opt.StartTime {
		return code.Error(errorx.New(errorx.ErrCodeParam, "bad param"))
	}
	// pack prefix
	prefix := packNodeSliceFilter(string(opt.Target))

	// get iter by prefix
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	// iterate iter
	var sl []string
	for iter.Next() {
		_, expireTime := getNodeSliceFileID(iter.Key())
		if (opt.Limit > 0 && int64(len(sl)) >= opt.Limit) || expireTime == 0 {
			break
		}
		if expireTime < opt.StartTime || expireTime > opt.EndTime {
			continue
		}

		sl = append(sl, string(iter.Value()))
	}
	rs, err := json.Marshal(sl)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Files"))
	}
	return code.OK(rs)
}

// GetSliceMigrateRecords queries node slice migration records
func (x *Xdata) GetSliceMigrateRecords(ctx code.Context) code.Response {
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.NodeSliceMigrateOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal NodeSliceMigrateOptions"))
	}
	pubkey, err := hex.DecodeString(string(opt.Target))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeParam, "wrong target node"))
	}
	if len(pubkey) != ecdsa.PublicKeyLength ||
		opt.StartTime < 0 || opt.EndTime <= 0 || opt.EndTime <= opt.StartTime {
		return code.Error(errorx.New(errorx.ErrCodeParam, "bad param"))
	}
	// pack prefix
	prefix := packNodeSliceMigrateFilter(string(opt.Target))

	// get iter by prefix
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	// iterate iter
	var sl []map[string]interface{}
	for iter.Next() {
		mTime := getNodeSliceMigrateTime(iter.Key())
		if (opt.Limit > 0 && int64(len(sl)) >= opt.Limit) || mTime == 0 {
			break
		}
		if mTime < opt.StartTime || mTime > opt.EndTime {
			continue
		}
		// unmarshal node
		var mr map[string]interface{}
		if err := json.Unmarshal(iter.Value(), &mr); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal migrate records"))
		}

		mr["ctime"] = mTime
		sl = append(sl, mr)
	}
	rs, err := json.Marshal(sl)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal migrate records"))
	}
	return code.OK(rs)
}

func checkAndSetNonce(ctx code.Context, nodeID []byte, nonce int64) error {
	nonceIndex := packNonceIndex(nodeID, nonce)
	if _, err := ctx.GetObject([]byte(nonceIndex)); err == nil {
		return errorx.New(errorx.ErrCodeAlreadyExists,
			"duplicated node nonceIndex")
	}
	if err := ctx.PutObject([]byte(nonceIndex), []byte(strconv.FormatInt(nonce, 10))); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set nonce on xchain")
	}
	return nil
}
