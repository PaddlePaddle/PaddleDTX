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
	"errors"
	"math/big"
	"math/rand"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/hash"
)

// loopRequest publishes challenge requests if local node is dataOwner-node,
//  and blocks current routine
func (c *ChallengingMonitor) loopRequest(ctx context.Context) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(c.PrivateKey)
	rand.Seed(time.Now().UnixNano())
	challengeAlgorithm, _ := c.challengeDB.GetChallengeConf()

	l := logger.WithField("runner", "request loop")
	defer l.Info("runner stopped")

	ticker := time.NewTicker(c.RequestInterval)
	defer ticker.Stop()

	c.doneLoopReqC = make(chan struct{})
	defer close(c.doneLoopReqC)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		nsopt := blockchain.ListNsOptions{
			Owner:       pubkey[:],
			TimeEnd:     time.Now().UnixNano(),
			CurrentTime: time.Now().UnixNano(),
		}
		nss, err := c.blockchain.ListFileNs(ctx, &nsopt)
		if err != nil {
			l.WithError(err).Warn("failed to list file ns from blockchain")
			continue
		}
		if len(nss) == 0 {
			l.WithField("amount", len(nss)).Debug("file ns list loaded")
			continue
		}
		rand.Seed(time.Now().UnixNano())
		var nsSelected int
		var files []blockchain.File
		selectedList := make(map[int]struct{})
		isContinue := false

		for i := 0; i < len(nss); i++ {
			for {
				nsSelected = rand.Int() % len(nss)
				if _, ok := selectedList[nsSelected]; !ok {
					selectedList[nsSelected] = struct{}{}
					break
				}
			}
			listOpt := blockchain.ListFileOptions{
				Owner:       pubkey[:],
				Namespace:   nss[nsSelected].Name,
				TimeEnd:     time.Now().UnixNano(),
				CurrentTime: time.Now().UnixNano(),
			}
			files, err = c.blockchain.ListFiles(ctx, &listOpt)
			if err != nil {
				l.WithError(err).Warn("failed to list files from blockchain")
				isContinue = true
				break
			}
			if len(files) > 0 || (i == len(nss)-1) {
				l.WithFields(logrus.Fields{
					"ns selected": nss[nsSelected].Name,
					"file num":    len(files),
					"ns index":    i,
				}).Info("file ns selected")
				break
			}
		}
		if isContinue {
			continue
		}

		l.WithField("amount", len(files)).Debug("files loaded")

		if len(files) == 0 {
			continue
		}
		if challengeAlgorithm == types.PDPChallengAlgorithm {
			c.doPDPChallengeRequest(ctx, challengeAlgorithm, files, pubkey, l)
		} else {
			c.doMerkleChallengeRequest(ctx, challengeAlgorithm, files, pubkey, l)
		}
	}
}

func (c *ChallengingMonitor) doPDPChallengeRequest(ctx context.Context, challengeAlgorithm string, files []blockchain.File,
	pubkey ecdsa.PublicKey, l *logrus.Entry) error {

	// select just one file and nodeID
	rand.Seed(time.Now().UnixNano())
	fileSelected := files[rand.Int()%len(files)]
	sliceSelected := fileSelected.Slices[rand.Int()%len(fileSelected.Slices)]
	nodeSelected := sliceSelected.NodeID

	l.WithField("fileID", fileSelected.ID).Info("file selected")

	// find amount of index to generate pdp challenge
	maxIdx := 1
	// map slice idx to slice ID
	sliceMap := make(map[int]string)
	sigmaMap := make(map[int][]byte)
	for _, slice := range fileSelected.Slices {
		if reflect.DeepEqual(slice.NodeID, nodeSelected) {
			if slice.SliceIdx > maxIdx {
				maxIdx = slice.SliceIdx
			}
			sliceMap[slice.SliceIdx] = slice.ID
			sigmaMap[slice.SliceIdx] = slice.SigmaI
		}
	}

	// calculate challenge request
	indices, vs, err := c.challengeDB.GenerateChallenge(maxIdx)
	if err != nil {
		l.WithError(err).Warn("failed GenerateChallenge")
		return err
	}
	var sliceIDs []string
	var sigmas [][]byte
	for _, idx := range indices {
		sliceIDs = append(sliceIDs, sliceMap[int(new(big.Int).SetBytes(idx).Int64())])
		sigmas = append(sigmas, sigmaMap[int(new(big.Int).SetBytes(idx).Int64())])
	}
	if len(sliceIDs) == 0 {
		l.Warn("failed GenerateChallenge, sliceIDs is empty")
		return errorx.New(errorx.ErrCodeInternal, "GenerateChallenge failed")
	}

	// publish challenge request
	requestOpt := blockchain.ChallengeRequestOptions{
		ChallengeID:       uuid.NewString(),
		FileOwner:         pubkey[:],
		TargetNode:        sliceSelected.NodeID,
		FileID:            fileSelected.ID,
		SliceIDs:          sliceIDs,
		ChallengeTime:     time.Now().UnixNano(),
		Indices:           indices,
		Vs:                vs,
		Sigmas:            sigmas,
		ChallengAlgorithm: challengeAlgorithm,
	}

	// sign request
	content, err := json.Marshal(requestOpt)
	if err != nil {
		l.WithField("challenge_id", requestOpt.ChallengeID).WithError(err).Warn("failed to marshal request")
		return err
	}
	sig, err := ecdsa.Sign(c.PrivateKey, hash.Hash(content))
	if err != nil {
		l.WithField("challenge_id", requestOpt.ChallengeID).WithError(err).Warn("failed to sign request")
		return err
	}
	requestOpt.Sig = sig[:]

	if err := c.blockchain.ChallengeRequest(ctx, &requestOpt); err != nil {
		l.WithField("challenge_id", requestOpt.ChallengeID).WithError(err).Warn("failed to publish challenge request")
		return err
	}
	l.WithFields(logrus.Fields{
		"challenge_id": requestOpt.ChallengeID,
		"target_node":  string(requestOpt.TargetNode),
		"indices":      requestOpt.Indices,
		"slices":       requestOpt.SliceIDs,
	}).Info("success to publish challenge request")
	return nil
}

