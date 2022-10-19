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

package core

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	fl_crypto "github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/monitor/challenging"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	util "github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/strings"
)

var xchainClient = new(fl_crypto.XchainCryptoClient)

// ListChallengeRequests List challenge requests on chain
// args = {ListChallengeOptions}
func (x *Xdata) ListChallengeRequests(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("incorrect arguments. expecting ListChallengeOptions")
	}

	// unmarshal opt
	var opt blockchain.ListChallengeOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListChallengeOptions").Error())
	}

	// pack prefix
	prefix, attr := packChallengeFilter(opt.FileOwner, opt.TargetNode)
	// get iter by prefix
	iterator, err := stub.GetStateByPartialCompositeKey(prefix, attr)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	var cs []blockchain.Challenge
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		if opt.Limit > 0 && int64(len(cs)) >= opt.Limit {
			break
		}
		index := packChallengeIndex(string(queryResponse.Value))
		resp := x.GetValue(stub, []string{index})
		if len(resp.Payload) == 0 {
			return shim.Error(errorx.New(errorx.ErrCodeNotFound,
				"the Challenge[%x] not found: %s", queryResponse.Value, resp.Message).Error())
		}

		var c blockchain.Challenge
		if err = json.Unmarshal(resp.Payload, &c); err != nil {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal Challenge").Error())
		}
		if c.Status != opt.Status || c.ChallengeTime < opt.TimeStart || c.ChallengeTime > opt.TimeEnd {
			continue
		}
		if len(opt.FileID) != 0 && c.FileID != opt.FileID {
			continue
		}
		cs = append(cs, c)
	}

	s, err := json.Marshal(cs)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Challenges").Error())
	}
	return shim.Success(s)
}

// ChallengeRequest Set a challenge request on chain
// args = {ChallengeRequestOptions}
func (x *Xdata) ChallengeRequest(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("incorrect arguments. expecting ChallengeRequestOptions")
	}

	var opt blockchain.ChallengeRequestOptions
	// unmarshal opt
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ChallengeRequestOptions").Error())
	}
	// judge if id exists
	index := packChallengeIndex(opt.ChallengeID)
	if resp := x.GetValue(stub, []string{index}); len(resp.Payload) != 0 {
		return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated ChallengeID").Error())
	}

	// verify signature
	msg, err := util.GetSigMessage(opt)
	if err != nil {
		return shim.Error(errorx.Internal(err, "failed to get the message to sign").Error())
	}
	err = x.checkSign(opt.Signature, opt.FileOwner, []byte(msg))
	if err != nil {
		return shim.Error(err.Error())
	}

	// make challenge
	c := blockchain.Challenge{
		ID:                 opt.ChallengeID,
		FileOwner:          opt.FileOwner,
		TargetNode:         opt.TargetNode,
		FileID:             opt.FileID,
		Status:             blockchain.ChallengeToProve,
		ChallengeTime:      opt.ChallengeTime,
		ChallengeAlgorithm: opt.ChallengeAlgorithm,
	}

	if opt.ChallengeAlgorithm == types.PairingChallengeAlgorithm {
		c.SliceIDs = opt.SliceIDs
		c.SliceStorIndexes = opt.SliceStorIndexes
		c.Indices = opt.Indices
		c.Round = opt.Round
		c.RandThisRound = opt.RandThisRound
		c.Vs = opt.Vs

	} else if opt.ChallengeAlgorithm == types.MerkleChallengeAlgorithm {
		c.SliceID = opt.SliceID
		c.SliceStorIndex = opt.SliceStorIndex
		c.Ranges = opt.Ranges
		c.HashOfProof = opt.HashOfProof
	} else {
		return shim.Error(errorx.New(errorx.ErrCodeParam, "bad param:opt-challengeAlgorithm").Error())
	}

	// marshal challenge
	s, err := json.Marshal(c)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal Challenge").Error())
	}
	// set challengeID-challenge on chain
	if resp := x.SetValue(stub, []string{index, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set ChallengeID-Challenge on chain: %s", resp.Message).Error())
	}

	// set index40owner-challengeID on chain
	index4Owner := packChallengeIndex4Owner(&c)
	if resp := x.SetValue(stub, []string{index4Owner, c.ID}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index40owner-ChallengeID on chain: %s", resp.Message).Error())
	}
	// set index4Target-challengeID on chain
	index4Target := packChallengeIndex4Target(&c)
	if resp := x.SetValue(stub, []string{index4Target, c.ID}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index4Target-ChallengeID on chain: %s", resp.Message).Error())
	}

	return shim.Success([]byte("requested"))
}

