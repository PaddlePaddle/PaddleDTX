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

package common

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var (
	ChallengeFileSuffix     = "_sigmas"
	pairingChallSegmentSize = 100
)

// GetMCRange generate and save more merkle challenges
func GetMCRange(challenger CommonChallenger, fileID string, es encryptor.EncryptedSlice, expireTime,
	startTime, interval int64) (m ctype.Material, err error) {

	rangeAmount := math.Ceil(float64(expireTime-startTime) / float64(interval))

	mRangeList, _ := challenger.Setup(es.CipherText, int(rangeAmount))
	sliceMaterial := ctype.Material{
		FileID:  fileID,
		SliceID: es.SliceID,
		NodeID:  es.NodeID,
		Ranges:  mRangeList,
	}
	return sliceMaterial, nil
}

// SaveMerkleChallenger save challenge material
func SaveMerkleChallenger(challenger CommonChallenger, challengingMaterial []ctype.Material) error {
	// Write challenging meta
	if err := challenger.Save(challengingMaterial); err != nil {
		return errorx.Wrap(err, "failed to save challenging materials")
	}
	return nil
}

// AddFileNewMerkleChallenge add file merkle challenges when extend file expire time
func AddFileNewMerkleChallenge(ctx context.Context, challenger CommonChallenger, chain CommonChain, copier CommonCopier,
	chalEncryptor CommonEncryptor, file blockchain.File, startTime, interval int64, logger *logrus.Entry) error {
	var challengingMaterial []ctype.Material
	nodes, err := GetHealthNodes(chain)
	if err != nil {
		return err
	}
	nodesMap := ToNodeHsMap(nodes)

	selectedNodes := make(map[string][]string)
	for _, slice := range file.Slices {
		selectedNodes[slice.ID] = append(selectedNodes[slice.ID], string(slice.NodeID))
	}

	for _, target := range file.Slices {
		// pull slice
		success := false
		node, exist := nodesMap[string(target.NodeID)]

		if !exist {
			logger.WithField("node_id", string(target.NodeID)).Warn("abnormal node")
		}
		for exist {
			r, err := copier.Pull(ctx, target.ID, file.ID, &node)
			if err != nil {
				logger.WithError(err).Warn("failed to pull slice")
				break
			}
			// read
			cipherText, err := ioutil.ReadAll(r)
			if err != nil {
				logger.WithError(err).Warn("failed to read slice from target node")
				r.Close()
				break
			}
			r.Close()

			es := encryptor.EncryptedSlice{
				EncryptedSliceMeta: encryptor.EncryptedSliceMeta{
					SliceID:    target.ID,
					NodeID:     target.NodeID,
					CipherHash: target.CipherHash,
					Length:     target.Length,
				},
				CipherText: cipherText,
			}
			m, err := GetMCRange(challenger, file.ID, es, file.ExpireTime, startTime, interval)
			if err != nil {
				logger.WithError(err).Warn("failed to get merkle challenge range")
				break
			}
			challengingMaterial = append(challengingMaterial, m)
			success = true
			break
		}

		if !success {
			// pull from other healthy nodes
			for _, n := range selectedNodes[target.ID] {
				newNode, exist := nodesMap[n]
				if exist {
					r, err := copier.Pull(ctx, target.ID, file.ID, &newNode)
					if err != nil {
						logger.WithField("node_id", n).WithError(err).Warn("failed to pull slice from other node")
						continue
					}
					// decrypt
					opt := &encryptor.RecoverOptions{
						FileID:  file.ID,
						SliceID: target.ID,
						NodeID:  newNode.ID,
					}
					plain, err := chalEncryptor.Recover(r, opt)
					if err != nil {
						logger.WithField("node_id", n).WithError(err).Warn("failed to recover slice from other node")
						r.Close()
						continue
					}
					r.Close()
					// encrypt by target nodeID
					encOpt := &encryptor.EncryptOptions{
						FileID:  file.ID,
						SliceID: target.ID,
						NodeID:  target.NodeID,
					}
					cipher, err := chalEncryptor.Encrypt(bytes.NewReader(plain), encOpt)
					if err != nil {
						logger.WithField("node_id", string(target.NodeID)).WithError(err).Warn("failed to encrypt slice by target node")
						continue
					}
					// get merkle challenge material
					m, err := GetMCRange(challenger, file.ID, cipher, file.ExpireTime, startTime, interval)
					if err != nil {
						logger.WithError(err).Warn("failed to get merkle challenge range")
						continue
					}
					logger.WithFields(logrus.Fields{
						"target_node": string(target.NodeID),
						"new_node":    n,
					}).Debug("got merkle challenge by pulling slice from other node")
					challengingMaterial = append(challengingMaterial, m)
					break
				}
			}
		}
	}

	if len(file.Slices) != len(challengingMaterial) {
		return errorx.New(errorx.ErrCodeInternal, "failed to add new merkle challenges, slices number and challenge material number not equal")
	}
	return SaveMerkleChallenger(challenger, challengingMaterial)
}

