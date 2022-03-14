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

package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerkle(t *testing.T) {
	hash0 := sha256.Sum256([]byte("0"))
	hash1 := sha256.Sum256([]byte("1"))
	hash2 := sha256.Sum256([]byte("2"))
	hash3 := sha256.Sum256([]byte("3"))
	hash4 := sha256.Sum256([]byte("4"))
	hashes := [][]byte{hash0[:], hash1[:], hash2[:], hash3[:], hash4[:]}
	correctRootHex := "54a14b0e6b4445648bce07d7a5882940f2d7a940cb01b81e531be0f2e33167fd"

	root := GetMerkleRoot(hashes)
	require.Equal(t, hex.EncodeToString(root), correctRootHex)
}
