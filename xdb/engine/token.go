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
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// verifyUserToken check user's token is valid or not
//  returns error if any mistake occurs or signature dismatches
func verifyUserToken(userID, token string, digest []byte) error {
	pubkey, err := ecdsa.DecodePublicKeyFromString(userID)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeParam, "bad user id")
	}

	sig, err := ecdsa.DecodeSignatureFromString(token)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeParam, "bad token")
	}

	if err := ecdsa.Verify(pubkey, digest, sig); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeBadSignature, "bad signature")
	}

	return nil
}

// verifyUserID verify if request userID is valid and equal to local nodeID
func (e *Engine) verifyUserID(userID string) error {
	localPub := ecdsa.PublicKeyFromPrivateKey(e.monitor.challengingMonitor.PrivateKey)
	if userID != localPub.String() {
		return errorx.New(errorx.ErrCodeNotAuthorized, "request userID does not match local userID")
	}
	return nil
}