// AddSlicesNewMerkleChallenge add merkle challenges for a storage node when slice migrates to new node
func AddSlicesNewMerkleChallenge(challenger CommonChallenger, file blockchain.File, expandSlices []encryptor.EncryptedSlice,
	interval int64, logger *logrus.Entry) error {
	var challengingMaterial []ctype.Material

	if len(expandSlices) == 0 {
		logger.WithFields(logrus.Fields{
			"file_id":     file.ID,
			"file_Slices": len(file.Slices),
		}).Info("no slice need add merkle challenge")
		return nil
	}
	if len(file.Slices) == 0 {
		return errorx.New(errorx.ErrCodeInternal, "failed to add slices merkle challenge, file-slices is empty")
	}
	var mSlices []string
	for _, eslice := range expandSlices {
		m, err := GetMCRange(challenger, file.ID, eslice, file.ExpireTime, time.Now().UnixNano(), interval)
		if err != nil {
			return err
		}
		challengingMaterial = append(challengingMaterial, m)
		mSlices = append(mSlices, eslice.SliceID)
	}
	if len(expandSlices) != len(challengingMaterial) {
		return errorx.New(errorx.ErrCodeInternal, "failed to add new slice merkle challenge")
	}
	if err := SaveMerkleChallenger(challenger, challengingMaterial); err != nil {
		return errorx.Wrap(err, "save merkle challenge error")
	}
	logger.WithFields(logrus.Fields{
		"file_id":           file.ID,
		"new_merkle_slices": mSlices,
	}).Info("success add slice merkle challenge")
	return nil
}

// AnswerPairingChallenge answers pairing based random challenge based on file content
func AnswerPairingChallenge(content, indices, randVs, sigmas [][]byte, rand []byte) ([]byte, []byte, error) {
	return xchainClient.ProvePairingChallenge(content, indices, randVs, sigmas, rand)
}

