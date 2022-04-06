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

package random

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	etype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/http"
)

var (
	logger = logrus.WithField("module", "random-copier")
)

// RandomCopier selects Storage Nodes randomly from healthy candidates.
//  You can call Push() to push slices onto Storage Node, and Pull() to pull slices from Storage Node.
//  If you want more Storage Nodes, you can call ReplicaExpansion(),
//  and it pulls slices from original nodes and decrypts and re-encrypts those slices,
//  then push them onto new Storage Nodes.
type RandomCopier struct {
	privateKey ecdsa.PrivateKey
}

func New(privkey ecdsa.PrivateKey) *RandomCopier {
	c := &RandomCopier{
		privateKey: privkey,
	}
	logger.Info("copier initialization")
	return c
}

type nodeWeight struct {
	nodeID []byte
	start  uint64
	end    uint64
}

// Select selects nodes for a slice randomly from healthy candidates
func (m *RandomCopier) Select(slice slicer.Slice, nodes blockchain.NodeHs, opt *copier.SelectOptions) (
	copier.LocatedSlice, error) {
	if opt.Excludes == nil {
		opt.Excludes = make(map[string]struct{})
	}

	if len(nodes) == 0 {
		return copier.LocatedSlice{}, errorx.New(errorx.ErrCodeInternal, "empty nodes array")
	}

	targetReplica := int(opt.Replica)
	if targetReplica == 0 {
		return copier.LocatedSlice{}, errorx.New(errorx.ErrCodeInternal, "empty replica")
	}

	rand.Seed(time.Now().UnixNano())

	// green nodes first
	nodesList := getSliceOptimalNodes(nodes, targetReplica)
	used := make(map[int]struct{})
	var selected blockchain.Nodes
	for {
		if len(selected) == targetReplica {
			break
		}
		if len(selected) == len(nodesList) {
			break
		}

		index := rand.Int() % len(nodesList)
		if _, existed := used[index]; !existed {
			used[index] = struct{}{}
			selected = append(selected, nodesList[index])
		}
	}
	logger.WithFields(logrus.Fields{
		"slice_id":       slice.ID,
		"online_nodes":   len(nodes),
		"selected_nodes": len(selected),
	}).Debug("selection done")

	ls := copier.LocatedSlice{
		Slice: slice,
		Nodes: selected,
	}
	return ls, nil
}

// Push pushes slices onto Storage Node
func (m *RandomCopier) Push(ctx context.Context, id, sourceID string, r io.Reader, node *blockchain.Node) error {
	// Todo add signature when pushing slices into storage nodes
	url := fmt.Sprintf("http://%s/v1/slice/push?slice_id=%s&source_id=%s", node.Address, id, sourceID)

	var resp etype.PushResponse
	if err := http.PostResponse(ctx, url, r, &resp); err != nil {
		return errorx.Wrap(err, "failed to do post")
	}

	return nil
}

func (m *RandomCopier) Pull(ctx context.Context, id, fileID string, node *blockchain.Node) (io.ReadCloser, error) {
	// Add signature when pulling slices from storage nodes
	timestamp := time.Now().UnixNano()
	msg := fmt.Sprintf("%s,%s,%d", id, fileID, timestamp)
	sig, err := ecdsa.Sign(m.privateKey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return nil, errorx.Wrap(err, "failed to sign file pull")
	}
	url := fmt.Sprintf("http://%s/v1/slice/pull?slice_id=%s&file_id=%s&timestamp=%d&signature=%s",
		node.Address, id, fileID, timestamp, sig.String())

	r, err := http.Get(ctx, url)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to do get")
	}

	return r, nil
}

