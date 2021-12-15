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

package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/kataras/iris/v12"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	etype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/server/types"
)

// write upload a local file
func (s *Server) write(ictx iris.Context) {
	expireTime, err := ictx.URLParamInt64("expireTime")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid param expireTime"))
		return
	}
	req := etype.WriteOptions{
		User:        ictx.URLParam("user"),
		Token:       ictx.URLParam("token"),
		Namespace:   ictx.URLParam("ns"),
		FileName:    ictx.URLParam("name"),
		ExpireTime:  expireTime,
		Description: ictx.URLParam("desc"),
		Extra:       ictx.URLParam("ext"),
	}
	if err := req.Valid(); err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid params"))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })

	result, err := s.handler.Write(ctx, req, ictx.Request().Body)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to write"))
		return
	}
	resp := types.WriteResponse{
		FileID: result.FileID,
	}
	responseJSON(ictx, resp)
}

// read download a file to local
func (s *Server) read(ictx iris.Context) {
	req := etype.ReadOptions{
		User:      ictx.URLParam("user"),
		Token:     ictx.URLParam("token"),
		Namespace: ictx.URLParam("ns"),
		FileName:  ictx.URLParam("name"),
		FileID:    ictx.URLParam("file_id"),
		Timestamp: ictx.URLParamUint64("timestamp"),
	}
	if err := req.Valid(); err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid params"))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })

	reader, err := s.handler.Read(ctx, req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to read"))
		return
	}
	defer reader.Close()

	responseStream(ictx, reader)
}

// push receives slice from others
func (s *Server) push(ictx iris.Context) {
	sliceID := ictx.URLParam("slice_id")
	sourceId := ictx.URLParam("source_id")

	opt := etype.PushOptions{
		SliceID:  sliceID,
		SourceId: sourceId,
	}

	_, err := s.handler.Push(opt, ictx.Request().Body)
	if err != nil {
		responseError(ictx, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to push slice"))
		return
	}

	resp := types.PushResponse{}
	responseJSON(ictx, resp)
}

// pull offers file slices to owner
func (s *Server) pull(ictx iris.Context) {
	sliceID := ictx.URLParam("slice_id")
	fileId := ictx.URLParam("file_id")
	timestamp := ictx.URLParamUint64("timestamp")
	sig := ictx.URLParam("signature")

	opt := etype.PullOptions{
		SliceID:   sliceID,
		FileID:    fileId,
		Timestamp: timestamp,
		Signature: sig,
	}

	resultReader, err := s.handler.Pull(opt)
	if err != nil {
		responseError(ictx, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to pull slice"))
		return
	}
	defer resultReader.Close()

	responseStream(ictx, resultReader)
}

// addNode adds storage node
func (s *Server) addNode(ictx iris.Context) {
	req := etype.AddNodeOptions{
		NodeID:  ictx.URLParam("node"),
		Name:    ictx.URLParam("name"),
		Address: ictx.URLParam("address"),
		Online:  true,
		Token:   ictx.URLParam("token"),
	}

	if err := s.handler.AddNode(req); err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to addnode"))
		return
	}

	responseJSON(ictx, "success")
}

// listNodes list storage nodes
func (s *Server) listNodes(ictx iris.Context) {
	resp, err := s.handler.ListNodes()
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to list nodes"))
		return
	}
	responseJSON(ictx, resp)
}

// getNode get storage node info
func (s *Server) getNode(ictx iris.Context) {
	id := ictx.URLParam("id")

	resp, err := s.handler.GetNode([]byte(id))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to getnode"))
		return
	}
	responseJSON(ictx, resp)
}

// nodeOffline set storage node status to offline
func (s *Server) nodeOffline(ictx iris.Context) {
	nonce, err := ictx.URLParamInt64("nonce")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid nonce"))
	}

	req := etype.NodeOfflineOptions{
		NodeID: ictx.URLParam("node"),
		Nonce:  nonce,
		Token:  ictx.URLParam("token"),
	}

	if err := s.handler.NodeOffline(req); err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to take node offline"))
		return
	}
	responseJSON(ictx, "success")
}

// nodeOnline set storage node status to online
func (s *Server) nodeOnline(ictx iris.Context) {
	nonce, err := ictx.URLParamInt64("nonce")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid nonce"))
	}
	req := etype.NodeOnlineOptions{
		NodeID: ictx.URLParam("node"),
		Nonce:  nonce,
		Token:  ictx.URLParam("token"),
	}

	if err := s.handler.NodeOnline(req); err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to take node online"))
		return
	}
	responseJSON(ictx, "success")
}

// getMRecord get storage node migration records
func (s *Server) getMRecord(ictx iris.Context) {
	opt := &blockchain.NodeSliceMigrateOptions{
		Target:    []byte(ictx.URLParam("id")),
		StartTime: ictx.URLParamInt64Default("start", 0),
		EndTime:   ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:     ictx.URLParamUint64("limit"),
	}

	resp, err := s.handler.GetSliceMigrateRecords(opt)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to list slice migrate records"))
		return
	}
	var ms []map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &ms); err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to unmarshal slice migrate records"))
	}
	responseJSON(ictx, ms)
}

