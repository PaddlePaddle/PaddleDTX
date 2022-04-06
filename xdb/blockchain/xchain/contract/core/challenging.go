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
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/monitor/challenging"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var xchainClient = new(fl_crypto.XchainCryptoClient)

// ListChallengeRequests lists challenge requests on blockchain
func (x *Xdata) ListChallengeRequests(ctx code.Context) code.Response {
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.ListChallengeOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListChallengeOptions"))
	}

	// pack prefix
	prefix := packChallengeFilter(opt.FileOwner, opt.TargetNode)

	// get iter by prefix
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	var cs []blockchain.Challenge
	for iter.Next() {
		if opt.Limit > 0 && int64(len(cs)) >= opt.Limit {
			break
		}
		index := packChallengeIndex(string(iter.Value()))
		s, err := ctx.GetObject([]byte(index))
		if err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound,
				"the Challenge[%x] not found", iter.Value()))
		}

		var c blockchain.Challenge
		if err = json.Unmarshal(s, &c); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal Challenge"))
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
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Challenges"))
	}
	return code.OK(s)
}

//ChallengeRequest sets a challenge request onto blockchain
func (x *Xdata) ChallengeRequest(ctx code.Context) code.Response {
	var opt blockchain.ChallengeRequestOptions
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ChallengeRequestOptions"))
	}

	// judge if id exists
	index := packChallengeIndex(opt.ChallengeID)
	if _, err := ctx.GetObject([]byte(index)); err == nil {
		return code.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated ChallengeID"))
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
		if err := x.pairingOptCheck(opt); err != nil {
			return code.Error(err)
		}
		c.SliceIDs = opt.SliceIDs
		c.Indices = opt.Indices
		c.Round = opt.Round
		c.RandThisRound = opt.RandThisRound
		c.Vs = opt.Vs

	} else if opt.ChallengeAlgorithm == types.MerkleChallengeAlgorithm {
		if err := x.merkleOptCheck(opt); err != nil {
			return code.Error(err)
		}
		c.SliceID = opt.SliceID
		c.Ranges = opt.Ranges
		c.HashOfProof = opt.HashOfProof
	} else {
		return code.Error(errorx.New(errorx.ErrCodeParam, "bad param:opt-challengeAlgorithm"))
	}

	// marshal challenge
	s, err := json.Marshal(c)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal Challenge"))
	}
	// set challengeID-challenge on xchain
	if err = ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"failed to set ChallengeID-Challenge on xchain"))
	}

	// set index40owner-challengeID on chain
	index4Owner := packChallengeIndex4Owner(&c)
	if err = ctx.PutObject([]byte(index4Owner), []byte(c.ID)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"failed to set index40owner-ChallengeID on xchain"))
	}
	// set index4Target-challengeID on chain
	index4Target := packChallengeIndex4Target(&c)
	if err = ctx.PutObject([]byte(index4Target), []byte(c.ID)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"failed to set index4Target-ChallengeID on xchain"))
	}

	return code.OK([]byte("requested"))
}

