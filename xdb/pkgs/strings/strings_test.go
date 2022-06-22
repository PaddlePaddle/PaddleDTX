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

package strings

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/test-go/testify/require"
)

func TestGetSigMessage(t *testing.T) {
	var pubkey [ecdsa.PublicKeyLength]byte
	owner, _ := hex.DecodeString("4637ef79f14b036ced59b76408b0d88453ac9e5baa523a86890aa547eac3e3a0f4a3c005178f021c1b060d916f42082c18e1d57505cdaaeef106729e6442f4e5")
	copy(pubkey[:], owner)

	privatekey := "14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21"
	pri, _ := ecdsa.DecodePrivateKeyFromString(privatekey)
	// bytes, _ := json.Marshal(pubkey)
	// var ms [ecdsa.PublicKeyLength]byte
	// _ = json.Unmarshal([]byte(bytes), &ms)

	// map
	signMes := map[string]interface{}{
		"name":    "hello",
		"age":     23,
		"weight":  65.2,
		"life":    1652716800000000000,
		"friends": []string{"bob", "hy", "my", "xiaoming"},
		"pubkey":  pubkey,
		"hobby": map[string]interface{}{
			"book":   []string{"blockchain", "machine learning"},
			"status": []bool{false, true},
		},
		"privatekey": pri[:],
		"high":       "",
		"slices":     [][]byte{[]byte("adv"), []byte("aev")},
	}
	sig, err := GetSigMessage(signMes)
	fmt.Println("sign message: ", sig)
	require.Equal(t, err, nil)

	// struct
	var ranges []blockchain.Range
	ranges = append(ranges, blockchain.Range{End: 3108567, Start: 3108224})
	ranges = append(ranges, blockchain.Range{End: 1896988, Start: 1896885})

	opt := blockchain.ChallengeRequestOptions{
		ChallengeID:        "e6ebf3c9-d1af-4381-8656-844405fc9c18",
		FileOwner:          pubkey[:],
		FileID:             "df89df51-8ce6-4d37-91a2-d989c3d02e16",
		SliceIDs:           []string{"12dc329c-bfcc-43c4-b6f7-b2c3a3034ffd"},
		ChallengeTime:      1652784142259787725,
		ChallengeAlgorithm: "Pairing",
		Ranges:             ranges,
	}
	sig, err = GetSigMessage(opt)
	fmt.Println("sign message: ", sig)
	require.Equal(t, err, nil)
}