// AddSlicesNewPairingChallenge generate pairing based challenge material for each slice and push it to storage node
// 1. get roundStart and roundEnd by start&end time
// 2. calculate sigmaI for each round
// 3. pack all sigmas and push to storage node
// interval, start, end are in nanoseconds
func AddSlicesNewPairingChallenge(ctx context.Context, pairingConf types.PairingChallengeConf, copier CommonCopier, slices []encryptor.EncryptedSlice,
	file blockchain.File, chain CommonChain, sourceID string, interval, start, end int64, existSigmas []map[int64][]byte, logger *logrus.Entry) error {

	// for each challenge round from start to end round, generate sigmas and push it to storage node
	roundStart := start / interval
	roundEnd := end / interval
	// chop challenge calculation into several segments
	segments := (roundEnd - roundStart) / int64(pairingChallSegmentSize)
	// get slice idx map, from slice+nodeID to slice idx
	idxMap := getSlicesIdxMap(file)
	// list nodes from blockchain
	allNodes, err := chain.ListNodes()
	if err != nil {
		return errorx.Wrap(err, "failed to list nodes from blockchain")
	}
	nodesMap := ToNodesMap(allNodes)

	var pushErr error
	wg := sync.WaitGroup{}
	for idx, slice := range slices {
		wg.Add(1)
		go func(idx int, slice encryptor.EncryptedSlice) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}
			node, exist := nodesMap[string(slice.NodeID)]
			if !exist {
				pushErr = errorx.Wrap(err, "abnormal storage node")
				return
			}

			// if sigmas already exist(when extending expire time), update original sigmas
			sigmas := make(map[int64][]byte)
			if existSigmas != nil && len(existSigmas) > idx {
				for key, value := range existSigmas[idx] {
					sigmas[key] = value
				}
			}

			// get index for this slice among all slices for the storage node
			idxInSlice, exist := idxMap[slice.SliceID+string(slice.NodeID)]
			if !exist {
				pushErr = errorx.New(errorx.ErrCodeInternal, "failed to find target slice idx")
				return
			}

			var sigmasLock sync.Mutex
			swg := sync.WaitGroup{}
			for k := int64(0); k <= segments; k++ {
				swg.Add(1)
				go func(k int64) {
					defer swg.Done()
					select {
					case <-ctx.Done():
						return
					default:
					}
					startRoundThisSegment := roundStart + k*int64(pairingChallSegmentSize)
					endRoundThisSegment := startRoundThisSegment + int64(pairingChallSegmentSize)
					// include the last round
					if endRoundThisSegment > roundEnd {
						endRoundThisSegment = roundEnd + 1
					}

					for i := startRoundThisSegment; i < endRoundThisSegment; i++ {
						// generate sigma for this round
						sigmaI, err := xchainClient.CalculateSigmaI(slice.CipherText, idxInSlice, pairingConf.RandV, pairingConf.RandU, pairingConf.Privkey, i)
						if err != nil {
							logger.WithFields(logrus.Fields{
								"file_id":      file.ID,
								"slice_id":     slice.SliceID,
								"storage_node": string(slice.NodeID),
							}).Error("failed to calculate sigmaI for pairing based challenge")
							pushErr = errorx.Wrap(err, "failed to calculate sigmaI")
							return
						}
						sigmasLock.Lock()
						sigmas[i] = sigmaI
						sigmasLock.Unlock()
					}
				}(k)
			}
			swg.Wait()

			if pushErr != nil {
				return
			}
			// push sigmas to storage node
			sliceSigmaID := GetSliceSigmasID(slice.SliceID)
			sigmasBytes, err := SigmasToBytes(sigmas)
			if err != nil {
				pushErr = errorx.Wrap(err, "failed to marshal sigmas")
				return
			}

			if err := copier.Push(ctx, sliceSigmaID, sourceID, bytes.NewReader(sigmasBytes), &node); err != nil {
				pushErr = errorx.Wrap(err, "failed to push pairing based challenge material")
				return
			}

			logger.WithFields(logrus.Fields{
				"file_id":      file.ID,
				"slice_id":     slice.SliceID,
				"storage_node": string(slice.NodeID),
			}).Info("successfully generated and pushed pairing based challenge material")
		}(idx, slice)
	}
	wg.Wait()
	return pushErr
}

