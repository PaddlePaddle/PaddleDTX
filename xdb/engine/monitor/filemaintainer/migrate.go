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

package filemaintainer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var l = logger.WithField("runner", "file migrate loop")

// migrate checks storage-nodes health conditions and migrate slices from bad nodes to healthy nodes
// The health of slices is determined by the number of slices's replicas and
// the health of storage nodes where slices stored,
// if the number of replicas is not enough, expand the slice replicas firstly during slice migration
func (m *FileMaintainer) migrate(ctx context.Context) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(m.localNode.PrivateKey)

	defer l.Info("file migrate stopped")

	ticker := time.NewTicker(m.fileMigrateInterval)
	interval := m.challengerInterval
	challengeAlgorithm, pairingConf := m.challenger.GetChallengeConf()
	defer ticker.Stop()

	m.doneMigrateC = make(chan struct{})
	defer close(m.doneMigrateC)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// 1. find all namespaces
		listNsOpt := blockchain.ListNsOptions{
			Owner:   pubkey[:],
			TimeEnd: time.Now().UnixNano(),
		}
		nsList, err := m.blockchain.ListFileNs(&listNsOpt)
		if err != nil {
			l.WithError(err).Error("failed to find ns list")
			continue
		}
		if len(nsList) == 0 {
			l.Info("no namespace found")
			continue
		}

		// 2. find all healthy nodes
		healthNodes, err := common.GetHealthNodes(m.blockchain)
		if err != nil {
			l.WithError(err).Error("failed to find healthy nodes")
			continue
		}
		if len(healthNodes) == 0 {
			l.WithError(err).Warn("empty healthy nodes")
			continue
		}
		healthNodesMap := make(map[string]blockchain.NodeH)
		var greenNodes blockchain.NodeHs

		for _, node := range healthNodes {
			healthNodesMap[string(node.Node.ID)] = node
			if node.Health == blockchain.NodeHealthGood {
				greenNodes = append(greenNodes, node)
			}
		}

		wg := sync.WaitGroup{}
		wg.Add(len(nsList))
		for _, ns := range nsList {
			go func(ns blockchain.Namespace) {
				defer wg.Done()
				// 3. find all files
				listFileOpt := blockchain.ListFileOptions{
					Owner:       pubkey[:],
					Namespace:   ns.Name,
					TimeEnd:     time.Now().UnixNano(),
					CurrentTime: time.Now().UnixNano(),
				}
				files, err := m.blockchain.ListFiles(&listFileOpt)
				if err != nil {
					l.WithError(err).Error("failed to find file list")
					return
				}
				l.WithField("namespace", ns.Name).Infof("%d files found", len(files))

				// 4. find unhealthy files
				wgf := sync.WaitGroup{}
				wgf.Add(len(files))
				for _, file := range files {
					select {
					case <-ctx.Done():
						return
					default:
					}
					go func(file blockchain.File) {
						defer wgf.Done()
						health, err := common.GetFileHealth(ctx, m.blockchain, file, ns.Replica)
						if err != nil {
							l.WithField("file_id", file.ID).WithError(err).Error("failed to get file health")
							return
						}
						if health == blockchain.NodeHealthGood {
							return
						}

						sliceNum := map[string]struct{}{}
						for _, slice := range file.Slices {
							if _, exist := sliceNum[slice.ID]; !exist {
								sliceNum[slice.ID] = struct{}{}
							}
						}

						// 5.1 check replica satisfaction
						if len(file.Slices) < ns.Replica*len(sliceNum) {
							nodesMap := common.ToNodeHsMap(healthNodes)
							if err := common.ExpandFileSlices(ctx, m.localNode.PrivateKey, m.copier, m.encryptor, m.blockchain,
								m.challenger, file, nodesMap, ns.Replica, healthNodes, interval, l); err != nil {
								l.WithField("file_id", file.ID).WithError(err).Error("failed to migrate file")
								return
							}
							l.WithField("file_id", file.ID).Info("file migrated success expand")
							ef, err := m.blockchain.GetFileByID(file.ID)
							if err != nil {
								l.WithField("file_id", file.ID).Info("failed to get file after expansion")
							} else {
								file.Slices = ef.Slices
							}
						}

						// find storage nodeIDs for each slice
						selectedNodes := make(map[string][]string)
						for _, slice := range file.Slices {
							selectedNodes[slice.ID] = append(selectedNodes[slice.ID], string(slice.NodeID))
						}

						// 5.2 migrate red nodes
						fileUpdated := false
						newSlices := file.Slices
						var yellowNodeSlices []blockchain.PublicSliceMeta
						var migrateEncSlices []encryptor.EncryptedSlice
						var mSlice encryptor.EncryptedSlice
						for _, slice := range file.Slices {
							nodeSliceMap := nodeSliceMap(newSlices, slice.ID)
							nh, err := m.blockchain.GetNodeHealth(slice.NodeID)
							if err != nil {
								l.WithField("slice_id", slice.ID).WithError(err).Error("failed to get slice node health")
								continue
							}
							if nh == blockchain.NodeHealthBad {
								newSlices, mSlice, selectedNodes, err = m.migrateSliceToNewNode(ctx, slice, nodeSliceMap, healthNodes,
									healthNodesMap, selectedNodes, file.ID, newSlices, challengeAlgorithm, hex.EncodeToString(file.Owner))
								if err != nil {
									l.WithFields(logrus.Fields{
										"file_id":  file.ID,
										"slice_id": slice.ID,
									}).WithError(err).Error("migrate red node failed")
								} else {
									fileUpdated = true
									migrateEncSlices = append(migrateEncSlices, mSlice)
								}
							}
							if nh == blockchain.NodeHealthMedium {
								yellowNodeSlices = append(yellowNodeSlices, slice)
							}
						}

						// 5.3 migrate yellow node, only migrate to green nodes
						if len(greenNodes) == 0 {
							l.WithError(err).Warn("empty Green nodes, unable to migrate yellow slices")
						} else {
							for _, slice := range yellowNodeSlices {
								nodeSliceMap := nodeSliceMap(newSlices, slice.ID)
								newSlices, mSlice, selectedNodes, err = m.migrateSliceToNewNode(ctx, slice, nodeSliceMap, greenNodes,
									healthNodesMap, selectedNodes, file.ID, newSlices, challengeAlgorithm, hex.EncodeToString(file.Owner))
								if err != nil {
									l.WithFields(logrus.Fields{
										"file_id":  file.ID,
										"slice_id": slice.ID,
									}).WithError(err).Error("migrate yellow node failed")
								} else {
									fileUpdated = true
									migrateEncSlices = append(migrateEncSlices, mSlice)
								}
							}
						}

						if fileUpdated {
							// add new merkle challenge material
							if challengeAlgorithm == types.MerkleChallengeAlgorithm {
								if err := common.AddSlicesNewMerkleChallenge(m.challenger, file, migrateEncSlices,
									interval, l); err != nil {
									l.WithFields(logrus.Fields{
										"file_id": file.ID,
									}).WithError(err).Error("failed to add slices merkle challenge material")
									return
								}
								l.WithField("file_id", file.ID).Info("file migrate merkle challenge material added successfully")
							}
							// add new pairing challenge material
							if challengeAlgorithm == types.PairingChallengeAlgorithm {
								file.Slices = newSlices
								if err := common.AddSlicesNewPairingChallenge(ctx, pairingConf, m.copier, migrateEncSlices, file, m.blockchain,
									hex.EncodeToString(file.Owner), interval, time.Now().UnixNano(), file.ExpireTime, nil, l); err != nil {
									l.WithFields(logrus.Fields{
										"file_id": file.ID,
									}).WithError(err).Error("failed to add slices pairing challenge material")
									return
								}
								l.WithField("file_id", file.ID).Info("file migrate pairing challenge material added successfully")
							}

							// update file slices
							if err := m.updateFileSlicesOnChain(file.ID, file.Owner, newSlices); err == nil {
								l.WithField("file_id", file.ID).Info("file migrate finished")
							} else {
								l.WithField("file_id", file.ID).WithError(err).Error("updateFileSlicesOnChain failed")
							}
						}

					}(file)
				}
				wgf.Wait()
				l.WithField("namespace", ns.Name).Info("ns migrate finished")
			}(ns)
		}
		wg.Wait()
		l.WithFields(logrus.Fields{
			"namespace_len": len(nsList),
			"end_time":      time.Now().Format("2006-01-02 15:04:05"),
		}).Info("ns list migrate finished")
	}
}