// ChallengeAnswer sets a challengeAnswer on chain
func (x *Xdata) ChallengeAnswer(ctx code.Context) code.Response {
	var opt blockchain.ChallengeAnswerOptions
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ChallengeAnswerOptions"))
	}
	// judge if challenge exists
	index := packChallengeIndex(opt.ChallengeID)
	s, err := ctx.GetObject([]byte(index))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "Challenge not found"))
	}
	// unmarshal challenge
	var c blockchain.Challenge
	if err = json.Unmarshal(s, &c); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal Challenge"))
	}
	if c.Status == blockchain.ChallengeProved || c.Status == blockchain.ChallengeFailed {
		return code.Error(errorx.New(errorx.ErrCodeAlreadyExists,
			"challenge already answered"))
	}

	// judge if file exists
	f, err := ctx.GetObject([]byte(c.FileID))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "File not found"))
	}
	// unmarshal file
	var file blockchain.File
	if err = json.Unmarshal(f, &file); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File"))
	}

	c.AnswerTime = opt.AnswerTime
	c.Status = blockchain.ChallengeProved

	// sig verification
	var verifyErr error
	if c.ChallengeAlgorithm == types.PairingChallengeAlgorithm {
		if err := x.pairingAnswerOptCheck(opt, c); err != nil {
			return code.Error(err)
		}
		// verify pairing based challenge
		v, err := xchainClient.VerifyPairingProof(opt.Sigma, opt.Mu, file.RandV, file.RandU, file.PdpPubkey, c.Indices, c.Vs)
		if err != nil || !v {
			ctx.Logf("bad proof, pairing challenge answer wrong, err:%v", err)
			e := fmt.Errorf("verify pairing based challenge proof failed: %v", err)
			verifyErr = errorx.NewCode(e, errorx.ErrCodeCrypto, "verification failed")
			c.Status = blockchain.ChallengeFailed
		}
	} else if c.ChallengeAlgorithm == types.MerkleChallengeAlgorithm {
		if err := x.merkleAnswerOptCheck(opt, c); err != nil {
			return code.Error(err)
		}
		var aopt ctype.AnswerCalculateOptions
		if err = json.Unmarshal(opt.Proof, &aopt); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal Challenge"))
		}
		eh := xchainClient.GetMerkleRoot(aopt.RangeHashes)
		cOpt := ctype.CalculateOptions{
			RangeHash: eh,
			Timestamp: aopt.Timestamp,
		}
		proof := challenging.Calculate(&cOpt)
		hashOfProof := xchainClient.HashUsingSha256(proof)
		if !bytes.Equal(hashOfProof, c.HashOfProof) {
			ctx.Logf("bad proof, hash supposed to be %v, got: %v", c.HashOfProof, hashOfProof)
			e := fmt.Errorf("hash not equal, supposed to be %v, got: %v", c.HashOfProof, hashOfProof)
			verifyErr = errorx.NewCode(e, errorx.ErrCodeCrypto, "verification failed")
			c.Status = blockchain.ChallengeFailed
		}
	} else {
		return code.Error(errorx.New(errorx.ErrCodeParam, "bad param:opt-challengeAlgorithm"))
	}

	// marshal challenge
	s, err = json.Marshal(c)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Challenge"))
	}
	// update challengeID-challenge on xchain
	if err = ctx.PutObject([]byte(index), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain,
			"failed to update ChallengeID-Challenge on xchain"))
	}

	if c.Status == blockchain.ChallengeProved {
		return code.OK([]byte("answered"))
	}
	return code.OK([]byte(verifyErr.Error()))
}

// GetChallengeByID queries challenge result
func (x *Xdata) GetChallengeByID(ctx code.Context) code.Response {
	// get id
	id, ok := ctx.Args()["id"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}
	// get challenge result by challenge id
	index := packChallengeIndex(string(id))
	s, err := ctx.GetObject([]byte(index))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "challenge not found"))
	}
	return code.OK(s)
}

// GetChallengeNum gets number of challenges with given filter
func (x *Xdata) GetChallengeNum(ctx code.Context) code.Response {
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.GetChallengeNumOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal GetChallengeNumOptions"))
	}

	// pack prefix
	prefix := packChallengeFilter(nil, opt.TargetNode)

	// get iter by prefix
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	var total uint64 = 0
	for iter.Next() {
		index := packChallengeIndex(string(iter.Value()))
		s, err := ctx.GetObject([]byte(index))
		if err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound,
				"the Challenge[%x] not found", iter.Value()))
		}

		var c blockchain.Challenge
		if err = json.Unmarshal(s, &c); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal Challenge"))
		}
		if c.ChallengeTime < opt.TimeStart || c.ChallengeTime > opt.TimeEnd {
			continue
		}
		if len(opt.Status) != 0 && c.Status != opt.Status {
			continue
		}

		total += 1
	}

	return code.OK([]byte(strconv.FormatUint(total, 10)))
}

