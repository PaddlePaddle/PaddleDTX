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
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/peer"
)

const (
	// Defines the default interval for files migration
	defaultFileMigrateInterval = time.Hour * 1
)

var (
	logger = logrus.WithField("monitor", "filemaintainer")
)

type Copier interface {
	Push(ctx context.Context, id, sourceID string, r io.Reader, node *blockchain.Node) error
	Pull(ctx context.Context, id, fileID string, node *blockchain.Node) (io.ReadCloser, error)
	ReplicaExpansion(ctx context.Context, opt *copier.ReplicaExpOptions, enc common.CommonEncryptor,
		challengeAlgorithm, sourceID, fileID string) ([]blockchain.PublicSliceMeta, []encryptor.EncryptedSlice, error)
}

type Encryptor interface {
	Encrypt(r io.Reader, opt *encryptor.EncryptOptions) (encryptor.EncryptedSlice, error)
	Recover(r io.Reader, opt *encryptor.RecoverOptions) ([]byte, error)
}

type Blockchain interface {
	PublishFile(file *blockchain.PublishFileOptions) error
	ListFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error)
	GetFileByID(id string) (blockchain.File, error)
	ListFileNs(opt *blockchain.ListNsOptions) ([]blockchain.Namespace, error)
	UpdateFilePublicSliceMeta(opt *blockchain.UpdateFilePSMOptions) error
	SliceMigrateRecord(nodeID, sig []byte, fileID, sliceID string, ctime int64) error

	ListNodes() (blockchain.Nodes, error)
	GetNode(id []byte) (blockchain.Node, error)
	GetNodeHealth(id []byte) (string, error)
	GetHeartbeatNum(id []byte, timestamp int64) (int, error)
}

type Challenger interface {
	// merkle Challenge
	Setup(sliceData []byte, rangeAmount int) ([]ctype.RangeHash, error)
	Save(cms []ctype.Material) error
	Take(fileID string, sliceID string, nodeID []byte) (ctype.RangeHash, error)

	GetChallengeConf() (string, types.PairingChallengeConf)
	Close()
}

type NewFileMaintainerOptions struct {
	LocalNode  peer.Local
	Blockchain Blockchain
	Copier     Copier
	Encryptor  Encryptor
	Challenger Challenger
}

// FileMaintainer runs if local node is dataOwner-node, and its main work is to check storage-nodes health conditions
//  and migrate slices from bad nodes to healthy nodes.
type FileMaintainer struct {
	localNode  peer.Local
	blockchain Blockchain
	copier     Copier
	encryptor  Encryptor
	challenger Challenger

	challengerInterval int64

	fileMigrateInterval time.Duration

	doneMigrateC chan struct{} //doneMigrateC will be closed when loop breaks
}

func New(conf *config.MonitorConf, opt *NewFileMaintainerOptions, interval int64) (*FileMaintainer, error) {

	fileMigrateInterval := time.Duration(conf.FilemigrateInterval) * time.Hour
	if fileMigrateInterval == 0 {
		fileMigrateInterval = defaultFileMigrateInterval
	}

	logger.WithFields(logrus.Fields{
		"filemigrate-interval": fileMigrateInterval,
	}).Info("monitor initialize...")

	return &FileMaintainer{
		localNode:           opt.LocalNode,
		blockchain:          opt.Blockchain,
		copier:              opt.Copier,
		encryptor:           opt.Encryptor,
		challenger:          opt.Challenger,
		challengerInterval:  interval,
		fileMigrateInterval: fileMigrateInterval,
	}, nil
}

// Migrate starts file migration
func (m *FileMaintainer) Migrate(ctx context.Context) {
	go m.migrate(ctx)
}

// StopMigrate stops file migration
func (m *FileMaintainer) StopMigrate() {
	if m.doneMigrateC == nil {
		return
	}

	logger.Info("stops file migration ...")

	select {
	case <-m.doneMigrateC:
		return
	default:
	}

	<-m.doneMigrateC
}