// AddFilePairingChallenges add file pairing based challenges when extend file expire time
// 1. pull slice and sigmas from storage node
// 2. generate new sigmas and push all sigmas to storage node
func AddFilePairingChallenges(ctx context.Context, pairingConf types.PairingChallengeConf, chain CommonChain, copier CommonCopier,
	file blockchain.File, sourceID string, oldExp, interval int64, logger *logrus.Entry) error {

	nodes, err := chain.ListNodes()
	if err != nil {
		return err
	}
	nodesMap := ToNodesMap(nodes)

	var addErr error
	wg := sync.WaitGroup{}
	for _, target := range file.Slices {
		wg.Add(1)
		go func(target blockchain.PublicSliceMeta) {
			defer wg.Done()

			// pull slice
			node, exist := nodesMap[string(target.NodeID)]
			if !exist {
				logger.WithField("node_id", string(target.NodeID)).Error("abnormal node")
				addErr = errorx.New(errorx.ErrCodeNotFound, "storage node not found in chain")
				return
			}

			r, err := copier.Pull(ctx, target.ID, file.ID, &node)
			if err != nil {
				logger.WithField("slice_id", target.ID).WithError(err).Error("failed to pull slice")
				addErr = errorx.NewCode(err, errorx.ErrCodeInternal, "failed to pull slice")
				return
			}
			// read slice content
			cipherText, err := ioutil.ReadAll(r)
			if err != nil {
				r.Close()
				logger.WithField("slice_id", target.ID).WithError(err).Error("failed to read slice from target node")
				addErr = errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read slice from target node")
				return
			}

			// pull sigmas
			sliceSigma := GetSliceSigmasID(target.ID)
			r, err = copier.Pull(ctx, sliceSigma, file.ID, &node)
			if err != nil {
				logger.WithField("slice_id", target.ID).WithError(err).Error("failed to pull slice sigmas")
				addErr = errorx.NewCode(err, errorx.ErrCodeInternal, "failed to pull slice sigmas")
				return
			}
			// read sigmas content
			sigmas, err := ioutil.ReadAll(r)
			if err != nil {
				r.Close()
				logger.WithField("slice_id", target.ID).WithError(err).Error("failed to read slice sigmas from target node")
				addErr = errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read slice sigmas from target node")
				return
			}
			r.Close()

			// retrieve sigma map
			sigmasMap, err := SigmasFromBytes(sigmas)
			if err != nil {
				logger.WithField("slice_id", target.ID).WithError(err).Error("failed to unmarshal sigmas from bytes")
				addErr = errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal sigmas from bytes")
				return
			}

			es := encryptor.EncryptedSlice{
				EncryptedSliceMeta: encryptor.EncryptedSliceMeta{
					SliceID:    target.ID,
					NodeID:     target.NodeID,
					CipherHash: target.CipherHash,
					Length:     target.Length,
				},
				CipherText: cipherText,
			}
			if err := AddSlicesNewPairingChallenge(ctx, pairingConf, copier, []encryptor.EncryptedSlice{es}, file, chain, sourceID,
				interval, oldExp, file.ExpireTime, []map[int64][]byte{sigmasMap}, logger); err != nil {
				logger.WithField("slice_id", target.ID).WithError(err).Error("failed to gen or push paring based challenge material")
				addErr = errorx.NewCode(err, errorx.ErrCodeInternal, "failed to gen or push paring based challenge material")
				return
			}
		}(target)
	}
	wg.Wait()

	return addErr
}

// GetPairingChallengeRound get challenge round number for pairing based challenge
// round = timeNow / challengeInterval (in nanoseconds)
func GetPairingChallengeRound(interval int64) int64 {
	nowMinutes := time.Now().UnixNano()
	return nowMinutes / interval
}

// getSlicesIdxMap get a map from slice and storage node to slice index
func getSlicesIdxMap(file blockchain.File) map[string][]byte {
	idxMap := make(map[string][]byte)

	for _, s := range file.Slices {
		key := s.ID + string(s.NodeID)
		idxMap[key] = big.NewInt(int64(s.SliceIdx)).Bytes()
	}
	return idxMap
}

// GetSliceSigmasID pack file name to store sigmas for a slice
func GetSliceSigmasID(sliceID string) string {
	return sliceID + ChallengeFileSuffix
}

// SigmasToBytes convert a slice's sigma list to bytes
func SigmasToBytes(sigmas map[int64][]byte) ([]byte, error) {
	return json.Marshal(sigmas)
}

// SigmasFromBytes convert sigma list from bytes
func SigmasFromBytes(sigmasBytes []byte) (map[int64][]byte, error) {
	sigmas := make(map[int64][]byte)
	err := json.Unmarshal(sigmasBytes, &sigmas)
	return sigmas, err
}
