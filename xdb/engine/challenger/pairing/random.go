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

package pairing

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"math"
	"math/big"

	fl_crypto "github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var (
	logger             = logrus.WithField("module", "random-challenger")
	defaultMaxIndexNum = 5

	xchainClient = new(fl_crypto.XchainCryptoClient)
)

// RandChallenger is the pairing based challenger
type RandChallenger struct {
	privateKey ecdsa.PrivateKey // dataOwner node private key

	challengeAlgorithm string

	maxIndexNum int // max file slices number when generating pairing challenge

	pairingConfig types.PairingChallengeConf // configuration for generating challenges
}

// New new a challenger by challenge related configuration
func New(conf *config.ChallengerPairingConf, privkey ecdsa.PrivateKey) (*RandChallenger, error) {
	maxIdx := int(conf.MaxIndexNum)
	if maxIdx == 0 {
		maxIdx = defaultMaxIndexNum
	}

	rc := &RandChallenger{
		privateKey:         privkey,
		maxIndexNum:        maxIdx,
		challengeAlgorithm: types.PairingChallengeAlgorithm,
	}

	pairingConfig := types.PairingChallengeConf{}

	var priv, pub, randu, randv []byte
	var err error
	if conf.Sk == "" || conf.Pk == "" {
		priv, pub, err = xchainClient.GenPairingKeyPair()
		if err != nil {
			return nil, errorx.Wrap(err, "GenPairingKeyPair failed")
		}
	} else {
		priv, err = base64.StdEncoding.DecodeString(conf.Sk)
		if err != nil {
			return nil, errorx.Wrap(err, "DecodeString failed")
		}
		pub, err = base64.StdEncoding.DecodeString(conf.Pk)
		if err != nil {
			return nil, errorx.Wrap(err, "DecodeString failed")
		}
	}
	logger.WithField("pdpPrivatekey", base64.StdEncoding.EncodeToString(priv)).Info("engine initialization")
	logger.WithField("pdpPublickey", base64.StdEncoding.EncodeToString(pub)).Info("engine initialization")

	if conf.Randu == "" {
		randu, err = xchainClient.RandomWithinPairingOrder()
		if err != nil {
			return nil, errorx.Wrap(err, "RandomWithinPairingOrder failed")
		}
	} else {
		randu, err = base64.StdEncoding.DecodeString(conf.Randu)
		if err != nil {
			return nil, errorx.Wrap(err, "DecodeString failed")
		}
	}
	if conf.Randv == "" {
		randv, err = xchainClient.RandomWithinPairingOrder()
		if err != nil {
			return nil, errorx.Wrap(err, "RandomWithinPairingOrder failed")
		}
	} else {
		randv, err = base64.StdEncoding.DecodeString(conf.Randv)
		if err != nil {
			return nil, errorx.Wrap(err, "DecodeString failed")
		}
	}
	logger.WithField("randU", base64.StdEncoding.EncodeToString(randu)).Info("engine initialization")
	logger.WithField("randV", base64.StdEncoding.EncodeToString(randv)).Info("engine initialization")

	pairingConfig.Privkey = priv
	pairingConfig.Pubkey = pub
	pairingConfig.RandU = randu
	pairingConfig.RandV = randv

	rc.pairingConfig = pairingConfig

	logger.WithField("maxIndexNum", maxIdx).Info("RandChallenger initialization")
	return rc, nil
}

// GetChallengeConf get challenge configuration
func (m *RandChallenger) GetChallengeConf() (string, types.PairingChallengeConf) {
	return m.challengeAlgorithm, m.pairingConfig
}

// GenerateChallenge randomly select a subset of the index list to generate challenge
// calculate random number for the challenge round by private key and challenge interval
func (m *RandChallenger) GenerateChallenge(sliceIdxList []int, interval int64) ([][]byte, [][]byte, int64, []byte, error) {
	idxNum := len(sliceIdxList)
	selectNum := int(math.Sqrt(float64(idxNum)-1)) + 1
	if selectNum > m.maxIndexNum {
		selectNum = m.maxIndexNum
	}

	var selectedIdx []int
	selectedMap := make(map[int64]bool)
	max := big.NewInt(int64(idxNum))
	for len(selectedIdx) < selectNum {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return nil, nil, 0, nil, errorx.Wrap(err, "failed to rand int")
		}
		if !selectedMap[idx.Int64()] {
			selectedIdx = append(selectedIdx, sliceIdxList[idx.Int64()])
			selectedMap[idx.Int64()] = true
		}
	}

	round := common.GetPairingChallengeRound(interval)
	indices, vs, randNum, err := xchainClient.GenPairingChallenge(selectedIdx, round, m.pairingConfig.Privkey)
	if err != nil {
		return nil, nil, 0, nil, errorx.Wrap(err, "GenPairingChallenge failed")
	}
	return indices, vs, round, randNum, nil
}

func (m *RandChallenger) Close() {}

// Setup not implemented for random challenge
func (m *RandChallenger) Setup(sliceData []byte, rangeAmount int) (c []ctype.RangeHash, err error) {
	return c, errorx.New(errorx.ErrCodeInternal, "pairing not implemented method Setup")
}

// Save not implemented for random challenge
func (m *RandChallenger) Save(cms []ctype.Material) error {
	return errorx.New(errorx.ErrCodeInternal, "pairing not implemented method Save")
}

// Take not implemented for random challenge
func (m *RandChallenger) Take(fileID string, sliceID string, nodeID []byte) (c ctype.RangeHash, err error) {
	return c, errorx.New(errorx.ErrCodeInternal, "pairing not implemented method Take")
}
