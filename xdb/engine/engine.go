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
	"context"
	"io"

	"github.com/sirupsen/logrus"

	fl_crypto "github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/aes"
	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/peer"
)

var (
	logger = logrus.WithField("module", "engine")

	xchainClient = new(fl_crypto.XchainCryptoClient)
)

// Slicer cuts data into blocks
type Slicer interface {
	// Slice Async slicing
	Slice(ctx context.Context, r io.Reader, opt *slicer.SliceOptions, onErr func(err error)) chan slicer.Slice
	GetBlockSize() int
}

// Encryptor encrypts data and decrypts encoded data
type Encryptor interface {
	GetKey(fileID, sliceID string, nodeID []byte) aes.AESKey
	Encrypt(r io.Reader, opt *encryptor.EncryptOptions) (encryptor.EncryptedSlice, error)
	Recover(r io.Reader, opt *encryptor.RecoverOptions) ([]byte, error)
}

// Challenger generates challenge requests as dataOwner-node for storage-nodes to answer
//  Pairing-based and MerkleTree-based are both supported
//  see more from engine.challenger
type Challenger interface {
	// pairing based Challenge
	GenerateChallenge(sliceIdxList []int, interval int64) ([][]byte, [][]byte, int64, []byte, error)

	// merkle Challenge
	Setup(sliceData []byte, rangeAmount int) ([]ctype.RangeHash, error)
	Save(cms []ctype.Material) error
	Take(fileID string, sliceID string, nodeID []byte) (ctype.RangeHash, error)

	GetChallengeConf() (string, types.PairingChallengeConf)
	Close()
}

// Copier selects Storage Nodes randomly from healthy candidates.
//  You can call Push() to push slices onto Storage Node, and Pull() to pull slices from Storage Node.
//  If you want more Storage Nodes, you can call ReplicaExpansion(),
//  and it pulls slices from original nodes and decrypts and re-encrypts those slices,
//  then push them onto new Storage Nodes.
type Copier interface {
	Select(slice slicer.Slice, nodes blockchain.NodeHs, opt *copier.SelectOptions) (copier.LocatedSlice, error)
	Push(ctx context.Context, id, sourceID string, r io.Reader, node *blockchain.Node) error
	Pull(ctx context.Context, id, fileID string, node *blockchain.Node) (io.ReadCloser, error)
	ReplicaExpansion(ctx context.Context, opt *copier.ReplicaExpOptions, enc common.CommonEncryptor,
		challengeAlgorithm, sourceID, fileID string) ([]blockchain.PublicSliceMeta, []encryptor.EncryptedSlice, error)
}

// Blockchain defines some contract methods
//  For xchain they are contract methods, and for fabric they are chaincode methods
//  see more from blockchain.xchain and blockchain.fabric
type Blockchain interface {
	// The following contract methods are used by storage node,
	// for distributed governance and healthy check of nodes
	AddNode(opt *blockchain.AddNodeOptions) error
	ListNodes() (blockchain.Nodes, error)
	GetNode(id []byte) (blockchain.Node, error)
	NodeOffline(opt *blockchain.NodeOperateOptions) error
	NodeOnline(opt *blockchain.NodeOperateOptions) error
	Heartbeat(id, sig []byte, timestamp int64) error
	GetHeartbeatNum(id []byte, timestamp int64) (int, error)
	GetNodeHealth(id []byte) (string, error)
	ListNodesExpireSlice(opt *blockchain.ListNodeSliceOptions) ([]string, error)
	GetSliceMigrateRecords(opt *blockchain.NodeSliceMigrateOptions) (string, error)

	// The following contract methods are used by dataOwner node
	PublishFile(file *blockchain.PublishFileOptions) error
	GetFileByName(owner []byte, ns, name string) (blockchain.File, error)
	GetFileByID(id string) (blockchain.File, error)
	UpdateFileExpireTime(opt *blockchain.UpdateExptimeOptions) (blockchain.File, error)
	AddFileNs(opt *blockchain.AddNsOptions) error
	UpdateNsReplica(opt *blockchain.UpdateNsReplicaOptions) error
	UpdateFilePublicSliceMeta(opt *blockchain.UpdateFilePSMOptions) error
	SliceMigrateRecord(id, sig []byte, fid, sid string, ctime int64) error
	GetNsByName(owner []byte, name string) (blockchain.Namespace, error)
	ListFileNs(opt *blockchain.ListNsOptions) ([]blockchain.Namespace, error)
	ListFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error)
	ListExpiredFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error)
	// The following contract methods used for authorizers to operate the file authorization application
	GetAuthApplicationByID(authID string) (blockchain.FileAuthApplication, error)
	ListFileAuthApplications(opt *blockchain.ListFileAuthOptions) (blockchain.FileAuthApplications, error)
	ConfirmFileAuthApplication(opt *blockchain.ConfirmFileAuthOptions) error
	RejectFileAuthApplication(opt *blockchain.ConfirmFileAuthOptions) error

	ListChallengeRequests(opt *blockchain.ListChallengeOptions) ([]blockchain.Challenge, error)
	ChallengeRequest(opt *blockchain.ChallengeRequestOptions) error
	ChallengeAnswer(opt *blockchain.ChallengeAnswerOptions) ([]byte, error)
	GetChallengeByID(id string) (blockchain.Challenge, error)
}

// Storage stores files locally
type Storage interface {
	Save(key string, value io.Reader) error
	Load(key string) (io.ReadCloser, error)
	Delete(key string) (bool, error)
	Exist(key string) (bool, error)
	LoadStr(key string) (string, error)
	SaveAndUpdate(key string, value io.Reader) error
}

type Engine struct {
	slicer     Slicer
	encryptor  Encryptor
	challenger Challenger
	chain      Blockchain
	copier     Copier
	storage    Storage

	monitor *Monitor
}

// NewEngineOption contains parameters for initiating Engine
type NewEngineOption struct {
	LocalNode peer.Local

	Slicer     Slicer
	Encryptor  Encryptor
	Challenger Challenger
	Chain      Blockchain
	Copier     Copier
	Storage    Storage
}

// NewEngine initiates Engine by the node's configuration file
func NewEngine(conf *config.MonitorConf, opt *NewEngineOption) (*Engine, error) {
	monitor, err := newMonitor(conf, opt)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to create monitor")
	}
	e := &Engine{
		slicer:     opt.Slicer,
		encryptor:  opt.Encryptor,
		challenger: opt.Challenger,
		chain:      opt.Chain,
		copier:     opt.Copier,
		storage:    opt.Storage,
		monitor:    monitor,
	}
	return e, nil
}

// Start starts Engine
func (e *Engine) Start(ctx context.Context) error {
	return e.monitor.Start(ctx)
}

func (e *Engine) Close() {
	if e.challenger != nil {
		e.challenger.Close()
	}
	if e.monitor != nil {
		e.monitor.Close()
	}
}
