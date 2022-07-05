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

package engine

import (
	"bytes"
	"encoding/hex"
	"io"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	util "github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/strings"
)

// Push receive slices from dataOwner nodes
// rewriting a slice is not allowed
func (e *Engine) Push(opt types.PushOptions, r io.Reader) (
	types.PushResponse, error) {

	var resp types.PushResponse
	// for content not a slice, save or update content
	if opt.NotASlice {
		if err := e.proveStorage.SaveAndUpdate(opt.SliceID, r); err != nil {
			logger.WithError(err).Errorf("push challenge material %s", opt.SliceID)
			return resp, errorx.Wrap(err, "failed to save slice")
		}
		resp.SliceStorIndex = opt.SliceID
	} else {
		storIndex, err := e.sliceStorage.Save(opt.SliceID, r)
		if err == nil {
			resp.SliceStorIndex = storIndex
		} else if errorx.Is(err, errorx.ErrCodeAlreadyExists) {
			// only in the case that storage.index is same with storage.key
			// could we receive this error code
			resp.SliceStorIndex = opt.SliceID
		} else {
			logger.WithError(err).Errorf("push %s", opt.SliceID)
			return resp, errorx.Wrap(err, "failed to save slice")
		}
	}

	logger.WithFields(logrus.Fields{
		"slice_id": opt.SliceID,
		"from":     opt.SourceID,
	}).Debug("slice received")
	return resp, nil
}

// Pull load ciphertext slices locally and return them to the dataOwner node
// To prevent the request is intercepted and the slice is downloaded maliciously,
// the request's validity is five minutes
func (e *Engine) Pull(opt types.PullOptions) (io.ReadCloser, error) {
	// Check timestamp
	var requestExpiredTime time.Duration = 5 * time.Minute
	if int64(opt.Timestamp) < (time.Now().UnixNano() - requestExpiredTime.Nanoseconds()) {
		return nil, errorx.New(errorx.ErrCodeParam, "request has expired")
	}
	file, err := e.chain.GetFileByID(opt.FileID)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"file_id": opt.FileID,
		}).Debug("failed to get file from chain")
		return nil, errorx.Wrap(err, "failed to read blockchain")
	}

	var verifyPubkey string
	// If opt.Pubkey is empty, use file owner's public key to verify signature
	if len(opt.Pubkey) == 0 || bytes.Equal(opt.Pubkey, file.Owner) {
		verifyPubkey = hex.EncodeToString(file.Owner)
	} else {
		verifyPubkey = hex.EncodeToString(opt.Pubkey)
		if err := e.checkApplierFileAuth(opt.Pubkey, file.Owner, opt.FileID); err != nil {
			return nil, err
		}
	}
	// Verify Signature
	msg, err := util.GetSigMessage(opt)
	if err != nil {
		return nil, errorx.Internal(err, "failed to get the message to sign for pull slices")
	}
	if err := verifyUserToken(verifyPubkey, opt.Signature, hash.HashUsingSha256([]byte(msg))); err != nil {
		return nil, errorx.Wrap(err, "failed to verify slice pull token")
	}

	var rc io.ReadCloser
	if opt.NotASlice {
		rc, err = e.proveStorage.Load(opt.SliceID)
		if err != nil {
			return nil, errorx.Wrap(err, "failed to load slice")
		}
	} else {
		rc, err = e.sliceStorage.Load(opt.SliceID, opt.StorIndex)
		if err != nil {
			return nil, errorx.Wrap(err, "failed to load sigma")
		}
	}

	logger.WithFields(logrus.Fields{
		"slice_id": opt.SliceID,
		"from":     hex.EncodeToString(file.Owner),
	}).Debug("slice served")

	return rc, nil
}

// checkApplierFileAuth used to check applier's file authorization application
// In addition to allowing file owners to download slice, authorized appliers can also download
func (e *Engine) checkApplierFileAuth(applier, authorizer []byte, fileID string) error {
	bcopt := blockchain.ListFileAuthOptions{
		Applier:    applier,
		Authorizer: authorizer,
		FileID:     fileID,
		Limit:      1,
		TimeEnd:    time.Now().UnixNano(),
		Status:     blockchain.FileAuthApproved,
	}
	fileAuths, err := e.chain.ListFileAuthApplications(&bcopt)
	if err != nil {
		return errorx.Wrap(err, "failed to read applier's authorization application from chain")
	}
	if len(fileAuths) == 0 {
		return errorx.New(errorx.ErrCodeNotFound, "applier's Approved authorization application not found")
	}
	if fileAuths[0].ExpireTime < time.Now().UnixNano() {
		return errorx.New(errorx.ErrCodeExpired, "applier's authorization application has expired")
	}
	return nil
}
