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

package challenging

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// loopAnswer listens challenge requests and answer them in order to prove it's storing related files
//   and blocks current routine
func (c *ChallengingMonitor) loopAnswer(ctx context.Context) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(c.PrivateKey)
	l := logger.WithField("runner", "answer loop")
	defer func() {
		nonce := time.Now().UnixNano()
		m := fmt.Sprintf("%s,%d", pubkey.String(), nonce)
		sig, err := ecdsa.Sign(c.PrivateKey, hash.HashUsingSha256([]byte(m)))
		if err != nil {
			l.WithError(err).Error("failed to sign node")
			return
		}
		nodeOpts := &blockchain.NodeOperateOptions{
			NodeID: []byte(pubkey.String()),
			Nonce:  nonce,
			Sig:    sig[:],
		}
		if err := c.blockchain.NodeOffline(nodeOpts); err != nil {
			l.WithError(err).Error("failed to offline the node")
		}
		l.Info("node offline")
		l.Info("runner stopped")
	}()

	ticker := time.NewTicker(c.AnswerInterval)
	defer ticker.Stop()

	c.doneLoopAnsC = make(chan struct{})
	defer close(c.doneLoopAnsC)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		// list requests
		queryOpts := blockchain.ListChallengeOptions{
			TargetNode: []byte(pubkey.String()),
			Status:     blockchain.ChallengeToProve,
			TimeEnd:    time.Now().UnixNano(),
		}
		requests, err := c.blockchain.ListChallengeRequests(&queryOpts)
		if err != nil {
			l.WithError(err).Warn("failed to list challenge requests from blockchain")
			continue
		}
		l.WithField("amount", len(requests)).Debug("requests listed")
		if len(requests) == 0 {
			continue
		}
		for _, r := range requests {
			if r.ChallengeAlgorithm == types.PairingChallengeAlgorithm {
				c.doPairingChallengeAnswer(r, l)
			} else if r.ChallengeAlgorithm == types.MerkleChallengeAlgorithm {
				c.doMerkleChallengeAnswer(r, l)
			} else {
				l.WithField("challenge_id", r.ID).Debug("challenge answer failed, algorithm not support")
			}
		}
	}
}

func (c *ChallengingMonitor) doPairingChallengeAnswer(r blockchain.Challenge, l *logrus.Entry) error {
	// answer for each request
	// calculate proof
	l.WithField("challenge_id", r.ID).Infof("indices: %v, slices: %v", r.Indices, r.SliceIDs)
	proof, err := c.doPairingCalculateProof(c.PrivateKey, &r)
	if err != nil {
		l.WithError(err).Warnf("failed to calculate pairing proof for round: %d", r.Round)
		return err
	}

	// publish proof
	answerOpt := blockchain.ChallengeAnswerOptions{
		ChallengeID: r.ID,
		Sigma:       proof.Sigma,
		Mu:          proof.Mu,
		Sig:         proof.Signature,
		AnswerTime:  time.Now().UnixNano(),
	}
	resp, err := c.blockchain.ChallengeAnswer(&answerOpt)
	if err != nil {
		l.WithError(err).Warn("failed to publish answer")
		return err
	}
	if string(resp) != "answered" {
		l.WithField("request_id", r.ID).Errorf("ChallengeAnswer err: %s", string(resp))
	}
	l.WithField("request_id", r.ID).Debug("successfully answered challenge request")
	return nil
}

func (c *ChallengingMonitor) doMerkleChallengeAnswer(r blockchain.Challenge, l *logrus.Entry) error {
	// answer for each request
	// calculate
	proof, err := c.doMerkleCalculation(c.PrivateKey, &r)
	if err != nil {
		l.WithError(err).Warn("failed to calculate merkle proof")
		return err
	}

	// publish proof
	answerOpt := blockchain.ChallengeAnswerOptions{
		ChallengeID: r.ID,
		Proof:       proof.Proof,
		Sig:         proof.Signature,
		AnswerTime:  time.Now().UnixNano(),
	}
	resp, err := c.blockchain.ChallengeAnswer(&answerOpt)
	if err != nil {
		l.WithError(err).Warn("failed to publish answer")
		return err
	}
	if string(resp) != "answered" {
		l.WithField("request_id", r.ID).Errorf("ChallengeAnswer err: %s", string(resp))
	}

	l.WithField("request_id", r.ID).Debug("successfully answered challenge request")
	return err
}