// ChallengeAnswer Set a challengeAnswer on chain
// args = {ChallengeAnswerOptions}
func (x *Xdata) ChallengeAnswer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("incorrect arguments. expecting ChallengeAnswerOptions")
	}

	var opt blockchain.ChallengeAnswerOptions
	// unmarshal opt
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ChallengeAnswerOptions").Error())
	}
	// judge if challenge exists
	index := packChallengeIndex(opt.ChallengeID)
	resp := x.GetValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "Challenge not found: %s", resp.Message).Error())
	}
	// unmarshal challenge
	var c blockchain.Challenge
	if err := json.Unmarshal(resp.Payload, &c); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal Challenge").Error())
	}
	if c.Status == blockchain.ChallengeProved || c.Status == blockchain.ChallengeFailed {
		return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists,
			"challenge already answered").Error())
	}

	// verify signature
	msg, err := util.GetSigMessage(opt)
	if err != nil {
		return shim.Error(errorx.Internal(err, "failed to get the message to sign").Error())
	}
	targetNode, err := hex.DecodeString(string(c.TargetNode))
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeParam, "pairing challenge wrong target node").Error())
	}
	err = x.checkSign(opt.Signature, targetNode, []byte(msg))
	if err != nil {
		return shim.Error(err.Error())
	}

	// judge if file exists
	resp = x.GetValue(stub, []string{c.FileID})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "File not found: %s", resp.Message).Error())
	}
	// unmarshal file
	var file blockchain.File
	if err := json.Unmarshal(resp.Payload, &file); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File").Error())
	}

	c.AnswerTime = opt.AnswerTime
	c.Status = blockchain.ChallengeProved

	// sig verification
	var verifyErr error
	if c.ChallengeAlgorithm == types.PairingChallengeAlgorithm {
		// verify pairing based challenge
		v, err := xchainClient.VerifyPairingProof(opt.Sigma, opt.Mu, file.RandV, file.RandU, file.PdpPubkey, c.Indices, c.Vs)
		if err != nil || !v {
			e := fmt.Errorf("verify pairing based challenge proof failed: %v", err)
			verifyErr = errorx.NewCode(e, errorx.ErrCodeCrypto, "verification failed")
			c.Status = blockchain.ChallengeFailed
		}
	} else if c.ChallengeAlgorithm == types.MerkleChallengeAlgorithm {
		var aopt ctype.AnswerCalculateOptions
		if err := json.Unmarshal(opt.Proof, &aopt); err != nil {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal Challenge").Error())
		}
		eh := xchainClient.GetMerkleRoot(aopt.RangeHashes)
		cOpt := ctype.CalculateOptions{
			RangeHash: eh,
			Timestamp: aopt.Timestamp,
		}
		proof := challenging.Calculate(&cOpt)
		hashOfProof := xchainClient.HashUsingSha256(proof)
		if !bytes.Equal(hashOfProof, c.HashOfProof) {
			e := fmt.Errorf("hash not equal, supposed to be %v, got: %v", c.HashOfProof, hashOfProof)
			verifyErr = errorx.NewCode(e, errorx.ErrCodeCrypto, "verification failed")
			c.Status = blockchain.ChallengeFailed
		}
	} else {
		return shim.Error(errorx.New(errorx.ErrCodeParam, "bad param:opt-challengeAlgorithm").Error())
	}

	// marshal challenge
	s, err := json.Marshal(c)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Challenge").Error())
	}
	// update challengeID-challenge on chain
	if resp := x.SetValue(stub, []string{index, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to update ChallengeID-Challenge on chain: %s", resp.Message).Error())
	}

	if c.Status == blockchain.ChallengeProved {
		return shim.Success([]byte("answered"))
	}
	return shim.Success([]byte(verifyErr.Error()))
}

// GetChallengeByID query challenge result
// args = {id}
func (x *Xdata) GetChallengeByID(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("incorrect arguments. expecting ChallengeID")
	}

	// get challenge result by challenge id
	index := packChallengeIndex(args[0])
	resp := x.GetValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "challenge not found: %s", resp.Message).Error())
	}
	return shim.Success(resp.Payload)
}

// GetChallengeNum get challenges number given filter
// args = {GetChallengeNumOptions}
func (x *Xdata) GetChallengeNum(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("incorrect arguments. expecting GetChallengeNumOptions")
	}

	// unmarshal opt
	var opt blockchain.GetChallengeNumOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal GetChallengeNumOptions").Error())
	}

	// pack prefix
	prefix, attr := packChallengeFilter(nil, opt.TargetNode)
	// get iter by prefix
	iterator, err := stub.GetStateByPartialCompositeKey(prefix, attr)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	var total uint64 = 0
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		index := packChallengeIndex(string(queryResponse.Value))
		resp := x.GetValue(stub, []string{index})
		if len(resp.Payload) == 0 {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeNotFound,
				"the Challenge[%x] not found: %s", queryResponse.Value, resp.Message).Error())
		}

		var c blockchain.Challenge
		if err = json.Unmarshal(resp.Payload, &c); err != nil {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal Challenge").Error())
		}
		if c.ChallengeTime < opt.TimeStart || c.ChallengeTime > opt.TimeEnd {
			continue
		}
		if len(opt.Status) != 0 && c.Status != opt.Status {
			continue
		}

		total += 1
	}

	return shim.Success([]byte(strconv.FormatUint(total, 10)))
}
