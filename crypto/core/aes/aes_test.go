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
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAES(t *testing.T) {
	key := sha256.Sum256([]byte("test key"))
	nonce := sha256.Sum256([]byte("test nonce"))
	plaintext := []byte("aes plaintext")
	aesKey := AESKey{
		Key:   key[:],
		Nonce: nonce[:12],
		AD:    nil,
	}

	cipher, err := EncryptUsingAESGCM(aesKey, plaintext, nil)
	if err != nil {
		t.Error(err)
	}

	plain, err := DecryptUsingAESGCM(aesKey, cipher, nil)
	if err != nil {
		t.Error(err)
	}

	require.Equal(t, plain, plaintext)
}
