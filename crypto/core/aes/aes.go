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

package aes

import (
	"crypto/aes"
	"crypto/cipher"
)

type AESKey struct {
	Key   []byte // 32 bytes
	Nonce []byte // 12 bytes = GCM.NonceSize
	AD    []byte // optional
}

// EncryptUsingAESGCM encrypt using AES_GCM
func EncryptUsingAESGCM(key AESKey, plaintext []byte, dst []byte) ([]byte, error) {
	// AES256
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, err
	}

	c, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return c.Seal(dst, key.Nonce, plaintext, key.AD), nil
}

// DecryptUsingAESGCM decrypt AES256-GCM
func DecryptUsingAESGCM(key AESKey, ciphertext []byte, dst []byte) ([]byte, error) {
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, err
	}

	c, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	raw, err := c.Open(dst, key.Nonce, ciphertext, key.AD)
	if err != nil {
		return nil, err
	}

	return raw, nil
}
