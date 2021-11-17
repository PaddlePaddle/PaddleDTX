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

package mock

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/testings"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mustDecodeHex(s string) []byte {
	bs, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return bs
}

func assertIn(t *testing.T, fs []blockchain.File, ids ...string) {
	idMap := make(map[string]struct{}, len(fs))
	for _, f := range fs {
		idMap[f.ID] = struct{}{}
	}

	for _, id := range ids {
		_, exist := idMap[id]
		require.Equal(t, true, exist)
	}
}

func TestListFiles(t *testing.T) {
	mc := New(&NewMockchainOptions{Persistent: false})
	ctx := context.Background()

	user1ns1file1 := blockchain.File{
		ID:        uuid.NewString(),
		Owner:     mustDecodeHex(testings.PK1),
		Namespace: "11111111",
		Name:      "name111111",
	}
	user1ns1file2 := blockchain.File{
		ID:        uuid.NewString(),
		Owner:     mustDecodeHex(testings.PK1),
		Namespace: "11111111",
		Name:      "name222",
	}
	user1ns2file3 := blockchain.File{
		ID:        uuid.NewString(),
		Owner:     mustDecodeHex(testings.PK1),
		Namespace: "222222222222",
		Name:      "name33333333",
	}
	user2ns3file4 := blockchain.File{
		ID:        uuid.NewString(),
		Owner:     mustDecodeHex(testings.PK2),
		Namespace: "33333333",
		Name:      "name444444",
	}
	optFiles1 := &blockchain.PublishFileOptions{
		File: user1ns1file1,
	}
	optFiles2 := &blockchain.PublishFileOptions{
		File: user1ns1file2,
	}
	optFiles3 := &blockchain.PublishFileOptions{
		File: user1ns2file3,
	}
	optFiles4 := &blockchain.PublishFileOptions{
		File: user2ns3file4,
	}

	err := mc.PublishFile(ctx, optFiles1)
	require.NoError(t, err)
	err = mc.PublishFile(ctx, optFiles2)
	require.NoError(t, err)
	err = mc.PublishFile(ctx, optFiles3)
	require.NoError(t, err)
	err = mc.PublishFile(ctx, optFiles4)
	require.NoError(t, err)

	// all files of user1
	fs, err := mc.ListFiles(ctx, &blockchain.ListFileOptions{
		Owner:   mustDecodeHex(testings.PK1),
		TimeEnd: time.Now().UnixNano(),
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(fs))
	assertIn(t, fs, user1ns1file1.ID, user1ns1file2.ID, user1ns2file3.ID)

	// all files of user1 (limit 2)
	fs, err = mc.ListFiles(ctx, &blockchain.ListFileOptions{
		Owner: mustDecodeHex(testings.PK1),
		Limit: 2,
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(fs))
	assertIn(t, fs, user1ns1file1.ID, user1ns1file2.ID)

	// all files of user1 (range from the second one)
	fs, err = mc.ListFiles(ctx, &blockchain.ListFileOptions{
		Owner:   user1ns1file2.Owner,
		TimeEnd: time.Now().UnixNano(),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(fs))
	assertIn(t, fs, user1ns2file3.ID)
}