// migrateSliceToNewNode find available healthy node and migrate a slice from bad node to it
// 1. pull slice from healthy node and decrypt it
// 2. encrypt slice and push into the new storage node
// 3. record slice migrated info and update it to the blockchain
func (m FileMaintainer) migrateSliceToNewNode(ctx context.Context, slice blockchain.PublicSliceMeta,
	nodeSliceMap map[string]blockchain.PublicSliceMeta, healthNodes blockchain.NodeHs,
	healthNodesMap map[string]blockchain.NodeH, selectedNodes map[string][]string, fileID string,
	slices []blockchain.PublicSliceMeta, challengeAlgorithm, sourceID string) ([]blockchain.PublicSliceMeta,
	encryptor.EncryptedSlice, map[string][]string, error) {

	var newMigrateEnSlice encryptor.EncryptedSlice
	// find new nodes to migrate slice
	newNodes, err := common.FindNewNodes(healthNodes, selectedNodes[slice.ID])
	if err != nil {
		return slices, newMigrateEnSlice, selectedNodes, errorx.Wrap(err, "failed to find new nodes")
	}

	// pull slice from other healthy nodes
	var plaintext []byte
	var pullErr error
	pulled := false
	for _, node := range selectedNodes[slice.ID] {
		nodeH, exist := healthNodesMap[node]
		if !exist {
			continue
		}

		// pull slice and decrypt
		pulled = true
		plaintext, err = common.PullAndDec(ctx, m.copier, m.encryptor, nodeSliceMap[node], &nodeH.Node, fileID)
		if err != nil {
			pullErr = err
			l.WithFields(logrus.Fields{
				"slice_id":    slice.ID,
				"target_node": node,
			}).WithError(err).Error("failed to recover slice")
		} else {
			break
		}
	}
	if len(plaintext) == 0 {
		if !pulled {
			return slices, newMigrateEnSlice, selectedNodes, errorx.New(errorx.ErrCodeInternal, "no healthy nodes to recover slice")
		}
		return slices, newMigrateEnSlice, selectedNodes, errorx.NewCode(pullErr, errorx.ErrCodeCrypto, "failed to recover slice")
	}

	success := false
	for _, node := range newNodes {
		l.WithFields(logrus.Fields{
			"slice_id":    slice.ID,
			"old_node":    string(slice.NodeID),
			"target_node": string(node.ID),
		}).Debug("migrate slice")

		// push to new node
		if es, err := common.EncAndPush(ctx, m.copier, m.encryptor, plaintext, slice.ID, sourceID, fileID, &node); err == nil {
			l.WithFields(logrus.Fields{
				"slice_id":    slice.ID,
				"old_node":    string(slice.NodeID),
				"target_node": string(node.ID),
			}).Debug("migrate slice pushed")
			success = true
			selectedNodes[slice.ID] = append(selectedNodes[slice.ID], string(node.ID))

			// put migrate record on blockchain
			m.migrateRecordOnChain(string(slice.NodeID), fileID, slice.ID)
			if challengeAlgorithm == types.PairingChallengeAlgorithm {
				// rearrange file slices, remove bad slice and insert new slice with new index
				slices, err = m.rearrangeSlices(slices, slice.ID, string(slice.NodeID), string(es.NodeID), es.CipherText)
				if err != nil {
					l.WithFields(logrus.Fields{
						"slice_id": slice.ID,
						"old_node": string(slice.NodeID),
						"new_node": string(es.NodeID),
					}).WithError(err).Error("rearrangeSlices failed")
					continue
				}
			} else {
				// remove old slice and insert new slice
				newMigrateSlice := blockchain.PublicSliceMeta{
					ID:         es.EncryptedSliceMeta.SliceID,
					CipherHash: es.EncryptedSliceMeta.CipherHash,
					Length:     es.EncryptedSliceMeta.Length,
					NodeID:     es.EncryptedSliceMeta.NodeID,
				}
				slices = append(slices, newMigrateSlice)
				slices = removeSlice(slices, slice)
			}
			newMigrateEnSlice = es
			break
		} else {
			l.WithFields(logrus.Fields{
				"slice_id":    slice.ID,
				"target_node": string(node.ID),
			}).WithError(err).Error("migrate push failed")
		}
	}
	if !success {
		return slices, newMigrateEnSlice, selectedNodes, errorx.New(errorx.ErrCodeInternal, "failed to migrate slice")
	}
	return slices, newMigrateEnSlice, selectedNodes, nil
}

