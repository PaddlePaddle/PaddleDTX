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
	"strings"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/kataras/iris/v12"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	etype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/server/types"
)

// write upload a local file
func (s *Server) write(ictx iris.Context) {
	req := etype.WriteOptions{
		User:        ictx.URLParam("user"),
		Token:       ictx.URLParam("token"),
		Namespace:   ictx.URLParam("ns"),
		FileName:    ictx.URLParam("name"),
		ExpireTime:  ictx.URLParamInt64Default("expireTime", 0),
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
		Timestamp: ictx.URLParamInt64Default("timestamp", 0),
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
	opt := etype.PushOptions{
		SliceID:  ictx.URLParam("slice_id"),
		SourceID: ictx.URLParam("source_id"),
	}
	// if sliceID has suffix, like 'sigmas', the pushed content is not a slice
	// currently, pairing based challenge material sigmas is supported
	if strings.TrimSuffix(opt.SliceID, common.ChallengeFileSuffix) != opt.SliceID {
		opt.NotASlice = true
	}
	if _, err := s.handler.Push(opt, ictx.Request().Body); err != nil {
		responseError(ictx, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to push slice"))
		return
	}

	resp := types.PushResponse{}
	responseJSON(ictx, resp)
}

// pull offers file slices to owner
func (s *Server) pull(ictx iris.Context) {
	opt := etype.PullOptions{
		SliceID:   ictx.URLParam("slice_id"),
		FileID:    ictx.URLParam("file_id"),
		Timestamp: ictx.URLParamInt64Default("timestamp", 0),
		Signature: ictx.URLParam("signature"),
	}
	if ictx.URLParam("pubkey") != "" {
		pubkey, err := ecdsa.DecodePublicKeyFromString(ictx.URLParam("pubkey"))
		if err != nil {
			responseError(ictx, errorx.Wrap(err, "failed to decode publickey"))
			return
		}
		opt.Pubkey = pubkey[:]
	}

	resultReader, err := s.handler.Pull(opt)
	if err != nil {
		responseError(ictx, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to pull slice"))
		return
	}
	defer resultReader.Close()

	responseStream(ictx, resultReader)
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

// getNode get storage node info, param-id is the storage node public key
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
	req := etype.NodeOfflineOptions{
		NodeID: ictx.URLParam("node"),
		Nonce:  ictx.URLParamInt64Default("nonce", 0),
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
	req := etype.NodeOnlineOptions{
		NodeID: ictx.URLParam("node"),
		Nonce:  ictx.URLParamInt64Default("nonce", 0),
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
		Limit:     ictx.URLParamInt64Default("limit", blockchain.ListMaxNumber),
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

// getHeartbeatNum get storage node heartbeat number, param-id is the storage node public key
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
// The currentTime is used to determine whether the file is expired,
// only show the list of unexpired files
func (s *Server) listFiles(ictx iris.Context) {
	req := etype.ListFileOptions{
		Owner:       ictx.URLParam("owner"),
		Namespace:   ictx.URLParam("ns"),
		TimeStart:   ictx.URLParamInt64Default("start", 0),
		TimeEnd:     ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		CurrentTime: ictx.URLParamInt64Default("ctime", time.Now().UnixNano()),
		Limit:       ictx.URLParamInt64Default("limit", blockchain.ListMaxNumber),
	}

	if err := req.Valid(); err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid params"))
		return
	}

	resp, err := s.handler.ListFiles(req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to list files"))
		return
	}
	responseJSON(ictx, resp)
}

// listExpiredFiles list expired but valid files
// The currentTime is used to determine whether the file is expired
func (s *Server) listExpiredFiles(ictx iris.Context) {
	req := etype.ListFileOptions{
		Owner:       ictx.URLParam("owner"),
		Namespace:   ictx.URLParam("ns"),
		TimeStart:   ictx.URLParamInt64Default("start", 0),
		TimeEnd:     ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		CurrentTime: ictx.URLParamInt64Default("ctime", time.Now().UnixNano()),
		Limit:       ictx.URLParamInt64Default("limit", blockchain.ListMaxNumber),
	}

	if err := req.Valid(); err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid params"))
		return
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })
	resp, err := s.handler.GetFileByName(ctx, ictx.URLParam("owner"), ictx.URLParam("ns"), ictx.URLParam("name"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get file by name"))
		return
	}
	responseJSON(ictx, resp)
}

// updateFileExpireTime update file expire time
func (s *Server) updateFileExpireTime(ictx iris.Context) {
	req := etype.UpdateFileEtimeOptions{
		FileID:      ictx.URLParam("id"),
		ExpireTime:  ictx.URLParamInt64Default("expireTime", 0),
		CurrentTime: ictx.URLParamInt64Default("ctime", 0),
		User:        ictx.URLParam("user"),
		Token:       ictx.URLParam("token"),
	}
	if err := req.Valid(); err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid params"))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })

	if err := s.handler.UpdateFileExpireTime(ctx, req); err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to update file expire time"))
		return
	}
	responseJSON(ictx, "success")
}

// addFileNs add a file namespace
func (s *Server) addFileNs(ictx iris.Context) {
	// check files replica of namespace, replica must no greater than nodes number
	replica := ictx.URLParamIntDefault("replica", 0)
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
		CreateTime:  ictx.URLParamInt64Default("ctime", time.Now().UnixNano()),
		User:        ictx.URLParam("user"),
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
	req := etype.UpdateNsOptions{
		Namespace:   ictx.URLParam("ns"),
		Replica:     ictx.URLParamIntDefault("replica", 0),
		CurrentTime: ictx.URLParamInt64Default("ctime", time.Now().UnixNano()),
		User:        ictx.URLParam("user"),
		Token:       ictx.URLParam("token"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })

	if err := s.handler.UpdateNsReplica(ctx, req); err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to update file ns replica"))
		return
	}
	responseJSON(ictx, "success")
}

// listFileNs list namespaces by owner
func (s *Server) listFileNs(ictx iris.Context) {
	req := etype.ListNsOptions{
		Owner:     ictx.URLParam("owner"),
		TimeStart: ictx.URLParamInt64Default("start", 0),
		TimeEnd:   ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:     ictx.URLParamInt64Default("limit", blockchain.ListMaxNumber),
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
	name := ictx.URLParam("name")
	if name == "" {
		responseError(ictx, errorx.New(errorx.ErrCodeParam, "bad params:ns is empty"))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })
	resp, err := s.handler.GetNsByName(ctx, ictx.URLParam("owner"), name)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get ns detail"))
		return
	}
	responseJSON(ictx, resp)
}

// getSysHealth get file owner system health status
func (s *Server) getSysHealth(ictx iris.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ictx.OnConnectionClose(func(iris.Context) { cancel() })
	resp, err := s.handler.GetFileSysHealth(ctx, ictx.URLParam("owner"))
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get file sys health detail"))
		return
	}
	responseJSON(ictx, resp)
}

