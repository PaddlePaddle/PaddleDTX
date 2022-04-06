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
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// PullAndDec pull a slice from healthy node and decrypt
func PullAndDec(ctx context.Context, copier CommonCopier, encrypt CommonEncryptor,
	slice blockchain.PublicSliceMeta, node *blockchain.Node, fileID string) ([]byte, error) {

	r, err := copier.Pull(ctx, slice.ID, fileID, node)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r != nil {
			r.Close()
		}
	}()
	// read slice ciphertext
	cipherText, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(cipherText) != int(slice.Length) {
		return nil, errorx.New(errorx.ErrCodeCrypto, "cipherText not match")
	}
	hashReceived := xchainClient.HashUsingSha256(cipherText)
	if !bytes.Equal(hashReceived, slice.CipherHash) {
		return nil, errorx.New(errorx.ErrCodeCrypto, "hash not match")
	}

	// decrypt the slice
	decOpt := encryptor.RecoverOptions{
		FileID:  fileID,
		SliceID: slice.ID,
		NodeID:  node.ID,
	}
	return encrypt.Recover(bytes.NewReader(cipherText), &decOpt)
}

// EncAndPush encrypt a slice and push to specified storage node
func EncAndPush(ctx context.Context, copier CommonCopier, encrypt CommonEncryptor,
	plaintext []byte, sliceID, sourceID, fileID string, node *blockchain.Node) (encryptor.EncryptedSlice, error) {

	encOpt := encryptor.EncryptOptions{
		FileID:  fileID,
		SliceID: sliceID,
		NodeID:  node.ID,
	}
	es, err := encrypt.Encrypt(bytes.NewReader(plaintext), &encOpt)
	if err != nil {
		return es, err
	}
	return es, copier.Push(ctx, es.SliceID, sourceID, bytes.NewReader(es.CipherText), node)
}

// ExpandFileSlices expand each slice to specific replica
// 1. find new storage node for the slice
// 2. pull slice from other node and re-encrypt for new node
// 3. push slice to new storage node
// 4. generate challenge material for new storage node
func ExpandFileSlices(ctx context.Context, privkey ecdsa.PrivateKey, cp CommonCopier, enc CommonEncryptor, chain CommonChain,
	challenger CommonChallenger, file blockchain.File, nodesMap map[string]blockchain.Node, replica int,
	healthNodes blockchain.NodeHs, interval int64, l *logrus.Entry) error {

	slices := file.Slices
	ca, pairingConf := challenger.GetChallengeConf()
	oldSliceLen := len(slices)
	sliceNodesMap := GetSliceNodes(slices, nodesMap)

	// record new slices
	var expandSlices []encryptor.EncryptedSlice
	// do capacity expansion for one slice
	// remark: slice cannot concurrent expand, because sindex can be covered
	for sid, snode := range sliceNodesMap {
		if len(snode) >= replica {
			continue
		}
		opt := &copier.ReplicaExpOptions{
			SliceID:       sid,
			SelectedNodes: snode,
			NewReplica:    replica,
			NodesList:     healthNodes,
			PrivateKey:    privkey[:],
			SliceMetas:    slices,
		}
		if ca == types.PairingChallengeAlgorithm {
			opt.PairingConf = pairingConf
		}
		nss, ess, err := cp.ReplicaExpansion(ctx, opt, enc, ca, hex.EncodeToString(file.Owner), file.ID)
		if err != nil {
			l.WithFields(logrus.Fields{
				"slice_id":      sid,
				"old_replica":   replica,
				"new_replica":   replica,
				"expand_length": len(nss),
			}).WithError(err).Error("failed to expansion slice replica")
			if len(nss) == 0 {
				continue
			}
		}
		expandSlices = append(expandSlices, ess...)
		slices = append(slices, nss...)
	}

	// if all expansion failed, return err
	if len(slices) == oldSliceLen {
		l.WithFields(logrus.Fields{
			"slices_len":    len(slices),
			"old_slice_len": oldSliceLen,
		}).Debug("expand slices all failed")
		return errorx.New(errorx.ErrCodeInternal, "expand slices all failed")
	}

	newSlicesJson, _ := json.Marshal(slices)
	file.Slices = slices
	// save merkle challenge material for new slice nodes
	if ca == types.MerkleChallengeAlgorithm {
		if err := AddSlicesNewMerkleChallenge(challenger, file, expandSlices, interval, l); err != nil {
			l.WithFields(logrus.Fields{
				"file_id":       file.ID,
				"new_replica":   replica,
				"expand_slices": newSlicesJson,
			}).WithError(err).Error("failed to add slices merkle challenge material")
			return err
		}
	}
	// generate and push pairing based challenge material for new slice nodes
	if ca == types.PairingChallengeAlgorithm {
		if err := AddSlicesNewPairingChallenge(ctx, pairingConf, cp, expandSlices, file, chain, hex.EncodeToString(file.Owner),
			interval, time.Now().UnixNano(), file.ExpireTime, nil, l); err != nil {
			l.WithFields(logrus.Fields{
				"file_id":       file.ID,
				"new_replica":   replica,
				"expand_slices": newSlicesJson,
			}).WithError(err).Error("failed to add slices pairing based challenge material")
			return err
		}
	}

	// sign slice info
	s, err := json.Marshal(slices)
	if err != nil {
		return errorx.Wrap(err, "failed to marshal slices")
	}
	sig, err := ecdsa.Sign(privkey, xchainClient.HashUsingSha256(s))
	if err != nil {
		return errorx.Wrap(err, "failed to marshal slices")
	}
	// update file slices on blockchain
	opt := blockchain.UpdateFilePSMOptions{
		FileID:    file.ID,
		Owner:     file.Owner,
		Slices:    slices,
		Signature: sig[:],
	}
	err = chain.UpdateFilePublicSliceMeta(&opt)
	if err != nil {
		l.WithFields(logrus.Fields{
			"file_id":     file.ID,
			"new_replica": replica,
		}).WithError(err).Error("failed to update file slice meta")
		return err
	}

	l.WithFields(logrus.Fields{
		"file_id":       file.ID,
		"new_replica":   replica,
		"new_slice_len": len(slices),
		"old_slice_len": oldSliceLen,
	}).Info("success file slices expanded")
	return nil
}
