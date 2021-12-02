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
	"io"
	"io/ioutil"
	"math/big"

	fl_crypto "github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// MigrateChain defines several contract/chaincode methods related to file migration
type MigrateChain interface {
	UpdateFilePublicSliceMeta(ctx context.Context, opt *blockchain.UpdateFilePSMOptions) error
}

// MigrateCopier defines slice copier when migrating a file
type MigrateCopier interface {
	Push(ctx context.Context, id, sourceId string, r io.Reader, node *blockchain.Node) error
	Pull(ctx context.Context, id, fileId string, node *blockchain.Node) (io.ReadCloser, error)
	ReplicaExpansion(ctx context.Context, opt *copier.ReplicaExpOptions, enc MigrateEncryptor, ca, sourceId, fileId string) (
		[]blockchain.PublicSliceMeta, []encryptor.EncryptedSlice, error)
}

// MigrateEncryptor defines encryptor for file encryption and decryption when migrating a file
type MigrateEncryptor interface {
	Encrypt(ctx context.Context, r io.Reader, opt *encryptor.EncryptOptions) (
		encryptor.EncryptedSlice, error)
	Recover(ctx context.Context, r io.Reader, opt *encryptor.RecoverOptions) (
		[]byte, error)
}

// PullAndDec pull a slice from healthy node and decrypt
func PullAndDec(ctx context.Context, copier MigrateCopier, encrypt MigrateEncryptor,
	slice blockchain.PublicSliceMeta, node *blockchain.Node, fileId string) ([]byte, error) {

	r, err := copier.Pull(ctx, slice.ID, fileId, node)
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
		SliceID: slice.ID,
		NodeID:  node.ID,
	}
	return encrypt.Recover(ctx, bytes.NewReader(cipherText), &decOpt)
}

// EncAndPush encrypt a slice and push to specified storage node
func EncAndPush(ctx context.Context, copier MigrateCopier, encrypt MigrateEncryptor,
	plaintext []byte, sliceID, sourceId string, node *blockchain.Node) (encryptor.EncryptedSlice, error) {

	encOpt := encryptor.EncryptOptions{
		SliceID: sliceID,
		NodeID:  node.ID,
	}
	es, err := encrypt.Encrypt(ctx, bytes.NewReader(plaintext), &encOpt)
	if err != nil {
		return es, err
	}
	return es, copier.Push(ctx, es.SliceID, sourceId, bytes.NewReader(es.CipherText), node)
}

// GetSigmaISliceIdx get random challenge material for a slice
func GetSigmaISliceIdx(ciphertext []byte, sliceIdx int, pdp types.PDP) (sigmaI []byte, err error) {
	idx := big.NewInt(int64(sliceIdx))
	xchainClient := new(fl_crypto.XchainCryptoClient)
	sigmaI, err = xchainClient.CalculatePDPSigmaI(ciphertext, idx.Bytes(), pdp.RandV, pdp.RandU, pdp.PdpPrivkey)
	if err != nil {
		return sigmaI, errorx.Wrap(err, "CalculatePDPSigmaI failed")
	}
	return sigmaI, nil
}

// ExpandFileSlices expand each slice to specific replica
func ExpandFileSlices(ctx context.Context, privkey ecdsa.PrivateKey, cp MigrateCopier, enc MigrateEncryptor, chain MigrateChain,
	challegener MerkleChallenger, file blockchain.File, nodesMap map[string]blockchain.Node, replica int,
	healthNodes blockchain.NodeHs, interval int64, l *logrus.Entry) error {

	slices := file.Slices
	ca, pdp := challegener.GetChallengeConf()
	oldSliceLen := len(slices)
	slice := GetSliceNodes(slices, nodesMap)
	var expandSlices []encryptor.EncryptedSlice
	// do capacity expansion for one slice
	// remark: slice cannot concurrent expand, because sindex can be covered
	for sid, snode := range slice {
		if len(snode) >= replica {
			continue
		}
		opt := &copier.ReplicaExpOptions{
			SliceId:       sid,
			SelectedNodes: snode,
			NewReplica:    replica,
			NodesList:     healthNodes,
			PrivateKey:    privkey[:],
			SliceMetas:    slices,
		}
		if ca == types.PDPChallengAlgorithm {
			opt.PDP = pdp
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
		if ca == types.MerkleChallengAlgorithm {
			expandSlices = append(expandSlices, ess...)
		}
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

	// sign slice info
	s, err := json.Marshal(slices)
	if err != nil {
		return errorx.Wrap(err, "failed to marshal slices")
	}
	sig, err := ecdsa.Sign(privkey, xchainClient.HashUsingSha256(s))
	if err != nil {
		return errorx.Wrap(err, "failed to marshal slices")
	}
	// save merkle expand slice
	if ca == types.MerkleChallengAlgorithm {
		if err := AddSlicesNewMerkleChallenge(ctx, challegener, cp, file, expandSlices, interval, l); err != nil {
			l.WithFields(logrus.Fields{
				"file_id":       file.ID,
				"new_replica":   replica,
				"expand_slices": slices,
			}).WithError(err).Error("failed to add slices merkle challenge")
			return err
		}
	}
	opt := blockchain.UpdateFilePSMOptions{
		FileID:    file.ID,
		Owner:     file.Owner,
		Slices:    slices,
		Signature: sig[:],
	}
	err = chain.UpdateFilePublicSliceMeta(ctx, &opt)
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