// listFileAuths query the list of authorization applications
// Support query by time range and fileID
func (s *Server) listFileAuths(ictx iris.Context) {
	req := etype.ListFileAuthOptions{
		Applier:    ictx.URLParam("applierPubkey"),
		Authorizer: ictx.URLParam("authorizerPubkey"),
		FileID:     ictx.URLParam("fileID"),
		Status:     ictx.URLParam("status"),
		TimeStart:  ictx.URLParamInt64Default("start", 0),
		TimeEnd:    ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:      ictx.URLParamInt64Default("limit", blockchain.ListMaxNumber),
	}

	resp, err := s.handler.ListFileAuths(req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get file authorization applications"))
		return
	}
	responseJSON(ictx, resp)
}

// confirmAuth the dataOwner node confirms or rejects the applier's file authorization application
func (s *Server) confirmAuth(ictx iris.Context) {
	status, err := ictx.URLParamBool("status")
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid param status"))
		return
	}
	req := etype.ConfirmAuthOptions{
		User:         ictx.URLParam("user"),
		AuthID:       ictx.URLParam("authID"),
		Status:       status,
		ExpireTime:   ictx.URLParamInt64Default("expireTime", 0),
		Token:        ictx.URLParam("token"),
		RejectReason: ictx.URLParam("rejectReason"),
	}
	if err := req.Valid(status); err != nil {
		responseError(ictx, errorx.Wrap(err, "invalid params"))
		return
	}
	err = s.handler.ConfirmAuth(req)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to confirm file authorization application"))
		return
	}
	responseJSON(ictx, "success")
}

// getAuthByID query authorization application detail by authID
func (s *Server) getAuthByID(ictx iris.Context) {
	id := ictx.URLParam("authID")
	resp, err := s.handler.GetAuthByID(id)
	if err != nil {
		responseError(ictx, errorx.Wrap(err, "failed to get file authorization application by authID"))
		return
	}
	responseJSON(ictx, resp)
}

// getChallengeByID get challenge by challenge id
func (s *Server) getChallengeByID(ictx iris.Context) {
	id := ictx.URLParam("id")
	resp, err := s.handler.GetChallengeByID(id)
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

	opt := blockchain.ListChallengeOptions{
		FileOwner:  owner,
		TargetNode: []byte(ictx.URLParam("node")),
		FileID:     ictx.URLParam("file"),
		Status:     blockchain.ChallengeToProve,
		TimeStart:  ictx.URLParamInt64Default("start", 0),
		TimeEnd:    ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:      ictx.URLParamInt64Default("limit", blockchain.ListMaxNumber),
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

	opt := blockchain.ListChallengeOptions{
		FileOwner:  owner,
		TargetNode: []byte(ictx.URLParam("node")),
		FileID:     ictx.URLParam("file"),
		Status:     blockchain.ChallengeProved,
		TimeStart:  ictx.URLParamInt64Default("start", 0),
		TimeEnd:    ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:      ictx.URLParamInt64Default("limit", blockchain.ListMaxNumber),
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

	opt := blockchain.ListChallengeOptions{
		FileOwner:  owner[:],
		TargetNode: []byte(ictx.URLParam("node")),
		FileID:     ictx.URLParam("file"),
		Status:     blockchain.ChallengeFailed,
		TimeStart:  ictx.URLParamInt64Default("start", 0),
		TimeEnd:    ictx.URLParamInt64Default("end", time.Now().UnixNano()),
		Limit:      ictx.URLParamInt64Default("limit", blockchain.ListMaxNumber),
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
