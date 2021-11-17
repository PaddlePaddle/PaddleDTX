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
	"sort"
	"strings"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

func (mc *MockChain) ListChallengeRequests(ctx context.Context,
	opt *blockchain.ListChallengeOptions) ([]blockchain.Challenge, error) {

	useOwnerIndex := true
	if len(opt.FileOwner) == 0 && len(opt.TargetNode) > 0 {
		useOwnerIndex = false
	}

	prefix := packChallengeFilter(opt.FileOwner, opt.TargetNode)

	// sort
	indexs := mc.ChallengeIndexs4Owner
	if !useOwnerIndex {
		indexs = mc.ChallengeIndexs4Target
	}
	indexArray := make([]indexTuple, 0, len(indexs))
	for idx, challengeID := range indexs {
		if strings.HasPrefix(idx, prefix) {
			indexArray = append(indexArray, indexTuple{Index: idx, ID: challengeID})
		}
	}
	sort.Sort(sortableIndexTuple(indexArray))

	// iterate
	var cs []blockchain.Challenge
	for _, tuple := range indexArray {
		if opt.Limit > 0 && uint64(len(cs)) >= opt.Limit {
			break
		}
		c, ok := mc.Challenges[tuple.ID]
		if !ok {
			panic("should never happen")
		}
		if c.Status != blockchain.ChallengeToProve || c.ChallengeTime < opt.TimeStart || c.ChallengeTime > opt.TimeEnd {
			continue
		}
		cs = append(cs, c)
	}

	return cs, nil
}

func (mc *MockChain) ChallengeRequest(ctx context.Context,
	opt *blockchain.ChallengeRequestOptions) error {

	if _, exist := mc.Challenges[opt.ChallengeID]; exist {
		return errorx.New(errorx.ErrCodeAlreadyExists, "duplicated id")
	}

	// TODO sig verification

	// write
	c := blockchain.Challenge{
		ID:         opt.ChallengeID,
		FileOwner:  opt.FileOwner,
		TargetNode: opt.TargetNode,
		FileID:     opt.FileID,
		Status:     blockchain.ChallengeToProve,
	}

	index4Owner := packChallengeIndex4Owner(&c)
	index4Target := packChallengeIndex4Target(&c)

	mc.Challenges[opt.ChallengeID] = c
	mc.ChallengeIndexs4Owner[index4Owner] = c.ID
	mc.ChallengeIndexs4Target[index4Target] = c.ID

	mc.persistent()
	return nil
}

func (mc *MockChain) ChallengeAnswer(ctx context.Context,
	opt *blockchain.ChallengeAnswerOptions) ([]byte, error) {

	c, exist := mc.Challenges[opt.ChallengeID]
	if !exist {
		return nil, errorx.ErrNotFound
	}

	if c.Status == blockchain.ChallengeProved || c.Status == blockchain.ChallengeFailed {
		return nil, errorx.New(errorx.ErrCodeAlreadyExists, "challenge already answered")
	}

	// TODO sig verification

	c.Status = blockchain.ChallengeProved
	mc.Challenges[opt.ChallengeID] = c

	mc.persistent()
	return nil, nil
}

// GetChallengeById query challenge result
func (mc *MockChain) GetChallengeById(ctx context.Context, id string) (blockchain.Challenge, error) {
	c, exist := mc.Challenges[id]
	if !exist {
		return c, errorx.ErrNotFound
	}
	return c, nil
}
