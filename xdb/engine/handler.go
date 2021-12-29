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
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// Push receive slices from dataOwner nodes
// rewriting a slice is not allowed
func (e *Engine) Push(opt types.PushOptions, r io.Reader) (
	types.PushResponse, error) {

	exist, err := e.storage.Exist(opt.SliceID)
	if err != nil {
		return types.PushResponse{}, errorx.Wrap(err, "failed to tell existence of slice")
	}
	// do not execute write to keep idempotency
	if exist {
		return types.PushResponse{}, nil
	}

	if err := e.storage.Save(opt.SliceID, r); err != nil {
		logger.WithError(err).Errorf("push %s", opt.SliceID)
		return types.PushResponse{}, errorx.Wrap(err, "failed to save slice")
	}

	logger.WithFields(logrus.Fields{
		"slice_id": opt.SliceID,
		"from":     opt.SourceId,
	}).Debug("slice received")

	return types.PushResponse{}, nil
}

// Pull load ciphertext slices locally and return them to the dataOwner node
// To prevent the request is intercepted and the slice is downloaded maliciously,
// the request's validity is five minutes
func (e *Engine) Pull(opt types.PullOptions) (io.ReadCloser, error) {
	//check timestamp
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

	// verify token
	msg := fmt.Sprintf("%s,%s,%d", opt.SliceID, opt.FileID, opt.Timestamp)
	msgDigest := hash.HashUsingSha256([]byte(msg))
	if err := verifyUserToken(hex.EncodeToString(file.Owner), opt.Signature, msgDigest); err != nil {
		return nil, errorx.Wrap(err, "failed to verify slice pull  token")
	}

	exist, err := e.storage.Exist(opt.SliceID)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to tell existence of slice")
	}
	if !exist {
		return nil, errorx.New(errorx.ErrCodeNotFound, "slice not found")
	}

	rc, err := e.storage.Load(opt.SliceID)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to load slice")
	}

	logger.WithFields(logrus.Fields{
		"slice_id": opt.SliceID,
		"from":     hex.EncodeToString(file.Owner),
	}).Debug("slice served")

	return rc, nil
}
