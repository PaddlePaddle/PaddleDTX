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

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// PublishFileAuthApplication add applier's file authorization application into chain
// In order to facilitate the applier or authorizer to query the list of applications,
// the authorization application will be written under the index_fileauth_list of applier and authorizer
func (x *Xdata) PublishFileAuthApplication(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// get PublishFileAuthApplication
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting PublishFileAuthOptions")
	}

	// unmarshal opt
	var opt blockchain.PublishFileAuthOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal PublishFileAuthOptions").Error())
	}

	fa := opt.FileAuthApplication
	s, err := json.Marshal(fa)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal FileAuthApplication").Error())
	}
	// verify signature by applier's public key
	err = x.checkSign(opt.Signature, fa.Applier, s)
	if err != nil {
		return shim.Error(err.Error())
	}

	fa.Status = blockchain.FileAuthUnapproved
	// marshal fileAuthApplication
	s, err = json.Marshal(fa)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal fileAuthApplication").Error())
	}

	// judge if fileAuthIndex exists
	fileAuthIndex := packFileAuthIndex(fa.ID)
	if resp := x.getValue(stub, []string{fileAuthIndex}); len(resp.Payload) != 0 {
		return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated file authID").Error())
	}

	// put index_fileauth into chain
	if resp := x.setValue(stub, []string{fileAuthIndex, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index_fileauth on chain: %s", resp.Message).Error())
	}

	// put index_fileauth_list_applier into chain
	applierListIndex := packFileAuthApplierIndex(fa.Applier, fa.ID, fa.CreateTime)
	if resp := x.setValue(stub, []string{applierListIndex, fa.ID}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index_fileauth_list_applier on chain: %s", resp.Message).Error())
	}

	// put index_fileauth_list_authorizer into chain
	authorizerListIndex := packFileAuthAuthorizerIndex(fa.Authorizer, fa.ID, fa.CreateTime)
	if resp := x.setValue(stub, []string{authorizerListIndex, fa.ID}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index_fileauth_list_authorizer on chain: %s", resp.Message).Error())
	}

	// put index_fileauth_list_applier_authorizer into chain
	authListIndex := packApplierAndAuthorizerIndex(fa.Applier, fa.Authorizer, fa.ID, fa.CreateTime)
	if resp := x.setValue(stub, []string{authListIndex, fa.ID}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index_fileauth_list_applier_authorizer on chain: %s", resp.Message).Error())
	}
	return shim.Success([]byte("OK"))
}

// ConfirmFileAuthApplication is called when the dataOwner node confirms file's authorization
func (x *Xdata) ConfirmFileAuthApplication(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	return x.setFileAuthConfirmStatus(stub, args, true)
}

// RejectFileAuthApplication is called when the dataOwner node rejects file's authorization
func (x *Xdata) RejectFileAuthApplication(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	return x.setFileAuthConfirmStatus(stub, args, false)
}

// setFileAuthConfirmStatus set file's authorization application status as Approved or Rejected
func (x *Xdata) setFileAuthConfirmStatus(stub shim.ChaincodeStubInterface, args []string, isConfirm bool) pb.Response {
	// get opt
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting ConfirmFileAuthOptions")
	}

	// unmarshal opt
	var opt blockchain.ConfirmFileAuthOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ConfirmFileAuthOptions").Error())
	}

	// query authorization application detail by authID
	fa, err := x.getFileAuthByID(stub, opt.ID)
	if err != nil {
		return shim.Error(err.Error())
	}
	// verify signature by authorizer's public key
	m := fmt.Sprintf("%s,%d,", opt.ID, opt.CurrentTime)
	if isConfirm {
		m += fmt.Sprintf("%x,%d", opt.AuthKey, opt.ExpireTime)
	} else {
		m += opt.RejectReason
	}
	if err := x.checkSign(opt.Signature, fa.Authorizer, []byte(m)); err != nil {
		return shim.Error(err.Error())
	}

	// check status
	if fa.Status != blockchain.FileAuthUnapproved {
		return shim.Error(errorx.New(errorx.ErrCodeParam,
			"confirm file auth error, fileAuthStatus is not Unapproved, authID: %s, fileAuthStatus: %s", fa.ID, fa.Status).Error())
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
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "fail to marshal FileAuthApplication").Error())
	}
	// update index_fileauth on chain
	index := packFileAuthIndex(fa.ID)
	if resp := x.setValue(stub, []string{index, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to confirm index_fileauth on chain: %s", resp.Message).Error())
	}

	return shim.Success([]byte("OK"))
}

// getFileAuthByID query file's authorization application by authID
func (x *Xdata) getFileAuthByID(stub shim.ChaincodeStubInterface, authID string) (fa blockchain.FileAuthApplication, err error) {
	index := packFileAuthIndex(authID)
	resp := x.getValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return fa, errorx.New(errorx.ErrCodeNotFound, "fileAuthApplication[%s] not found: %s", authID, resp.Message)
	}

	if err = json.Unmarshal([]byte(resp.Payload), &fa); err != nil {
		return fa, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal FileAuthApplication")
	}
	return fa, nil
}

// ListFileAuthApplications list the authorization applications of files
// Support query by time range and fileID
func (x *Xdata) ListFileAuthApplications(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting ListFileAuthOptions")
	}

	// unmarshal opt
	var opt blockchain.ListFileAuthOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListFileAuthOptions").Error())
	}
	prefix, attr := packFileAuthFilter(opt.Applier, opt.Authorizer)

	// get iter by prefix
	iterator, err := stub.GetStateByPartialCompositeKey(prefix, attr)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	// iterate iter
	var fas blockchain.FileAuthApplications
	for iterator.HasNext() {
		if opt.Limit > 0 && int64(len(fas)) >= opt.Limit {
			break
		}
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		fa, err := x.getFileAuthByID(stub, string(queryResponse.Value))
		if err != nil {
			return shim.Error(err.Error())
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
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal FileAuthApplications").Error())
	}
	return shim.Success(s)
}

// GetAuthApplicationByID query authorization application detail by authID
func (x *Xdata) GetAuthApplicationByID(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. missing param: fileAuthID")
	}

	// get authorization application detail by index_fileauth
	index := packFileAuthIndex(string(args[0]))
	resp := x.getValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "fileAuthApplication not found: %s", resp.Message).Error())
	}
	return shim.Success(resp.Payload)
}
