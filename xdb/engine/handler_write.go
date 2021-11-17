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
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/hash"
)

const (
	defaultLocatorAmount     = 4
	defaultEncryptorAmount   = 4
	defaultDistributorAmount = 4

	defaultRetryTime = 5 // seconds
)

type finishWritenSlice struct {
	eSlice encryptor.EncryptedSlice
}

// Write upload a file and push file slices to storage nodes
func (e *Engine) Write(ctx context.Context, opt types.WriteOptions,
	r io.Reader) (resp types.WriteResponse, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var errOccurred error

	// check key match
	if err := e.verifyUserID(opt.User); err != nil {
		return resp, err
	}
	// verify token
	msg := fmt.Sprintf("%s:%s:%s", opt.User, opt.Namespace, opt.FileName)
	if err := verifyUserToken(opt.User, opt.Token, hash.Hash([]byte(msg))); err != nil {
		return resp, errorx.Wrap(err, "failed to verify token")
	}

	// duplicate check
	pubkey, _ := hex.DecodeString(opt.User)
	if _, err := e.chain.GetFileByName(ctx, pubkey, opt.Namespace, opt.FileName); err == nil {
		return resp, errorx.New(errorx.ErrCodeAlreadyExists, "duplicated name")
	} else if !errorx.Is(err, errorx.ErrCodeNotFound) {
		return resp, errorx.Wrap(err, "failed to read blockchain")
	}
	ns, err := e.chain.GetNsByName(ctx, pubkey, opt.Namespace)
	if err != nil {
		return resp, errorx.Wrap(err, "failed to get ns from blockchain")
	}
	fileID, err := uuid.NewRandom()
	if err != nil {
		return resp, errorx.Internal(err, "failed to get uuid")
	}
	nodes, err := common.GetHealthNodes(ctx, e.chain)
	if err != nil {
		return resp, err
	}
	if len(nodes) < ns.Replica {
		return resp, errorx.Internal(err, "available healthy nodes smaller than replica")
	}
	nodesMap := common.ToNodeHsMap(nodes)

	logger.WithFields(logrus.Fields{
		"file_id":       fileID.String(),
		"file_name":     opt.FileName,
		"namespace":     opt.Namespace,
		"ns_total_file": ns.FileTotalNum,
		"exp_time":      time.Unix(0, opt.ExpireTime).Format("2006-01-02 15:04:05"),
	}).Info("write file")

	// encrypt file first
	cipher, err := e.encryptor.Encrypt(context.TODO(), r, &encryptor.EncryptOptions{})
	if err != nil {
		logger.WithError(err).Error("file encryption failed")
		return resp, errorx.NewCode(err, errorx.ErrCodeCrypto, "file encryption failed")
	}
	r = bytes.NewReader(cipher.CipherText)
	originalLen := len(cipher.CipherText) - 16

	slicesNum := math.Ceil(float64(len(cipher.CipherText)) / float64(e.slicer.GetBlockSize()))
	fileStrucSize := calculateFileMaxStructSize(int(slicesNum), ns.Replica)
	if (ns.FilesStruSize + fileStrucSize) >= blockchain.ContractMessageMaxSize {
		logger.WithFields(logrus.Fields{
			"file_id":        fileID.String(),
			"ns_files_size":  ns.FilesStruSize,
			"file_strc_size": fileStrucSize,
		}).Warnf("files total struct size of ns more than maximum")
		return resp, errorx.New(errorx.ErrCodeParam, "files total struct size of ns more than maximum")
	}
	// Slice. sliceQueue will be closed when slicer get EOF
	sliceOpts := slicer.SliceOptions{}
	sliceQueue := e.slicer.Slice(ctx, r, &sliceOpts, func(err error) {
		logger.WithError(err).Error("slicing stopped")
		cancel()
	})

	// Find nodes for slices.
	// Both sliceMetaQueue and locatedSliceQueue will be closed when sliceQueue is closed
	sliceMetaQueue := make(chan slicer.SliceMeta, 10)
	locatedSliceQueue := make(chan copier.LocatedSlice, defaultLocatorAmount*2)
	go e.locateRoutine(ctx, ns.Replica, nodes, sliceQueue, locatedSliceQueue, sliceMetaQueue, func(err error) {
		logger.WithError(err).Error("slice location stopped")
		errOccurred = err
		cancel()
	})
	var sliceMetas []slicer.SliceMeta
	go func() {
		for s := range sliceMetaQueue {
			sliceMetas = append(sliceMetas, s)
		}
	}()

	// Encrypt. encryptedSliceQueue will be closed when locatedSliceQueue is closed
	encryptedSliceQueue := make(chan encryptor.EncryptedSlice, 10)
	go e.encryptRoutine(ctx, locatedSliceQueue, encryptedSliceQueue, func(err error) {
		logger.WithError(err).Error("slice encryption stopped")
		errOccurred = err
		cancel()
	})

	// get chanller
	ca, pdp := e.challenger.GetChallengeConf()

	// Setup challenging materials && Distribute
	// both finishedQueue and failedQueue will be closed when encryptedSliceQueue is closed
	finishedQueue := make(chan finishWritenSlice, 10)
	failedQueue := make(chan encryptor.EncryptedSlice, 10)
	go e.distributeRoutine(ctx, nodesMap, encryptedSliceQueue, finishedQueue, failedQueue, opt.User)
	var finishedEncSlices []encryptor.EncryptedSlice
	for m := range finishedQueue {
		finishedEncSlices = append(finishedEncSlices, m.eSlice)
	}

	// retry push
	var failedSlices []encryptor.EncryptedSlice
	for f := range failedQueue {
		failedSlices = append(failedSlices, f)
	}
	finishedQueue2 := make(chan finishWritenSlice, 10)
	failedQueue2 := make(chan encryptor.EncryptedSlice, 10)
	e.retryRoutine(ctx, failedSlices, finishedQueue2, failedQueue2, nodesMap, opt.User)
	for m := range finishedQueue2 {
		finishedEncSlices = append(finishedEncSlices, m.eSlice)
	}
	var failedTwice []encryptor.EncryptedSlice
	for m := range failedQueue2 {
		failedTwice = append(failedTwice, m)
	}

	// if push fails again, push to another node
	finishedQueue3 := e.pushToOtherNode(ctx, opt.User,
		failedTwice, finishedEncSlices, nodes, func(err error) {
			logger.WithError(err).Error("pushToOtherNode failed")
			errOccurred = err
			cancel()
		})

	// check writing error
	if errOccurred != nil {
		return resp, errorx.Wrap(errOccurred, "error occurred in writing")
	}

	// all pushed slice info
	finishedEncSlices = append(finishedEncSlices, finishedQueue3...)
	if ca == types.MerkleChallengAlgorithm {
		if err := e.generateAndSaveMerkle(ctx, finishedEncSlices, fileID.String(), opt.ExpireTime); err != nil {
			return resp, err
		}
	}

	// Write meta info to blockchain
	chainFile, err := e.packChainFile(fileID.String(), ca, opt, sliceMetas, originalLen, finishedEncSlices, pdp)
	if err != nil {
		return resp, errorx.Wrap(err, "failed to pack chain file")
	}

	// sign file info
	s, err := json.Marshal(chainFile)
	if err != nil {
		return resp, errorx.Wrap(err, "failed to marshal File")
	}
	sig, err := ecdsa.Sign(e.monitor.challengingMonitor.PrivateKey, hash.Hash(s))
	if err != nil {
		return resp, errorx.Wrap(err, "failed to sign File")
	}
	publishFileOpt := blockchain.PublishFileOptions{
		File:      chainFile,
		Signature: sig[:],
	}

	if err := e.chain.PublishFile(ctx, &publishFileOpt); err != nil {
		return resp, errorx.Wrap(err, "failed to write file to blockchain")
	}

	logger.WithField("file_id", fileID.String()).Debug("file uploaded")
	resp.FileID = fileID.String()
	return resp, nil
}

