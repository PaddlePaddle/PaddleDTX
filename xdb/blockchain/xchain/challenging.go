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

package xchain

import (
	"encoding/json"
	"strconv"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// ListChallengeRequests lists all challenge requests on blockchain
func (x *XChain) ListChallengeRequests(opt *blockchain.ListChallengeOptions) ([]blockchain.Challenge, error) {

	opts, err := json.Marshal(*opt)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal ListChallengeOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ListChallengeRequests"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return nil, err
	}
	var cs []blockchain.Challenge
	if err = json.Unmarshal(s, &cs); err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal Challenges")
	}
	return cs, nil
}

// ChallengeRequest sets a challenge request on chain
func (x *XChain) ChallengeRequest(opt *blockchain.ChallengeRequestOptions) error {

	opts, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal ChallengeRequestOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ChallengeRequest"
	if _, err = x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// ChallengeAnswer sets a challenge answer on chain
func (x *XChain) ChallengeAnswer(opt *blockchain.ChallengeAnswerOptions) ([]byte, error) {

	opts, err := json.Marshal(*opt)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal ChallengeAnswerOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ChallengeAnswer"
	resp, err := x.InvokeContract(args, mName)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetChallengeByID gets a challenge by id
func (x *XChain) GetChallengeByID(id string) (blockchain.Challenge, error) {

	var c blockchain.Challenge
	args := map[string]string{
		"id": id,
	}
	mName := "GetChallengeByID"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return c, err
	}

	if err = json.Unmarshal(s, &c); err != nil {
		return c, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal Challenge")
	}

	return c, nil
}

// GetChallengeNum gets challenge number given filter
func (x *XChain) GetChallengeNum(opt *blockchain.GetChallengeNumOptions) (uint64, error) {

	opts, err := json.Marshal(*opt)
	if err != nil {
		return 0, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal GetChallengeNumOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "GetChallengeNum"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return 0, err
	}

	num, err := strconv.ParseUint(string(s), 10, 64)
	if err != nil {
		return 0, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to parse contract response to number")
	}
	return num, nil
}
