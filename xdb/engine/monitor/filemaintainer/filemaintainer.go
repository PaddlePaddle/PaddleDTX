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
	defaultFileMigrateInterval      = time.Hour * 1
	defaultNsFilesCapUpdateInterval = time.Minute * 53
)

var (
	logger = logrus.WithField("monitor", "filemaintainer")
)

type Copier interface {
	Push(ctx context.Context, id, sourceId string, r io.Reader, node *blockchain.Node) error
	Pull(ctx context.Context, id, fileId string, node *blockchain.Node) (io.ReadCloser, error)
	ReplicaExpansion(ctx context.Context, opt *copier.ReplicaExpOptions, enc common.MigrateEncryptor,
		challengAlgorithm, sourceId, fileId string) ([]blockchain.PublicSliceMeta, []encryptor.EncryptedSlice, error)
}

type Encryptor interface {
	Encrypt(ctx context.Context, r io.Reader, opt *encryptor.EncryptOptions) (
		encryptor.EncryptedSlice, error)
	Recover(ctx context.Context, r io.Reader, opt *encryptor.RecoverOptions) (
		[]byte, error)
}

type Blockchain interface {
	PublishFile(ctx context.Context, file *blockchain.PublishFileOptions) error
	ListFiles(ctx context.Context, opt *blockchain.ListFileOptions) ([]blockchain.File, error)
	GetFileByID(ctx context.Context, id string) (blockchain.File, error)
	UpdateNsFilesCap(ctx context.Context, opt *blockchain.UpdateNsFilesCapOptions) (blockchain.Namespace, error)
	ListFileNs(ctx context.Context, opt *blockchain.ListNsOptions) ([]blockchain.Namespace, error)
	UpdateFilePublicSliceMeta(ctx context.Context, opt *blockchain.UpdateFilePSMOptions) error
	SliceMigrateRecord(ctx context.Context, nodeID, sig []byte, fileID, sliceID string, ctime int64) error

	ListNodes(ctx context.Context) (blockchain.Nodes, error)
	GetNode(ctx context.Context, id []byte) (blockchain.Node, error)
	GetNodeHealth(ctx context.Context, id []byte) (string, error)
	GetHeartbeatNum(ctx context.Context, id []byte, timestamp int64) (int, error)
}

type Challenger interface {
	// merkle Challenge
	Setup(sliceData []byte, rangeAmount int) ([]ctype.RangeHash, error)
	NewSetup(sliceData []byte, rangeAmount int, merkleMaterialQueue chan<- ctype.Material, cm ctype.Material) error
	Save(ctx context.Context, cms []ctype.Material) error
	Take(ctx context.Context, fileID string, sliceID string, nodeID []byte) (ctype.RangeHash, error)

	GetChallengeConf() (string, types.PDP)
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
//  The other part of its main work is to update files capacity of namespaces on blockchain
type FileMaintainer struct {
	localNode  peer.Local
	blockchain Blockchain
	copier     Copier
	encryptor  Encryptor
	challenger Challenger

	challengerInterval int64

	fileMigrateInterval  time.Duration
	nsFilesCapUpInterval time.Duration

	doneMigrateC       chan struct{} //doneMigrateC will be closed when loop breaks
	doneUpdNsFilesCapC chan struct{} //doneUpdNsFilesCapC will be closed when loop breaks
}

func New(conf *config.MonitorConf, opt *NewFileMaintainerOptions, interval int64) (*FileMaintainer, error) {

	fileMigrateInterval := time.Duration(conf.FilemigrateInterval) * time.Hour
	if fileMigrateInterval == 0 {
		fileMigrateInterval = defaultFileMigrateInterval
	}

	logger.WithFields(logrus.Fields{
		"filemigrate-interval":  fileMigrateInterval,
		"nsfilescapup-interval": defaultNsFilesCapUpdateInterval,
	}).Info("monitor initialize...")

	return &FileMaintainer{
		localNode:            opt.LocalNode,
		blockchain:           opt.Blockchain,
		copier:               opt.Copier,
		encryptor:            opt.Encryptor,
		challenger:           opt.Challenger,
		challengerInterval:   interval,
		fileMigrateInterval:  fileMigrateInterval,
		nsFilesCapUpInterval: defaultNsFilesCapUpdateInterval,
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

// UpdateNsFilesCap starts task to update files'cap
func (m *FileMaintainer) UpdateNsFilesCap(ctx context.Context) {
	go m.updateNsFilesCap(ctx)
}

// StopUpdateNsFilesCap stops task updating files'cap
func (m *FileMaintainer) StopUpdateNsFilesCap() {
	if m.doneUpdNsFilesCapC == nil {
		return
	}

	logger.Info("stops task updating files'cap ...")

	select {
	case <-m.doneUpdNsFilesCapC:
		return
	default:
	}

	<-m.doneUpdNsFilesCapC
}
