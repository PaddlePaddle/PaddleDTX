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
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKDF(t *testing.T) {
	se := SoftEncryptor{
		password: "hello world",
	}

	fileID := "26bf4ded-5b36-44fa-9488-259de30a3c37"
	sliceID := "a80809b9-d8de-4c43-b680-ad3466c33b9d"
	key0, _ := hex.DecodeString("7a81d8031e60dac244baa03f4567522d048623c751824153337c573f6e25e1ab")

	nodeID, _ := hex.DecodeString("363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6")
	key1 := se.getKey(fileID, sliceID, nodeID)

	require.Equal(t, key0, key1)

	key2 := se.getKey(fileID, sliceID+"xx", nodeID)
	require.NotEqual(t, key0, key2)

	key3 := se.getKey(fileID, sliceID, append(nodeID, 1))
	require.NotEqual(t, key0, key3)
}