// ReplicaExpansion slice performs Replica-Expand, that is to
//  pull slices from original nodes and decrypt and re-encrypt those slices,
//  then push them onto new Storage Nodes.
func (m *RandomCopier) ReplicaExpansion(ctx context.Context, opt *copier.ReplicaExpOptions,
	enc common.CommonEncryptor, challengeAlgorithm, sourceID, fileID string) (
	nSlice []blockchain.PublicSliceMeta, eSlices []encryptor.EncryptedSlice, err error) {
	// 1 get more proper storage nodes
	nNodes, err := getOptionalNode(opt.SelectedNodes, opt.NodesList)
	if err != nil {
		return nSlice, eSlices, errorx.NewCode(err, errorx.ErrCodeInternal, "no optional node to expand replica")
	}

	// 2 pull slices from original nodes and decrypt those slices
	plainText := m.pullSlice(ctx, opt.SelectedNodes, opt.SliceMetas, opt.SliceID, fileID, enc)
	if len(plainText) == 0 {
		return nSlice, eSlices, errorx.New(errorx.ErrCodeInternal, "slice pull from all healthy nodes error")
	}

	// 3 re-encrypt those slices and push them onto new storage nodes
	sliceExpandNum := opt.NewReplica - len(opt.SelectedNodes)
	for i := 0; i < sliceExpandNum; i++ {
		pushRes := false
		for _, n := range nNodes {
			es, err := common.EncAndPush(ctx, m, enc, plainText, opt.SliceID, sourceID, fileID, &n)

			if err != nil {
				logger.WithFields(logrus.Fields{
					"slice_id":    es.SliceID,
					"target_node": string(n.ID),
				}).Debug("replica slice failed to enc and re-pushed")
				continue
			}
			newSp := blockchain.PublicSliceMeta{
				ID:         es.SliceID,
				CipherHash: es.CipherHash,
				Length:     es.Length,
				NodeID:     es.NodeID,
			}
			if challengeAlgorithm == types.PairingChallengeAlgorithm {
				// 4 gets SliceIdx
				sindex := getNodeSliceIdx(opt.SliceMetas, string(es.NodeID))
				newSp.SliceIdx = sindex
			}

			nSlice = append(nSlice, newSp)
			eSlices = append(eSlices, es)
			opt.SelectedNodes = append(opt.SelectedNodes, n)
			opt.SliceMetas = append(opt.SliceMetas, newSp)
			pushRes = true
			break
		}
		if !pushRes {
			return nSlice, eSlices, errorx.Wrap(err, "replica slice re-pushed error")
		}
		if i+1 < sliceExpandNum {
			nNodes, err = getOptionalNode(opt.SelectedNodes, opt.NodesList)
			if err != nil {
				return nSlice, eSlices, errorx.Wrap(err, "no optional node to expand replica")
			}
		}
	}
	return nSlice, eSlices, nil
}

// getOptionalNode gets more proper Storage Nodes
func getOptionalNode(selectedNodes blockchain.Nodes, allNodes blockchain.NodeHs) (expandNodes blockchain.Nodes, err error) {
	var selectedS []string
	for _, n := range selectedNodes {
		selectedS = append(selectedS, string(n.ID))
	}
	return common.FindNewNodes(allNodes, selectedS)
}

// pullSlice pull slices from selected Storage Nodes
func (m *RandomCopier) pullSlice(ctx context.Context, selectedNodes blockchain.Nodes,
	sliceMetas []blockchain.PublicSliceMeta, sliceID, fileID string, enc common.CommonEncryptor) (plainText []byte) {
	for _, n := range selectedNodes {
		sm := getSliceMetaByID(sliceMetas, sliceID, string(n.ID))
		plainText, err := common.PullAndDec(ctx, m, enc, sm, &n, fileID)

		if err != nil {
			logger.WithError(err).Error("failed to decrypt slice")
			continue
		}
		return plainText
	}
	return plainText
}

func getSliceMetaByID(sliceMetas []blockchain.PublicSliceMeta, sliceID, nodeID string) (metas blockchain.PublicSliceMeta) {
	for _, slice := range sliceMetas {
		if slice.ID == sliceID && string(slice.NodeID) == nodeID {
			return slice
		}
	}
	return metas
}

func getNodeSliceIdx(sliceMetas []blockchain.PublicSliceMeta, nodeID string) int {
	sliceIdxMap := make(map[string]int)
	for _, s := range sliceMetas {
		// denote slice index for each node (for pairing based challenge)
		if string(s.NodeID) == nodeID {
			_, ok := sliceIdxMap[nodeID]
			if !ok || (ok && s.SliceIdx > sliceIdxMap[nodeID]) {
				sliceIdxMap[nodeID] = s.SliceIdx
			}
		}
	}
	if len(sliceIdxMap) == 0 {
		return 1
	}
	return sliceIdxMap[nodeID] + 1
}
