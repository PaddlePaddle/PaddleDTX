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
	"context"
	"io"

	fl_crypto "github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
)

var xchainClient = new(fl_crypto.XchainCryptoClient)

// CommonCopier defines slice copier when migrating a file
type CommonCopier interface {
	Push(ctx context.Context, id, sourceID string, r io.Reader, node *blockchain.Node) error
	Pull(ctx context.Context, id, fileID string, node *blockchain.Node) (io.ReadCloser, error)
	ReplicaExpansion(ctx context.Context, opt *copier.ReplicaExpOptions, enc CommonEncryptor, ca, sourceID, fileID string) (
		[]blockchain.PublicSliceMeta, []encryptor.EncryptedSlice, error)
}

// CommonChallenger defines Merkle-Tree based / pairing based challenger
type CommonChallenger interface {
	Setup(sliceData []byte, rangeAmount int) ([]ctype.RangeHash, error)
	Save(cms []ctype.Material) error
	Take(fileID string, sliceID string, nodeID []byte) (ctype.RangeHash, error)

	GetChallengeConf() (string, types.PairingChallengeConf)
}

// CommonEncryptor defines encryptor for file encryption and decryption when adding more merkle challenges or migrate files
type CommonEncryptor interface {
	Encrypt(r io.Reader, opt *encryptor.EncryptOptions) (encryptor.EncryptedSlice, error)
	Recover(r io.Reader, opt *encryptor.RecoverOptions) ([]byte, error)
}

// CommonChain defines several contract/chaincode methods related to file migration and node/file health
type CommonChain interface {
	ListNodes() (blockchain.Nodes, error)
	GetNode(id []byte) (blockchain.Node, error)
	GetNodeHealth(id []byte) (string, error)
	GetHeartbeatNum(id []byte, timestamp int64) (int, error)

	ListFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error)
	ListFileNs(opt *blockchain.ListNsOptions) ([]blockchain.Namespace, error)
	UpdateFilePublicSliceMeta(opt *blockchain.UpdateFilePSMOptions) error
}
