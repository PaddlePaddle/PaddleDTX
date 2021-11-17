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
	"io"
	"io/ioutil"
	"math"
	"time"

	"github.com/sirupsen/logrus"
	fl_crypto "github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var xchainClient = new(fl_crypto.XchainCryptoClient)

// MerkleChallenger defines Merkle-Tree based challenger
type MerkleChallenger interface {
	Setup(sliceData []byte, rangeAmount int) ([]ctype.RangeHash, error)
	NewSetup(sliceData []byte, rangeAmount int, merkleMaterialQueue chan<- ctype.Material, cm ctype.Material) error
	Save(ctx context.Context, cms []ctype.Material) error
	Take(ctx context.Context, fileID string, sliceID string, nodeID []byte) (ctype.RangeHash, error)

	GetChallengeConf() (string, types.PDP)
}

// ChallengeEncryptor defines encryptor for file encryption and decryption when adding more merkle challenges
type ChallengeEncryptor interface {
	Encrypt(ctx context.Context, r io.Reader, opt *encryptor.EncryptOptions) (
		encryptor.EncryptedSlice, error)
	Recover(ctx context.Context, r io.Reader, opt *encryptor.RecoverOptions) (
		[]byte, error)
}

// GetMCRange generate and save more merkle challenges
func GetMCRange(challenger MerkleChallenger, fileID string, es encryptor.EncryptedSlice, expireTime,
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

// GetAddFileMCRange generate and save merkle challenges when uploading a file
func GetAddFileMCRange(challenger MerkleChallenger, fileID string, es encryptor.EncryptedSlice, expireTime,
	startTime, interval int64, merkleMaterialQueue chan<- ctype.Material) (err error) {

	rangeAmount := math.Ceil(float64(expireTime-startTime) / float64(interval))
	sliceMaterial := ctype.Material{
		FileID:  fileID,
		SliceID: es.SliceID,
		NodeID:  es.NodeID,
	}
	err = challenger.NewSetup(es.CipherText, int(rangeAmount), merkleMaterialQueue, sliceMaterial)

	return err
}

// SaveMerkleChallenger save challenge material
func SaveMerkleChallenger(ctx context.Context, challenger MerkleChallenger, challengingMaterial []ctype.Material) error {
	// Write challenging meta
	if err := challenger.Save(ctx, challengingMaterial); err != nil {
		return errorx.Wrap(err, "failed to save challenging materials")
	}
	return nil
}

// AddFileNewMerkleChallenge add file merkle challenges when extend file expire time
func AddFileNewMerkleChallenge(ctx context.Context, challenger MerkleChallenger, chain HealthChain, copier MigrateCopier,
	chalEncryptor ChallengeEncryptor, file blockchain.File, startTime, interval int64, logger *logrus.Entry) error {
	var challengingMaterial []ctype.Material
	nodes, err := GetHealthNodes(ctx, chain)
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
						SliceID: target.ID,
						NodeID:  newNode.ID,
					}
					plain, err := chalEncryptor.Recover(ctx, r, opt)
					if err != nil {
						logger.WithField("node_id", n).WithError(err).Warn("failed to recover slice from other node")
						r.Close()
						continue
					}
					r.Close()
					// encrypt by target nodeID
					encOpt := &encryptor.EncryptOptions{
						SliceID: target.ID,
						NodeID:  target.NodeID,
					}
					cipher, err := chalEncryptor.Encrypt(ctx, bytes.NewReader(plain), encOpt)
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
	if err := SaveMerkleChallenger(ctx, challenger, challengingMaterial); err != nil {
		return err
	}
	return nil
}

// AddSlicesNewMerkleChallenge add merkle challenges for a storage node when slice migrates to new node
func AddSlicesNewMerkleChallenge(ctx context.Context, challenger MerkleChallenger, copier MigrateCopier, file blockchain.File,
	expandSlices []encryptor.EncryptedSlice, interval int64, logger *logrus.Entry) error {
	var challengingMaterial []ctype.Material

	if len(expandSlices) == 0 {
		logger.WithFields(logrus.Fields{
			"file_id":     file.ID,
			"file.Slices": len(file.Slices),
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
	if err := SaveMerkleChallenger(ctx, challenger, challengingMaterial); err != nil {
		return errorx.Wrap(err, "save merkle challenge error")
	}
	logger.WithFields(logrus.Fields{
		"file_id":           file.ID,
		"new_merkle_slices": mSlices,
	}).Info("success add slice merkle challenge")
	return nil
}

// AnswerPDPChallenge answers pdp random challenge based on file content
func AnswerPDPChallenge(content, indices, randVs, sigmas [][]byte) ([]byte, []byte, error) {
	return xchainClient.ProvePDP(content, indices, randVs, sigmas)
}