// getHeartbeatNum get storage node heartbeat number
func (s *Server) getHeartbeatNum(ictx iris.Context) {
	id := ictx.URLParam("id")
	ctime := ictx.URLParamInt64Default("ctime", 0)

	heartBeatTotal, heartBeatMax, err := s.handler.GetHeartbeatNum([]byte(id), ctime)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to getnode heartbeat num"))
		return
	}
	resp := map[string]int{
		"heartBeatTotal": heartBeatTotal,
		"heartBeatMax":   heartBeatMax,
	}
	responseJSON(ictx, resp)
}

// listFiles list files
func (s *Server) listFiles(ictx iris.Context) {
	owner, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to decode public key"))
		return
	}
	if ictx.URLParam("ns") == "" {
		responseError(ictx, errorx.New(errorx.ErrCodeParam, "bad params:ns"))
		return
	}

	req := etype.ListFileOptions{
		Owner:       owner[:],
		Namespace:   ictx.URLParam("ns"),
		TimeStart:   ictx.URLParamInt64Default("start", 0),
		TimeEnd:     ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		CurrentTime: ictx.URLParamInt64Default("ctime", time.Now().UnixNano()),
		Limit:       ictx.URLParamUint64("limit"),
	}

	resp, err := s.handler.ListFiles(req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to list files"))
		return
	}
	responseJSON(ictx, resp)
}

// listExpiredFiles list expired but valid files
func (s *Server) listExpiredFiles(ictx iris.Context) {
	owner, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to decode public key"))
		return
	}
	if ictx.URLParam("ns") == "" {
		responseError(ictx, errorx.New(errorx.ErrCodeParam, "bad params:ns is empty"))
		return
	}

	req := etype.ListFileOptions{
		Owner:       owner[:],
		Namespace:   ictx.URLParam("ns"),
		TimeStart:   ictx.URLParamInt64Default("start", 0),
		TimeEnd:     ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		CurrentTime: ictx.URLParamInt64Default("ctime", time.Now().UnixNano()),
		Limit:       ictx.URLParamUint64("limit"),
	}

	resp, err := s.handler.ListExpiredFiles(req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to list expired files"))
		return
	}
	responseJSON(ictx, resp)
}

// getFileByID get file by id
func (s *Server) getFileByID(ictx iris.Context) {
	id := ictx.URLParam("id")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })
	resp, err := s.handler.GetFileByID(ctx, id)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get file by id"))
		return
	}
	responseJSON(ictx, resp)
}

// getFileByName get file by file name and namespace
func (s *Server) getFileByName(ictx iris.Context) {
	owner, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to decode publickey"))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })
	resp, err := s.handler.GetFileByName(ctx, owner[:], ictx.URLParam("ns"), ictx.URLParam("name"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get file by name"))
		return
	}
	responseJSON(ictx, resp)
}

// updateFileExpireTime update file expire time
func (s *Server) updateFileExpireTime(ictx iris.Context) {
	expireTime, err := ictx.URLParamInt64("expireTime")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid param expireTime"))
		return
	}
	if expireTime <= time.Now().UnixNano() {
		responseError(ictx, errorx.New(errorx.ErrCodeParam, "invalid param expireTime"))
		return
	}

	cTime, err := ictx.URLParamInt64("ctime")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid current time"))
		return
	}

	req := etype.UpdateFileEtimeOptions{
		Owner:       ictx.URLParam("owner"),
		FileID:      ictx.URLParam("id"),
		ExpireTime:  expireTime,
		CurrentTime: cTime,
		Token:       ictx.URLParam("token"),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })

	err = s.handler.UpdateFileExpireTime(ctx, req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to update file expire time"))
		return
	}
	responseJSON(ictx, "success")
}

// addFileNs add a file namespace
func (s *Server) addFileNs(ictx iris.Context) {
	addTime, err := ictx.URLParamInt64("ctime")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid create time"))
		return
	}

	replica, err := ictx.URLParamInt("replica")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid replica params"))
		return
	}

	lresp, err := s.handler.ListNodes()
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to list nodes"))
		return
	}
	if replica > len(lresp) {
		responseError(ictx, errorx.New(errorx.ErrCodeParam, "invalid param: replica must no greater than nodes number"))
		return
	}
	req := etype.AddNsOptions{
		Owner:       ictx.URLParam("owner"),
		Namespace:   ictx.URLParam("ns"),
		Description: ictx.URLParam("desc"),
		Replica:     replica,
		CreateTime:  addTime,
		Token:       ictx.URLParam("token"),
	}
	err = s.handler.AddFileNs(req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to add file ns"))
		return
	}
	responseJSON(ictx, "success")
}