// locateRoutine block current routine
func (e *Engine) locateRoutine(ctx context.Context, replica int, nodes blockchain.NodeHs, sliceQueue <-chan slicer.Slice,
	locatedQueue chan<- copier.LocatedSlice, metaQueue chan<- slicer.SliceMeta, onErr func(err error)) {
	wg := sync.WaitGroup{}

	taskQueue := make(chan slicer.Slice, 10)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(taskQueue)

		for {
			s, ok := <-sliceQueue
			if !ok {
				return
			}

			metaQueue <- s.SliceMeta
			taskQueue <- s
		}
	}()

	wg.Add(defaultLocatorAmount)
	for i := 0; i < defaultLocatorAmount; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				slice, ok := <-taskQueue
				if !ok {
					return
				}

				locatedSlice, err := e.copier.Select(slice, nodes, &copier.SelectOptions{Replica: uint32(replica)})
				if err != nil {
					onErr(errorx.Wrap(err, "failed to select nodes for slice %x", slice.Hash))
					return
				}

				locatedQueue <- locatedSlice
			}
		}()
	}

	wg.Wait()
	close(metaQueue)
	close(locatedQueue)
}

func (e *Engine) encryptRoutine(ctx context.Context, locatedQueue <-chan copier.LocatedSlice,
	encryptedQueue chan<- encryptor.EncryptedSlice, onErr func(err error)) {
	wg := sync.WaitGroup{}

	type SliceNodePair struct {
		Slice slicer.Slice
		Nodes blockchain.Node
	}
	taskQueue := make(chan SliceNodePair, defaultEncryptorAmount)
	wg.Add(1)
	go func() {
		defer close(taskQueue)
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			sliceNodesPair, ok := <-locatedQueue
			if !ok {
				return
			}
			for _, node := range sliceNodesPair.Nodes {
				taskQueue <- SliceNodePair{sliceNodesPair.Slice, node}
			}
		}
	}()

	wg.Add(defaultEncryptorAmount)
	for i := 0; i < defaultEncryptorAmount; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				lSlice, ok := <-taskQueue
				if !ok {
					return
				}

				eopt := encryptor.EncryptOptions{
					SliceID: lSlice.Slice.ID,
					NodeID:  lSlice.Nodes.ID,
				}
				es, err := e.encryptor.Encrypt(ctx, bytes.NewReader(lSlice.Slice.Data), &eopt)
				if err != nil {
					onErr(errorx.Wrap(err, "failed to encrypt slice"))
					return
				}
				logger.WithField("slice_id", es.EncryptedSliceMeta.SliceID).Info("slice encrypted")
				encryptedQueue <- es
			}
		}()
	}

	wg.Wait()
	close(encryptedQueue)
}

