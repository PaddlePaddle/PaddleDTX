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

package soft

import (
	"io"

	"golang.org/x/crypto/hkdf"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
)

// getKey derive encrypt key by password, fileID, sliceID and NodeID using key derivation function
func (se *SoftEncryptor) getKey(fileID, sliceID string, nodeID []byte) []byte {
	secret := []byte(se.password)
	salt := append(append([]byte(fileID), []byte(sliceID)...), nodeID...)
	r := hkdf.New(hash.DefaultHasher, secret, salt, nil)

	// 256 bit symmetric encryption
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		panic(err)
	}

	return key
}