// updateNsReplica update file namespace replica
func (s *Server) updateNsReplica(ictx iris.Context) {
	cTime, err := ictx.URLParamInt64("ctime")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid current time params"))
		return
	}

	replica, err := ictx.URLParamInt("replica")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid replica params"))
		return
	}

	req := etype.UpdateNsOptions{
		Owner:       ictx.URLParam("owner"),
		Namespace:   ictx.URLParam("ns"),
		Replica:     replica,
		CurrentTime: cTime,
		Token:       ictx.URLParam("token"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })

	err = s.handler.UpdateNsReplica(ctx, req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to update file ns replica"))
		return
	}
	responseJSON(ictx, "success")
}

// listFileNs list namespaces by owner
func (s *Server) listFileNs(ictx iris.Context) {
	owner, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to decode publickey"))
		return
	}

	req := etype.ListNsOptions{
		Owner:     owner[:],
		TimeStart: ictx.URLParamInt64Default("start", 0),
		TimeEnd:   ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:     ictx.URLParamUint64("limit"),
	}

	resp, err := s.handler.ListFileNs(req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to listfiles"))
		return
	}
	responseJSON(ictx, resp)
}

// getNsByName get namespace by name
func (s *Server) getNsByName(ictx iris.Context) {
	owner, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to decode publickey"))
		return
	}
	name := ictx.URLParam("name")
	if name == "" {
		responseError(ictx, errorx.New(errorx.ErrCodeParam, "bad params:ns is empty"))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })
	resp, err := s.handler.GetNsByName(ctx, owner[:], name)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get ns detail"))
		return
	}
	responseJSON(ictx, resp)
}

// getSysHealth get file owner system health status
func (s *Server) getSysHealth(ictx iris.Context) {
	owner, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to decode publickey"))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })
	resp, err := s.handler.GetFileSysHealth(ctx, owner[:])
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get file sys health detail"))
		return
	}
	responseJSON(ictx, resp)
}

// getChallengeById get challenge by challenge id
func (s *Server) getChallengeById(ictx iris.Context) {
	id := ictx.URLParam("id")
	resp, err := s.handler.GetChallengeById(id)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get challenge by id"))
		return
	}
	responseJSON(ictx, resp)
}

// getToProveChallenges get challenges with status "ToProve"
func (s *Server) getToProveChallenges(ictx iris.Context) {
	var owner []byte
	if len(ictx.URLParam("owner")) != 0 {
		pubkey, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
		if err != nil {
			responseError(ictx, errorx.Wrap(err, "failed to decode owner public key"))
			return
		}
		owner = append(owner, pubkey[:]...)
	}
	node := []byte(ictx.URLParam("node"))
	file := ictx.URLParam("file")

	opt := blockchain.ListChallengeOptions{
		FileOwner:  owner,
		TargetNode: node,
		FileID:     file,
		Status:     blockchain.ChallengeToProve,
		TimeStart:  ictx.URLParamInt64Default("start", 0),
		TimeEnd:    ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:      ictx.URLParamUint64("limit"),
	}

	resp, err := s.handler.GetChallenges(opt)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get proves challenge"))
		return
	}
	responseJSON(ictx, resp)
}

// getProvedChallenges get challenges with status "Proved"
func (s *Server) getProvedChallenges(ictx iris.Context) {
	var owner []byte
	if len(ictx.URLParam("owner")) != 0 {
		pubkey, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
		if err != nil {
			responseError(ictx, errorx.Wrap(err, "failed to decode owner public key"))
			return
		}
		owner = append(owner, pubkey[:]...)
	}
	node := []byte(ictx.URLParam("node"))
	file := ictx.URLParam("file")

	opt := blockchain.ListChallengeOptions{
		FileOwner:  owner,
		TargetNode: node,
		FileID:     file,
		Status:     blockchain.ChallengeProved,
		TimeStart:  ictx.URLParamInt64Default("start", 0),
		TimeEnd:    ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:      ictx.URLParamUint64("limit"),
	}

	resp, err := s.handler.GetChallenges(opt)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get proves challenge"))
		return
	}
	responseJSON(ictx, resp)
}

// getFailedChallenges get challenges with status "Failed"
func (s *Server) getFailedChallenges(ictx iris.Context) {
	var owner []byte
	if len(ictx.URLParam("owner")) != 0 {
		pubkey, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("owner"))
		if err != nil {
			responseError(ictx, errorx.Wrap(err, "failed to decode owner public key"))
			return
		}
		owner = append(owner, pubkey[:]...)
	}
	node := []byte(ictx.URLParam("node"))
	file := ictx.URLParam("file")

	opt := blockchain.ListChallengeOptions{
		FileOwner:  owner[:],
		TargetNode: node,
		FileID:     file,
		Status:     blockchain.ChallengeFailed,
		TimeStart:  ictx.URLParamInt64Default("start", 0),
		TimeEnd:    ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:      ictx.URLParamUint64("limit"),
	}

	resp, err := s.handler.GetChallenges(opt)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get failed challenge"))
		return
	}
	responseJSON(ictx, resp)
}

// getNodeHealth get storage node health status
func (s *Server) getNodeHealth(ictx iris.Context) {
	id := []byte(ictx.URLParam("id"))

	resp, err := s.handler.GetNodeHealth(id)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get node health status"))
		return
	}
	responseJSON(ictx, resp)
}
