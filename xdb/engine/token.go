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

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
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

// getPubKey get the public key from string. if pubKeyStr is empty, return the node public key
func (e *Engine) getPubKey(pubKeyStr string) (pubKey []byte, err error) {
	if pubKeyStr == "" {
		nodePubKey := ecdsa.PublicKeyFromPrivateKey(e.monitor.challengingMonitor.PrivateKey)
		return nodePubKey[:], nil
	}
	pubkey, err := ecdsa.DecodePublicKeyFromString(pubKeyStr)
	if err != nil {
		return pubKey, errorx.Wrap(err, "failed to decode publickey")
	}
	return pubkey[:], nil
}

// verifyUserID verify whether the request userID is valid,
// userID can only be the local node public key or authorized dataOwner node client's public key
func (e *Engine) verifyUserID(userID string) error {
	userKeyFileName := hash.HashUsingSha256([]byte(userID))
	if ok, err := file.IsFileExisted(file.AuthKeyFilePath, hex.EncodeToString(userKeyFileName)); ok {
		logger.Info("userID:", userID, ok, err)
		return nil
	}
	return e.verifyUserIDIsLocalNodeID(userID)
}

// verifyUserIsID verify if request userID is valid and equal to local nodeID
func (e *Engine) verifyUserIDIsLocalNodeID(userID string) error {
	localPub := ecdsa.PublicKeyFromPrivateKey(e.monitor.challengingMonitor.PrivateKey)
	if userID != localPub.String() {
		return errorx.New(errorx.ErrCodeNotAuthorized, "request userID isn't be authorized or does not match local userID")
	}
	return nil
}
