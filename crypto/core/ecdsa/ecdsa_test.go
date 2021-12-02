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

package ecdsa

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEcdsa(t *testing.T) {
	privkey, pubkey, err := GenerateKeyPair()
	if err != nil {
		t.Error(err)
	}

	digest := sha256.Sum256([]byte("test"))
	sign, err := Sign(privkey, digest[:])
	if err != nil {
		t.Error(err)
	}
	if err := Verify(pubkey, digest[:], sign); err != nil {
		t.Error(err)
	}

	pubFromPrvi := PublicKeyFromPrivateKey(privkey)
	require.Equal(t, pubFromPrvi, pubkey)

	privkeyStr := privkey.String()
	privFromStr, err := DecodePrivateKeyFromString(privkeyStr)
	if err != nil {
		t.Error(err)
	}
	require.Equal(t, privFromStr, privkey)

	pubkeyStr := pubkey.String()
	pubFromStr, err := DecodePublicKeyFromString(pubkeyStr)
	if err != nil {
		t.Error(err)
	}
	require.Equal(t, pubFromStr, pubkey)

	signStr := sign.String()
	signFromStr, err := DecodeSignatureFromString(signStr)
	if err != nil {
		t.Error(err)
	}
	require.Equal(t, signFromStr, sign)
}
