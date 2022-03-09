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

package engine

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/cjqpker/slidewindow"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var defaultConcurrency uint64 = 10

func verifyReadToken(opt types.ReadOptions) error {
	// check timestamp
	var requestExpiredTime = 5 * time.Minute
	if int64(opt.Timestamp) < (time.Now().UnixNano() - requestExpiredTime.Nanoseconds()) {
		return errorx.New(errorx.ErrCodeParam, "request has expired")
	}

	// verify token
	var msg string
	if len(opt.FileID) > 0 {
		msg = fmt.Sprintf("%s,%d", opt.FileID, opt.Timestamp)
	} else {
		msg = fmt.Sprintf("%s,%s,%s,%d", opt.User, opt.Namespace, opt.FileName, opt.Timestamp)
	}
	msgDigest := hash.HashUsingSha256([]byte(msg))
	if err := verifyUserToken(opt.User, opt.Token, msgDigest); err != nil {
		return errorx.Wrap(err, "failed to verify token")
	}

	return nil
}

// Read download file by pulling slices from storage nodes
// The detailed steps are as follows:
// 1. parameters check
// 2. read file info from the blockchain
// 3. decrypt the file's struct to get slice's order
// 4. download slices from the storage node, if request fails, pull slices from other storage nodes
// 5. slices decryption and combination
// 6. decrypt the combined slices to get the original file
func (e *Engine) Read(ctx context.Context, opt types.ReadOptions) (io.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)

	// check key match
	if err := e.verifyUserID(opt.User); err != nil {
		cancel()
		return nil, err
	}
	// verify token
	if err := verifyReadToken(opt); err != nil {
		cancel()
		return nil, err
	}
	pubkey := ecdsa.PublicKeyFromPrivateKey(e.monitor.challengingMonitor.PrivateKey)
	opt.User = pubkey.String()

	// prepare
	allNodes, err := e.chain.ListNodes()
	if err != nil {
		cancel()
		return nil, errorx.Wrap(err, "failed to get nodes from blockchain")
	}
	// get online nodes
	var nodes blockchain.Nodes
	for _, n := range allNodes {
		if n.Online {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		cancel()
		return nil, errorx.New(errorx.ErrCodeInternal, "empty online nodes")
	}
	nodesMap := common.ToNodesMap(nodes)

	// find file from blockchain
	f, err := getBlockchainFile4Read(e.chain, &opt)
	if err != nil {
		cancel()
		return nil, err
	}
	if opt.User != hex.EncodeToString(f.Owner) {
		cancel()
		return nil, errorx.New(errorx.ErrCodeNotAuthorized, "not authorized")
	}

	// recover structure
	fs, err := e.recoverChainFileStructure(f.Structure, opt.FileID)
	if err != nil {
		cancel()
		return nil, err
	}

	// use sliding window
	sw := slidewindow.SlideWindow{
		Total:       uint64(len(fs)),
		Concurrency: defaultConcurrency,
	}

	sw.Init = func(ctx context.Context, s *slidewindow.Session) error {
		return nil
	}

	slicesPool := makeSlicesPool4Read(f.Slices)
	sw.Task = func(ctx context.Context, s *slidewindow.Session) error {
		slice := fs[int(s.Index())]

		targetPool, ok := slicesPool[slice.SliceID]
		if !ok {
			return errorx.Internal(nil, "bad file structure")
		}
		for _, target := range targetPool {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// pull slice
			node, exist := nodesMap[string(target.NodeID)]
			if !exist || !node.Online {
				logger.WithField("node_id", string(target.NodeID)).Warn("abnormal node")
				continue
			}

			r, err := e.copier.Pull(ctx, target.ID, f.ID, &node)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"slice_id":    target.ID,
					"file_id":     opt.FileID,
					"target_node": string(node.ID),
				}).WithError(err).Warn("failed to pull slice")
				continue
			}
			defer r.Close()

			// read
			cipherText, err := ioutil.ReadAll(r)
			if err != nil {
				logger.WithError(err).Warn("failed to read slice from target node")
				continue
			}
			if len(cipherText) != int(target.Length) {
				logger.WithFields(logrus.Fields{"expected": target.Length, "got": len(cipherText)}).
					Warn("invalid slice length.")
				continue
			}
			hGot := hash.HashUsingSha256(cipherText)
			if !bytes.Equal(hGot, target.CipherHash) {
				logger.WithFields(logrus.Fields{"expected": target.CipherHash, "got": hGot}).
					Warn("invalid slice hash.")
				continue
			}

			// decrypt
			eOpt := encryptor.RecoverOptions{
				FileID:  opt.FileID,
				SliceID: target.ID,
				NodeID:  target.NodeID,
			}
			plainText, err := e.encryptor.Recover(bytes.NewReader(cipherText), &eOpt)
			if err != nil {
				logger.WithError(err).Error("failed to decrypt slice")
				continue
			}

			// trim 0 at the end of file
			if s.Index() != sw.Total-1 {
				s.Set("data", plainText)
			} else {
				s.Set("data", bytes.TrimRight(plainText, string([]byte{0})))
			}

			break
		}

		if _, exist := s.Get("data"); !exist {
			return errorx.New(errorx.ErrCodeNotFound, "failed to pull slice %s", slice.SliceID)
		}

		return nil
	}

	reader, writer := io.Pipe()
	sw.Done = func(ctx context.Context, s *slidewindow.Session) error {
		data, exist := s.Get("data")
		if !exist {
			return errorx.New(errorx.ErrCodeNotFound, "failed to find data")
		}

		if _, err := writer.Write(data.([]byte)); err != nil {
			return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to write")
		}

		// exit on success
		if s.Index() == uint64(len(fs)-1) {
			writer.Close()
		}

		return nil
	}

	go func() {
		defer cancel()
		if err := sw.Start(ctx); err != nil {
			writer.CloseWithError(err)
		}
	}()

	// decrypt recovered file
	plain, err := e.encryptor.Recover(reader, &encryptor.RecoverOptions{FileID: opt.FileID})
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeCrypto, "failed to recover original file")
	}
	return ioutil.NopCloser(bytes.NewReader(plain)), nil
}

// getBlockchainFile4Read query file details by fileID or fileName from blockchain
func getBlockchainFile4Read(chain Blockchain, opt *types.ReadOptions) (
	blockchain.File, error) {
	var err error
	var f blockchain.File
	if len(opt.FileID) > 0 {
		f, err = chain.GetFileByID(opt.FileID)
	} else {
		pubkey, _ := hex.DecodeString(opt.User)
		f, err = chain.GetFileByName(pubkey, opt.Namespace, opt.FileName)
	}
	if err != nil {
		return f, errorx.Wrap(err, "failed to read file from blockchain")
	}

	return f, nil
}

// makeSlicesPool4Read get the list of storage nodes of slices by blockchain.PublicSliceMeta.
// return map, key is sliceID, value is storage nodeID lists
func makeSlicesPool4Read(srs []blockchain.PublicSliceMeta) map[string][]blockchain.PublicSliceMeta {
	slicesPool := make(map[string][]blockchain.PublicSliceMeta)
	for _, s := range srs {
		var ss []blockchain.PublicSliceMeta
		if v, exist := slicesPool[s.ID]; exist {
			ss = v
		}
		ss = append(ss, s)
		slicesPool[s.ID] = ss
	}

	return slicesPool
}