// distributeRoutine push slices to storage nodes
func (e *Engine) distributeRoutine(ctx context.Context, nodes map[string]blockchain.Node,
	encryptedQueue <-chan encryptor.EncryptedSlice, finishedQueue chan<- finishWritenSlice,
	failedQueue chan<- encryptor.EncryptedSlice, owner string) {
	wg := sync.WaitGroup{}

	wg.Add(defaultDistributorAmount)
	for i := 0; i < defaultDistributorAmount; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				es, ok := <-encryptedQueue
				if !ok {
					return
				}

				// push slice
				node := nodes[string(es.NodeID)]
				dataReader := bytes.NewReader(es.CipherText)
				if err := e.copier.Push(ctx, es.SliceID, owner, dataReader, &node); err != nil {
					logger.WithError(err).Errorf("failed to push to: %v, slice: %s", node, es.SliceID)
					failedQueue <- es
					continue
				}
				logger.WithFields(logrus.Fields{
					"target_node": node.Name,
					"address":     node.Address,
				}).Debug("slice pushed")

				finishedQueue <- finishWritenSlice{
					eSlice: es,
				}
			}
		}()
	}

	wg.Wait()
	close(finishedQueue)
	close(failedQueue)
}

// retryRoutine re-push failed slice
func (e *Engine) retryRoutine(ctx context.Context, failedSlices []encryptor.EncryptedSlice,
	finishedQueue chan<- finishWritenSlice, failedQueue chan<- encryptor.EncryptedSlice,
	nodes map[string]blockchain.Node, owner string) {

	wg := sync.WaitGroup{}
	wg.Add(len(failedSlices))
	for i := 0; i < len(failedSlices); i++ {
		go func(i int) {
			defer wg.Done()
			es := failedSlices[i]
			node := nodes[string(es.NodeID)]
			dataReader := bytes.NewReader(es.CipherText)
			logger.Infof("retry push %s to %s", es.SliceID, es.NodeID)

			endTime := time.Now().Unix() + defaultRetryTime
			// retry once every second
			ticker := time.NewTicker(time.Duration(1) * time.Second)
			for time.Now().Unix() < endTime {
				select {
				case <-ticker.C:
					if err := e.copier.Push(ctx, es.SliceID, owner, dataReader, &node); err == nil {
						logger.WithFields(logrus.Fields{
							"target_node": node.Name,
							"address":     node.Address,
						}).Debug("slice pushed")

						finishedQueue <- finishWritenSlice{
							eSlice: es,
						}
						return
					} else {
						logger.WithError(err).Errorf("failed to push %s to %s", es.SliceID, es.NodeID)
					}
				}
			}
			logger.WithFields(logrus.Fields{
				"slice_id":    es.SliceID,
				"target_node": string(es.NodeID),
			}).Error("retryRoutine timeout")
			failedQueue <- es
		}(i)
	}

	wg.Wait()
	close(finishedQueue)
	close(failedQueue)
}

