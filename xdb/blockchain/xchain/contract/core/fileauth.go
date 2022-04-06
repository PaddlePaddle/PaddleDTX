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
	"encoding/json"
	"fmt"

	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// PublishFileAuthApplication add applier's file authorization application into chain
// In order to facilitate the applier or authorizer to query the list of applications,
// the authorization application will be written under the index_fileauth_list of applier and authorizer
func (x *Xdata) PublishFileAuthApplication(ctx code.Context) code.Response {
	// get PublishFileAuthApplication
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param: opt"))
	}
	// unmarshal opt
	var opt blockchain.PublishFileAuthOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal PublishFileAuthOptions"))
	}

	fa := opt.FileAuthApplication
	s, err := json.Marshal(fa)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal FileAuthApplication"))
	}
	// verify signature by applier's public key
	err = x.checkSign(opt.Signature, fa.Applier, s)
	if err != nil {
		return code.Error(err)
	}

	fa.Status = blockchain.FileAuthUnapproved
	// marshal fileAuthApplication
	s, err = json.Marshal(fa)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal fileAuthApplication"))
	}

	// judge if fileAuthIndex exists
	fileAuthIndex := packFileAuthIndex(fa.ID)
	if _, err := ctx.GetObject([]byte(fileAuthIndex)); err == nil {
		return code.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated file authID"))
	}

	// put index_fileauth into chain
	if err := ctx.PutObject([]byte(fileAuthIndex), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set index_fileauth on chain"))
	}
	// put index_fileauth_list_applier into chain
	applierListIndex := packFileAuthApplierIndex(fa.Applier, fa.ID, fa.CreateTime)
	if err := ctx.PutObject([]byte(applierListIndex), []byte(fa.ID)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set index_fileauth_list_applier on chain"))
	}
	// put index_fileauth_list_authorizer into chain
	authorizerListIndex := packFileAuthAuthorizerIndex(fa.Authorizer, fa.ID, fa.CreateTime)
	if err := ctx.PutObject([]byte(authorizerListIndex), []byte(fa.ID)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set index_fileauth_list_authorizer on chain"))
	}

	// put index_fileauth_list_applier_authorizer into chain
	authListIndex := packApplierAndAuthorizerIndex(fa.Applier, fa.Authorizer, fa.ID, fa.CreateTime)
	if err := ctx.PutObject([]byte(authListIndex), []byte(fa.ID)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set index_fileauth_list_applier_authorizer on chain"))
	}

	return code.OK([]byte("OK"))
}

// ConfirmFileAuthApplication is called when the dataOwner node confirms file's authorization
func (x *Xdata) ConfirmFileAuthApplication(ctx code.Context) code.Response {
	return x.setFileAuthConfirmStatus(ctx, true)
}

// RejectFileAuthApplication is called when the dataOwner node rejects file's authorization
func (x *Xdata) RejectFileAuthApplication(ctx code.Context) code.Response {
	return x.setFileAuthConfirmStatus(ctx, false)
}

// setFileAuthConfirmStatus set file's authorization application status as Approved or Rejected
func (x *Xdata) setFileAuthConfirmStatus(ctx code.Context, isConfirm bool) code.Response {
	var opt blockchain.ConfirmFileAuthOptions
	// get opt
	p, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	if err := json.Unmarshal(p, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal ConfirmFileAuthOptions"))
	}
	// query authorization application detail by authID
	fa, err := x.getFileAuthByID(ctx, opt.ID)
	if err != nil {
		return code.Error(err)
	}
	// verify signature by authorizer's public key
	m := fmt.Sprintf("%s,%d,", opt.ID, opt.CurrentTime)
	if isConfirm {
		m += fmt.Sprintf("%x,%d", opt.AuthKey, opt.ExpireTime)
	} else {
		m += opt.RejectReason
	}
	if err := x.checkSign(opt.Signature, fa.Authorizer, []byte(m)); err != nil {
		return code.Error(err)
	}

	// check status
	if fa.Status != blockchain.FileAuthUnapproved {
		return code.Error(errorx.New(errorx.ErrCodeParam,
			"confirm file auth error, fileAuthStatus is not Unapproved, authID: %s, fileAuthStatus: %s", fa.ID, fa.Status))
	}
	// update authorization status
	fa.ApprovalTime = opt.CurrentTime
	if isConfirm {
		fa.Status = blockchain.FileAuthApproved
		fa.ExpireTime = opt.ExpireTime
		fa.AuthKey = opt.AuthKey
	} else {
		fa.Status = blockchain.FileAuthRejected
		fa.RejectReason = opt.RejectReason
	}
	s, err := json.Marshal(fa)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "fail to marshal FileAuthApplication"))
	}
	// update index_fileauth on xchain
	index := packFileAuthIndex(fa.ID)
	if err := ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"fail to confirm index_fileauth on xchain"))
	}
	return code.OK([]byte("OK"))
}

// getFileAuthByID query file's authorization application by authID
func (x *Xdata) getFileAuthByID(ctx code.Context, authID string) (fa blockchain.FileAuthApplication, err error) {
	index := packFileAuthIndex(authID)
	s, err := ctx.GetObject([]byte(index))
	if err != nil {
		return fa, errorx.NewCode(err, errorx.ErrCodeNotFound,
			"the file authApplication[%s] not found", authID)
	}

	if err = json.Unmarshal([]byte(s), &fa); err != nil {
		return fa, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal FileAuthApplication")
	}
	return fa, nil
}

// ListFileAuthApplications list the authorization applications of files
// Support query by time range and fileID
func (x *Xdata) ListFileAuthApplications(ctx code.Context) code.Response {
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.ListFileAuthOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListFileAuthOptions"))
	}
	prefix := packFileAuthFilter(opt.Applier, opt.Authorizer)

	// get iter by prefix
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	var fas blockchain.FileAuthApplications
	for iter.Next() {
		if opt.Limit > 0 && int64(len(fas)) >= opt.Limit {
			break
		}
		fa, err := x.getFileAuthByID(ctx, string(iter.Value()))
		if err != nil {
			return code.Error(err)
		}
		if fa.CreateTime < opt.TimeStart || fa.CreateTime > opt.TimeEnd {
			continue
		}
		// If the fileID is not empty, query this fileID's authorization applications
		if opt.FileID != "" && opt.FileID != fa.FileID {
			continue
		}
		if opt.Status != "" && opt.Status != fa.Status {
			continue
		}
		fas = append(fas, &fa)
	}

	s, err := json.Marshal(fas)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal FileAuthApplications"))
	}
	return code.OK(s)
}

// GetAuthApplicationByID query authorization application detail by authID
func (x *Xdata) GetAuthApplicationByID(ctx code.Context) code.Response {
	// get id
	fileAuthID, ok := ctx.Args()["id"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param: fileAuthID"))
	}

	// get authorization application detail by index_fileauth
	index := packFileAuthIndex(string(fileAuthID))
	s, err := ctx.GetObject([]byte(index))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "fileAuthApplication not found"))
	}
	return code.OK(s)
}
