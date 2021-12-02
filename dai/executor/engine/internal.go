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

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// verifyUserID checks if request userID is valid and equal to local nodeID
func verifyUserID(userID []byte, privateKey ecdsa.PrivateKey) error {
	localPub := ecdsa.PublicKeyFromPrivateKey(privateKey)
	if !bytes.Equal(userID[:], localPub[:]) {
		return errorx.New(errorx.ErrCodeNotAuthorized, "request userID does not match local userID")
	}
	return nil
}