// nodeSliceMap map node->sliceMeta for specific sliceID
func nodeSliceMap(sliceMetas []blockchain.PublicSliceMeta, sliceID string) map[string]blockchain.PublicSliceMeta {
	ret := make(map[string]blockchain.PublicSliceMeta)
	for _, slice := range sliceMetas {
		if slice.ID == sliceID {
			ret[string(slice.NodeID)] = slice
		}
	}
	return ret
}

// rearrangeSlices update slices in file structure saved on blockchain
func (m FileMaintainer) rearrangeSlices(oldSlices []blockchain.PublicSliceMeta, sliceID, badNode,
	newNode string, ciphertext []byte) ([]blockchain.PublicSliceMeta, error) {

	var badSlice blockchain.PublicSliceMeta
	newNodeLargestIdx := 0
	for _, slice := range oldSlices {
		// get bad slice from file slice list
		if slice.ID == sliceID && string(slice.NodeID) == badNode {
			badSlice = slice
		}
		// get largest slice index for new storage node
		if string(slice.NodeID) == newNode {
			if newNodeLargestIdx < slice.SliceIdx {
				newNodeLargestIdx = slice.SliceIdx
			}
		}
	}
	// remove bad slice
	newSlices := removeSlice(oldSlices, badSlice)

	// get slice for new node
	newSliceHash := hash.HashUsingSha256(ciphertext)
	newNodeSlice := blockchain.PublicSliceMeta{
		ID:         sliceID,
		CipherHash: newSliceHash,
		Length:     uint64(len(ciphertext)),
		NodeID:     []byte(newNode),
		SliceIdx:   newNodeLargestIdx + 1,
	}
	newSlices = append(newSlices, newNodeSlice)
	return newSlices, nil
}

