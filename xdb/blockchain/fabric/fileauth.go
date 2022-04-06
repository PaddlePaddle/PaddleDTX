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

package fabric

import (
	"encoding/json"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// PublishFileAuthApplication used for appliers to publish file authorization application
func (f *Fabric) PublishFileAuthApplication(opt *blockchain.PublishFileAuthOptions) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal PublishFileAuthOptions")
	}
	if _, err := f.InvokeContract([][]byte{opts}, "PublishFileAuthApplication"); err != nil {
		return err
	}
	return nil
}

// ConfirmFileAuthApplication dataOwner node confirms the applier's file authorization application
func (f *Fabric) ConfirmFileAuthApplication(opt *blockchain.ConfirmFileAuthOptions) error {
	return f.setFileAuthConfirmStatus(opt, true)
}

// RejectFileAuthApplication dataOwner node rejects the applier's file authorization application
// and gives the reason for the rejection
func (f *Fabric) RejectFileAuthApplication(opt *blockchain.ConfirmFileAuthOptions) error {
	return f.setFileAuthConfirmStatus(opt, false)
}

// setFileAuthConfirmStatus set file's authorization application status into the blokchain
func (f *Fabric) setFileAuthConfirmStatus(opt *blockchain.ConfirmFileAuthOptions, isConfirm bool) error {
	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal ConfirmFileAuthOptions")
	}
	mName := "RejectFileAuthApplication"
	if isConfirm {
		mName = "ConfirmFileAuthApplication"
	}
	if _, err := f.InvokeContract([][]byte{opts}, mName); err != nil {
		return err
	}
	return nil
}

// ListFileAuthApplications query the list of authorization applications
// Support query by time range and fileID
func (f *Fabric) ListFileAuthApplications(opt *blockchain.ListFileAuthOptions) (blockchain.FileAuthApplications, error) {
	var fas blockchain.FileAuthApplications

	opts, err := json.Marshal(*opt)
	if err != nil {
		return fas, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to marshal FileAuthApplications")
	}
	s, err := f.QueryContract([][]byte{opts}, "ListFileAuthApplications")
	if err != nil {
		return fas, err
	}
	if err = json.Unmarshal(s, &fas); err != nil {
		return fas, errorx.NewCode(err, errorx.ErrCodeInternal,
			"fail to unmarshal fileAuthApplications")
	}
	return fas, nil
}

// GetAuthApplicationByID query authorization application detail by authID
func (f *Fabric) GetAuthApplicationByID(authID string) (blockchain.FileAuthApplication, error) {
	var fa blockchain.FileAuthApplication
	s, err := f.QueryContract([][]byte{[]byte(authID)}, "GetAuthApplicationByID")
	if err != nil {
		return fa, err
	}

	if err = json.Unmarshal(s, &fa); err != nil {
		return fa, errorx.NewCode(err, errorx.ErrCodeInternal, "fail to unmarshal FileAuthApplication")
	}

	return fa, nil
}