// pushToOtherNode re-push failed slice to another node
func (e *Engine) pushToOtherNode(ctx context.Context, owner string, failedSlices []encryptor.EncryptedSlice,
	finishedEncSlices []encryptor.EncryptedSlice, nodes blockchain.NodeHs, onErr func(error)) []encryptor.EncryptedSlice {

	var finishedSlices []encryptor.EncryptedSlice
	alreadySelected := make(map[string][]string)
	for _, slice := range finishedEncSlices {
		alreadySelected[slice.SliceID] = append(alreadySelected[slice.SliceID], string(slice.NodeID))
	}

	for _, slice := range failedSlices {
		alreadySelected[slice.SliceID] = append(alreadySelected[slice.SliceID], string(slice.NodeID))

		// select available nodes for failed slice
		nodeList, err := common.FindNewNodes(nodes, alreadySelected[slice.SliceID])
		if err != nil {
			logger.WithError(err).Errorf("findNewNodes failed for slice: %s", slice.SliceID)
			onErr(errorx.Wrap(err, "failed to findNewNodes"))
			return nil
		}

		done := false
		for _, node := range nodeList {
			logger.Infof("re-push %s to %s", slice.SliceID, node.ID)
			alreadySelected[slice.SliceID] = append(alreadySelected[slice.SliceID], string(node.ID))

			// decrypt
			ropt := encryptor.RecoverOptions{
				SliceID: slice.SliceID,
				NodeID:  slice.NodeID,
			}
			plain, err := e.encryptor.Recover(ctx, bytes.NewReader(slice.CipherText), &ropt)
			if err != nil {
				onErr(errorx.Wrap(err, "failed to decrypt slice"))
				return nil
			}

			// re-encrypt
			eopt := encryptor.EncryptOptions{
				SliceID: slice.SliceID,
				NodeID:  node.ID,
			}
			es, err := e.encryptor.Encrypt(ctx, bytes.NewReader(plain), &eopt)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"slice_id":    slice.SliceID,
					"target_node": string(node.ID),
				}).WithError(err).Error("failed to re-encrypt")
				onErr(errorx.Wrap(err, "failed to re-encrypt slice"))
				return nil
			}

			// push to new node
			if err := e.copier.Push(ctx, es.SliceID, owner, bytes.NewReader(es.CipherText), &node); err == nil {
				logger.WithFields(logrus.Fields{
					"slice_id":    es.SliceID,
					"target_node": string(node.ID),
				}).Debug("slice re-pushed")

				finishedSlices = append(finishedSlices, es)
				done = true
				break
			} else {
				logger.WithFields(logrus.Fields{
					"slice_id":    es.SliceID,
					"target_node": string(node.ID),
				}).WithError(err).Error("re-push failed")
			}
		}

		if !done {
			onErr(errorx.New(errorx.ErrCodeInternal, "failed to pushToOtherNode"))
			return nil
		}
	}

	return finishedSlices
}

// generateAndSaveMerkle file owner generates and saves merkle challenges to local
func (e *Engine) generateAndSaveMerkle(ctx context.Context, finishedEncSlices []encryptor.EncryptedSlice,
	fileID string, expireTime int64) error {
	var isSaveErr error
	wg := sync.WaitGroup{}
	for _, es := range finishedEncSlices {
		wg.Add(1)
		merkleMaterialQueue := make(chan ctype.Material, 5)
		go func(wg *sync.WaitGroup, es encryptor.EncryptedSlice) {
			defer wg.Done()
			e.getMerkleChallengerRange(fileID, es, expireTime, merkleMaterialQueue)
		}(&wg, es)
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				cm, ok := <-merkleMaterialQueue
				if !ok {
					return
				}
				sliceMaterial := ctype.Material{
					NodeID:  cm.NodeID,
					FileID:  cm.FileID,
					SliceID: cm.SliceID,
					Ranges:  cm.Ranges,
				}
				if len(sliceMaterial.Ranges) == 0 {
					logger.WithFields(logrus.Fields{
						"file_id":     sliceMaterial.FileID,
						"slice_id":    sliceMaterial.SliceID,
						"target_node": sliceMaterial.NodeID,
					}).Warnf("empty hashes")
					isSaveErr = errorx.New(errorx.ErrCodeInternal, "failed get challenging merkle materials, ranges empty")
					return
				}
				csMaterial := []ctype.Material{sliceMaterial}
				if err := common.SaveMerkleChallenger(ctx, e.challenger, csMaterial); err != nil {
					isSaveErr = errorx.Wrap(err, "failed save challenging merkle materials")
				}
			}
		}(&wg)
	}
	wg.Wait()
	if isSaveErr != nil {
		return isSaveErr
	}
	return nil
}