func (x *Xdata) pairingOptCheck(opt blockchain.ChallengeRequestOptions) error {
	// sig verification
	challengeOpt := blockchain.ChallengeRequestOptions{
		ChallengeID:        opt.ChallengeID,
		FileOwner:          opt.FileOwner,
		TargetNode:         opt.TargetNode,
		FileID:             opt.FileID,
		SliceIDs:           opt.SliceIDs,
		ChallengeTime:      opt.ChallengeTime,
		Indices:            opt.Indices,
		Vs:                 opt.Vs,
		Round:              opt.Round,
		RandThisRound:      opt.RandThisRound,
		ChallengeAlgorithm: opt.ChallengeAlgorithm,
	}
	content, err := json.Marshal(challengeOpt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal Challenge")
	}
	if len(opt.FileOwner) != ecdsa.PublicKeyLength || len(opt.Sig) != ecdsa.SignatureLength {
		return errorx.New(errorx.ErrCodeParam, "bad challenges")
	}
	var pubkey [ecdsa.PublicKeyLength]byte
	var sig [ecdsa.SignatureLength]byte
	copy(pubkey[:], opt.FileOwner)
	copy(sig[:], opt.Sig)
	if err := ecdsa.Verify(pubkey, xchainClient.HashUsingSha256(content), sig); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeBadSignature, "failed to verify Challenge")
	}
	return nil
}

func (x *Xdata) merkleOptCheck(opt blockchain.ChallengeRequestOptions) error {
	// sig verification
	challengeOpt := blockchain.ChallengeRequestOptions{
		ChallengeID:        opt.ChallengeID,
		FileOwner:          opt.FileOwner,
		TargetNode:         opt.TargetNode,
		FileID:             opt.FileID,
		SliceID:            opt.SliceID,
		Ranges:             opt.Ranges,
		ChallengeTime:      opt.ChallengeTime,
		HashOfProof:        opt.HashOfProof,
		ChallengeAlgorithm: opt.ChallengeAlgorithm,
	}

	content, err := json.Marshal(challengeOpt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal Challenge")
	}
	if len(opt.FileOwner) != ecdsa.PublicKeyLength || len(opt.Sig) != ecdsa.SignatureLength {
		return errorx.New(errorx.ErrCodeParam, "bad challenges")
	}
	var pubkey [ecdsa.PublicKeyLength]byte
	var sig [ecdsa.SignatureLength]byte
	copy(pubkey[:], opt.FileOwner)
	copy(sig[:], opt.Sig)
	if err := ecdsa.Verify(pubkey, xchainClient.HashUsingSha256(content), sig); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeBadSignature, "failed to verify Challenge")
	}
	return nil
}

func (x *Xdata) pairingAnswerOptCheck(opt blockchain.ChallengeAnswerOptions, c blockchain.Challenge) error {
	digest := []byte(c.ID)
	digest = append(digest, opt.Sigma...)
	digest = append(digest, opt.Mu...)
	digest = xchainClient.HashUsingSha256(digest)
	targetNode, err := hex.DecodeString(string(c.TargetNode))
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeParam, "pairing challenge wrong target node")
	}
	if len(targetNode) != ecdsa.PublicKeyLength || len(opt.Sig) != ecdsa.SignatureLength {
		return errorx.New(errorx.ErrCodeParam, "pairing challenge bad proof")
	}
	var pubkey [ecdsa.PublicKeyLength]byte
	var sig [ecdsa.SignatureLength]byte
	copy(pubkey[:], targetNode[:])
	copy(sig[:], opt.Sig[:])
	if err := ecdsa.Verify(pubkey, digest, sig); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeBadSignature, "pairing challenge signature verification failed")
	}
	return nil
}

func (x *Xdata) merkleAnswerOptCheck(opt blockchain.ChallengeAnswerOptions, c blockchain.Challenge) error {
	targetNode, err := hex.DecodeString(string(c.TargetNode))
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeParam, "merkle wrong target node")
	}
	if len(targetNode) != ecdsa.PublicKeyLength || len(opt.Sig) != ecdsa.SignatureLength {
		return errorx.New(errorx.ErrCodeParam, "merkle bad proof")
	}
	var pubkey [ecdsa.PublicKeyLength]byte
	var sig [ecdsa.SignatureLength]byte
	copy(pubkey[:], targetNode[:])
	copy(sig[:], opt.Sig[:])
	if err := ecdsa.Verify(pubkey, opt.Proof, sig); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeBadSignature, "merkle signature verification failed")
	}
	return nil
}