func (c *ChallengingMonitor) doMerkleChallengeRequest(ctx context.Context, challengeAlgorithm string, files []blockchain.File,
	pubkey ecdsa.PublicKey, l *logrus.Entry) error {
	// select just one slice
	fileSelected := files[rand.Int()%len(files)]
	sliceSelected := fileSelected.Slices[rand.Int()%len(fileSelected.Slices)]

	// take one range
	rangeSelected, err := c.challengeDB.Take(ctx, fileSelected.ID, sliceSelected.ID, sliceSelected.NodeID)
	if err != nil {
		if !errors.Is(err, errorx.ErrNotFound) {
			l.WithError(err).Warn("failed to take range material")
		} else {
			l.Debug("no available range, do nothing")
		}
		return err
	}

	// pre calculate proof
	timestamp := time.Now().UnixNano()
	cOpt := ctype.CalculateOptions{
		RangeHash: rangeSelected.Hash,
		Timestamp: timestamp,
	}
	proof := Calculate(&cOpt)

	hashOfProof := hash.Hash(proof)

	// select some parts of slices and send challenge requests
	var branges []blockchain.Range
	for _, v := range rangeSelected.Ranges {
		branges = append(branges, blockchain.Range{
			Start: v.Start,
			End:   v.End,
		})
	}

	// publish challenge request
	requestOpt := blockchain.ChallengeRequestOptions{
		ChallengeID:       uuid.NewString(),
		FileOwner:         pubkey[:],
		TargetNode:        sliceSelected.NodeID,
		FileID:            fileSelected.ID,
		SliceID:           sliceSelected.ID,
		Ranges:            branges,
		ChallengeTime:     timestamp,
		HashOfProof:       hashOfProof,
		ChallengAlgorithm: challengeAlgorithm,
	}

	// sign request
	content, err := json.Marshal(requestOpt)
	if err != nil {
		l.WithField("challenge_id", requestOpt.ChallengeID).WithError(err).Warn("failed to marshal request")
		return err
	}
	sig, err := ecdsa.Sign(c.PrivateKey, hash.Hash(content))
	if err != nil {
		l.WithField("challenge_id", requestOpt.ChallengeID).WithError(err).Warn("failed to sign request")
		return err
	}
	requestOpt.Sig = sig[:]

	if err := c.blockchain.ChallengeRequest(ctx, &requestOpt); err != nil {
		l.WithField("challenge_id", requestOpt.ChallengeID).WithError(err).Warn("failed to publish challenge request")
		return err
	}

	l.WithFields(logrus.Fields{
		"challenge_id":  requestOpt.ChallengeID,
		"target_node":   string(sliceSelected.NodeID),
		"fileSelected":  fileSelected.ID,
		"sliceSelected": sliceSelected.ID,
		"range_list":    requestOpt.Ranges,
	}).Info("success to publish merkle challenge request")
	return nil
}
