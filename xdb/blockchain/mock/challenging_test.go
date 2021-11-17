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
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/testings"
)

var (
	challenge1 = uuid.NewString()
	challenge2 = uuid.NewString()
	challenge3 = uuid.NewString()
	challenge4 = uuid.NewString()
)

func assertChallengeIn(t *testing.T, cs []blockchain.Challenge, ids ...string) {
	idMap := make(map[string]struct{}, len(cs))
	for _, c := range cs {
		idMap[c.ID] = struct{}{}
	}

	for _, id := range ids {
		_, exist := idMap[id]
		require.Equal(t, true, exist)
	}
}

func request4(t *testing.T, mc *MockChain) {
	ctx := context.Background()

	user1, _ := hex.DecodeString(testings.PK1)
	user2, _ := hex.DecodeString(testings.PK2)
	user3, _ := hex.DecodeString(testings.PK3)

	// user1 -> user2 @ slice1
	opt1 := blockchain.ChallengeRequestOptions{
		ChallengeID: challenge1,
		FileOwner:   user1,
		TargetNode:  user2,
	}

	// user1 -> user2 @ slice2
	opt2 := blockchain.ChallengeRequestOptions{
		ChallengeID: challenge2,
		FileOwner:   user1,
		TargetNode:  user2,
	}

	// user1 -> user3 @ slice3
	opt3 := blockchain.ChallengeRequestOptions{
		ChallengeID: challenge3,
		FileOwner:   user1,
		TargetNode:  user3,
	}

	// user2 -> user3 @ slice4
	opt4 := blockchain.ChallengeRequestOptions{
		ChallengeID: challenge4,
		FileOwner:   user2,
		TargetNode:  user3,
	}

	err := mc.ChallengeRequest(ctx, &opt1)
	require.NoError(t, err)
	err = mc.ChallengeRequest(ctx, &opt2)
	require.NoError(t, err)
	err = mc.ChallengeRequest(ctx, &opt3)
	require.NoError(t, err)
	err = mc.ChallengeRequest(ctx, &opt4)
	require.NoError(t, err)

	// duplicated
	err = mc.ChallengeRequest(ctx, &opt4)
	require.Error(t, err)
}

func answer2nd(t *testing.T, mc *MockChain) {
	ctx := context.Background()

	opt1 := blockchain.ChallengeAnswerOptions{
		ChallengeID: challenge2,
	}
	_, err := mc.ChallengeAnswer(ctx, &opt1)
	require.NoError(t, err)

	// already done
	_, err = mc.ChallengeAnswer(ctx, &opt1)
	require.Error(t, err)

	// bad proof
	opt2 := blockchain.ChallengeAnswerOptions{
		ChallengeID: challenge3,
	}
	_, err = mc.ChallengeAnswer(ctx, &opt2)
	require.Error(t, err)

}

func listChallenge(t *testing.T, mc *MockChain) {
	ctx := context.Background()
	user1, _ := hex.DecodeString(testings.PK1)
	user2, _ := hex.DecodeString(testings.PK2)
	user3, _ := hex.DecodeString(testings.PK3)

	// all
	cs, err := mc.ListChallengeRequests(ctx, &blockchain.ListChallengeOptions{})
	require.NoError(t, err)
	require.Equal(t, 4, len(cs))
	ordered := cs

	// all limit 3
	cs, err = mc.ListChallengeRequests(ctx, &blockchain.ListChallengeOptions{
		Limit: 3,
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(cs))
	assertChallengeIn(t, cs, ordered[0].ID, ordered[1].ID, ordered[2].ID)

	// exclude done
	cs, err = mc.ListChallengeRequests(ctx, &blockchain.ListChallengeOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, len(cs))
	assertChallengeIn(t, cs, challenge1, challenge3, challenge4)

	// owner:user1
	cs, err = mc.ListChallengeRequests(ctx, &blockchain.ListChallengeOptions{
		FileOwner: user1,
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(cs))
	assertChallengeIn(t, cs, challenge1, challenge3)

	// owner:user2
	cs, err = mc.ListChallengeRequests(ctx, &blockchain.ListChallengeOptions{
		FileOwner: user2,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(cs))
	assertChallengeIn(t, cs, challenge4)

	// target:user3
	cs, err = mc.ListChallengeRequests(ctx, &blockchain.ListChallengeOptions{
		TargetNode: user3,
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(cs))
	assertChallengeIn(t, cs, challenge3, challenge4)

	// owner:user1 target:user3
	cs, err = mc.ListChallengeRequests(ctx, &blockchain.ListChallengeOptions{
		FileOwner:  user1,
		TargetNode: user3,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(cs))
	assertChallengeIn(t, cs, challenge3)

	// owner:user1 range from the second
	lastOne := ""
	for i, o := range ordered {
		if bytes.Equal(o.FileOwner, user1) {
			lastOne = ordered[i+2].ID
			break
		}
	}
	cs, err = mc.ListChallengeRequests(ctx, &blockchain.ListChallengeOptions{
		FileOwner: user1,
		TimeEnd:   time.Now().UnixNano(),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(cs))
	assertChallengeIn(t, cs, lastOne)
}

func TestChallengeOp(t *testing.T) {
	mc := New(&NewMockchainOptions{Persistent: false})

	request4(t, mc)

	answer2nd(t, mc)

	listChallenge(t, mc)
}