// doPairingCalculateProof calculate proof using stored files and random challenge
func (c *ChallengingMonitor) doPairingCalculateProof(privkey ecdsa.PrivateKey, req *blockchain.Challenge) (randomProof, error) {

	var content [][]byte
	var sigmaContent [][]byte
	for _, sliceID := range req.SliceIDs {
		// read slice content
		dataReader, err := c.sliceStorage.Load(sliceID)
		if err != nil {
			return randomProof{}, errorx.Wrap(err, "failed to load local slice %s", sliceID)
		}
		data, err := ioutil.ReadAll(dataReader)
		if err != nil {
			return randomProof{}, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read slice data")
		}
		content = append(content, data)

		// read sigmas content
		sigmaFile := common.GetSliceSigmasID(sliceID)
		dataReader, err = c.sliceStorage.Load(sigmaFile)
		if err != nil {
			return randomProof{}, errorx.Wrap(err, "failed to load local slice sigmas %s", sliceID)
		}
		data, err = ioutil.ReadAll(dataReader)
		if err != nil {
			return randomProof{}, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read slice sigmas")
		}
		sigmaContent = append(sigmaContent, data)
		dataReader.Close()
	}

	if req.Indices == nil || req.Vs == nil {
		return randomProof{}, errorx.New(errorx.ErrCodeInternal, "invalid challenge")
	}

	// get sigma for each slice with respect to the challenge round
	var sigmas [][]byte
	for _, sigma := range sigmaContent {
		sigmaMap, err := common.SigmasFromBytes(sigma)
		if err != nil {
			return randomProof{}, errorx.New(errorx.ErrCodeInternal, "failed to unmarshal sigmas")
		}
		if len(sigmaMap[req.Round]) == 0 {
			return randomProof{}, errorx.New(errorx.ErrCodeInternal, "empty sigma for this round")
		}
		sigmas = append(sigmas, sigmaMap[req.Round])
	}

	sigma, mu, err := common.AnswerPairingChallenge(content, req.Indices, req.Vs, sigmas, req.RandThisRound)
	if err != nil {
		return randomProof{}, errorx.NewCode(err, errorx.ErrCodeInternal, "AnswerChallenge failed")
	}

	// sign proof
	signMsg := []byte(req.ID)
	signMsg = append(signMsg, sigma...)
	signMsg = append(signMsg, mu...)
	signMsg = hash.HashUsingSha256(signMsg)
	sig, err := ecdsa.Sign(privkey, signMsg)
	if err != nil {
		return randomProof{}, errorx.NewCode(err, errorx.ErrCodeCrypto, "failed to sign")
	}
	rp := randomProof{
		Sigma:     sigma,
		Mu:        mu,
		Signature: sig[:],
	}

	return rp, nil
}

func (c *ChallengingMonitor) doMerkleCalculation(privkey ecdsa.PrivateKey, req *blockchain.Challenge) (rangeProof, error) {

	dataReader, err := c.sliceStorage.Load(req.SliceID)
	if err != nil {
		return rangeProof{}, errorx.Wrap(err, "failed to load local slice %s", req.SliceID)
	}
	defer dataReader.Close()

	data, err := ioutil.ReadAll(dataReader)
	if err != nil {
		return rangeProof{}, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read slice during doMerkleCalculation")
	}
	var hs [][]byte
	for _, rr := range req.Ranges {
		if rr.Start > rr.End {
			return rangeProof{}, errorx.New(errorx.ErrCodeParam,
				"invalid range. start[%d] end[%d]", rr.Start, rr.End)
		}
		if rr.End >= uint64(len(data)) {
			return rangeProof{}, errorx.New(errorx.ErrCodeParam,
				"invalid range. length[%d] end[%d]", len(data), rr.End)
		}
		h := hash.HashUsingSha256(data[rr.Start:rr.End])
		hs = append(hs, h)
	}

	opt := ctype.AnswerCalculateOptions{
		RangeHashes: hs,
		Timestamp:   req.ChallengeTime,
	}

	// sign file info
	proof, err := json.Marshal(opt)
	if err != nil {
		return rangeProof{}, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal merkle answer options")
	}

	sig, err := ecdsa.Sign(privkey, proof)
	if err != nil {
		return rangeProof{}, errorx.NewCode(err, errorx.ErrCodeCrypto, "failed to sign")
	}
	rp := rangeProof{
		Proof:     proof,
		Signature: sig[:],
	}

	return rp, nil
}
