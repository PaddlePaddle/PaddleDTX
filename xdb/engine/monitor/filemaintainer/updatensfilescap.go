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
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/hash"
)

// updateNsFilesCap updates files capacity of namespaces on blockchain
func (m *FileMaintainer) updateNsFilesCap(ctx context.Context) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(m.localNode.PrivateKey)

	l := logger.WithField("runner", "ns files cap update loop")
	defer l.Info("runner stopped")

	ticker := time.NewTicker(m.nsFilesCapUpInterval)
	defer ticker.Stop()

	m.doneUpdNsFilesCapC = make(chan struct{})
	defer close(m.doneUpdNsFilesCapC)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		listNsOpt := blockchain.ListNsOptions{
			Owner:       pubkey[:],
			TimeStart:   0,
			TimeEnd:     time.Now().UnixNano(),
			CurrentTime: time.Now().UnixNano(),
		}
		nsList, err := m.blockchain.ListFileNs(ctx, &listNsOpt)
		if err != nil {
			l.WithError(err).Error("failed to find ns list")
			continue
		}
		if len(nsList) == 0 {
			l.Info("no namespace found")
			continue
		}
		for _, ns := range nsList {
			timestamp := time.Now().UnixNano()
			mes := fmt.Sprintf("%s,%d", ns.Name, timestamp)
			sig, err := ecdsa.Sign(m.localNode.PrivateKey, hash.Hash([]byte(mes)))
			if err != nil {
				l.WithError(err).Warn("failed to sign ns files cap")
				continue
			}
			updateNsCapOpt := blockchain.UpdateNsFilesCapOptions{
				Owner:       pubkey[:],
				Name:        ns.Name,
				CurrentTime: timestamp,
				Signature:   sig[:],
			}
			newNs, err := m.blockchain.UpdateNsFilesCap(ctx, &updateNsCapOpt)
			if err != nil {
				if errorx.Is(err, errorx.ErrCodeAlreadyUpdate) {
					l.WithField("namespace", ns.Name).Info("ns-struct-size is already updated, not need to modify again")
				} else {
					l.WithField("namespace", ns.Name).WithError(err).Warn("failed to update ns files cap")
				}
				continue
			}

			l.WithFields(logrus.Fields{
				"namespace":  ns.Name,
				"ns_old_cap": ns.FilesStruSize,
				"ns_new_cap": newNs.FilesStruSize,
			}).Info("success to update ns files struct size")
		}
	}
}