// migrateRecordOnChain put migrate record on blockchain
func (m FileMaintainer) migrateRecordOnChain(nodeID, fileID, sliceID string) {
	now := time.Now().UnixNano()
	msg := fileID + sliceID + nodeID + fmt.Sprintf("%d", now)
	sign, err := ecdsa.Sign(m.localNode.PrivateKey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		l.WithFields(logrus.Fields{
			"file_id":     fileID,
			"slice_id":    sliceID,
			"target_node": nodeID,
		}).WithError(err).Error("failed to sign migrate message")
		return
	}
	if err := m.blockchain.SliceMigrateRecord([]byte(nodeID), sign[:], fileID, sliceID, now); err != nil {
		l.WithFields(logrus.Fields{
			"file_id":     fileID,
			"slice_id":    sliceID,
			"target_node": nodeID,
		}).WithError(err).Error("failed to put migrate record on blockchain")
	}
}

// updateFileSlicesOnChain update file slices structure on blockchain
func (m FileMaintainer) updateFileSlicesOnChain(fileID string, owner []byte, slices []blockchain.PublicSliceMeta) error {
	msg, err := json.Marshal(slices)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal slices")
	}
	sign, err := ecdsa.Sign(m.localNode.PrivateKey, hash.HashUsingSha256(msg))
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeCrypto, "failed to sign slices")
	}

	opt := blockchain.UpdateFilePSMOptions{
		FileID:    fileID,
		Owner:     owner,
		Slices:    slices,
		Signature: sign[:],
	}
	return m.blockchain.UpdateFilePublicSliceMeta(&opt)
}

// orderSlicesByIdx rearrange slices with descending sliceIdx order with respect to each storage node
func orderSlicesByIdx(slices []blockchain.PublicSliceMeta) []blockchain.PublicSliceMeta {
	nodeSlicesMap := make(map[string][]blockchain.PublicSliceMeta)
	for _, slice := range slices {
		nodeSlicesMap[string(slice.NodeID)] = append(nodeSlicesMap[string(slice.NodeID)], slice)
	}
	var newSlices []blockchain.PublicSliceMeta
	for _, sliceList := range nodeSlicesMap {
		newSliceList := sortSlices(sliceList)
		newSlices = append(newSlices, newSliceList...)
	}
	return newSlices
}

// sortSlices sort slices by idx in descending order
func sortSlices(slices []blockchain.PublicSliceMeta) []blockchain.PublicSliceMeta {
	newSlices := make([]blockchain.PublicSliceMeta, len(slices))
	for _, slice := range slices {
		newSlices[len(slices)-slice.SliceIdx] = slice
	}
	return newSlices
}

// removeSlice remove old slice
func removeSlice(slices []blockchain.PublicSliceMeta, slice blockchain.PublicSliceMeta) []blockchain.PublicSliceMeta {
	var newSlices []blockchain.PublicSliceMeta
	for _, v := range slices {
		if v.ID == slice.ID && string(v.NodeID) == string(slice.NodeID) && string(v.CipherHash) == string(slice.CipherHash) {
			continue
		}
		newSlices = append(newSlices, v)
	}
	return newSlices
}
