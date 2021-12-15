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

package pdp

import (
	"crypto/rand"
	"encoding/base64"
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

// RandChallenger is the PDP based challenger
type RandChallenger struct {
	privateKey ecdsa.PrivateKey // dataOwner node private key

	challengeAlgorithm string

	maxIndexNum int // max file slices number when generating pdp challenge

	pdpConfig types.PDP // configuration for generating challenges
}

// New new a challenger by challenge related configuration
func New(conf *config.ChallengerPdpConf, privkey ecdsa.PrivateKey) (*RandChallenger, error) {
	maxIdx := int(conf.MaxIndexNum)
	if maxIdx == 0 {
		maxIdx = defaultMaxIndexNum
	}

	rc := &RandChallenger{
		privateKey:         privkey,
		maxIndexNum:        maxIdx,
		challengeAlgorithm: types.PDPChallengeAlgorithm,
	}

	pdpConfig := types.PDP{}

	var pdpPriv, pdpPub, randu, randv []byte
	var err error
	if conf.Sk == "" || conf.Pk == "" {
		pdpPriv, pdpPub, err = xchainClient.GenPDPRandomKeyPair()
		if err != nil {
			return nil, errorx.Wrap(err, "GenPDPRandomKeyPair failed")
		}
	} else {
		pdpPriv, err = base64.StdEncoding.DecodeString(conf.Sk)
		if err != nil {
			return nil, errorx.Wrap(err, "DecodeString failed")
		}
		pdpPub, err = base64.StdEncoding.DecodeString(conf.Pk)
		if err != nil {
			return nil, errorx.Wrap(err, "DecodeString failed")
		}
	}
	logger.WithField("pdpPrivatekey", base64.StdEncoding.EncodeToString(pdpPriv)).Info("engine initialization")
	logger.WithField("pdpPublickey", base64.StdEncoding.EncodeToString(pdpPub)).Info("engine initialization")

	if conf.Randu == "" {
		randu, err = xchainClient.RandomPDPWithinOrder()
		if err != nil {
			return nil, errorx.Wrap(err, "RandomPDPWithinOrder failed")
		}
	} else {
		randu, err = base64.StdEncoding.DecodeString(conf.Randu)
		if err != nil {
			return nil, errorx.Wrap(err, "DecodeString failed")
		}
	}
	if conf.Randv == "" {
		randv, err = xchainClient.RandomPDPWithinOrder()
		if err != nil {
			return nil, errorx.Wrap(err, "RandomPDPWithinOrder failed")
		}
	} else {
		randv, err = base64.StdEncoding.DecodeString(conf.Randv)
		if err != nil {
			return nil, errorx.Wrap(err, "DecodeString failed")
		}
	}
	logger.WithField("pdpRandU", base64.StdEncoding.EncodeToString(randu)).Info("engine initialization")
	logger.WithField("pdpRandV", base64.StdEncoding.EncodeToString(randv)).Info("engine initialization")

	pdpConfig.PdpPrivkey = pdpPriv
	pdpConfig.PdpPubkey = pdpPub
	pdpConfig.RandU = randu
	pdpConfig.RandV = randv

	rc.pdpConfig = pdpConfig

	logger.WithField("maxIndexNum", maxIdx).Info("RandChallenger initialization")
	return rc, nil
}

// GetChallengeConf get challenge configuration
func (m *RandChallenger) GetChallengeConf() (string, types.PDP) {
	return m.challengeAlgorithm, m.pdpConfig
}

// GenerateChallenge randomly select index list to generate challenge
func (m *RandChallenger) GenerateChallenge(maxIdx int) ([][]byte, [][]byte, error) {
	selectNum := int(math.Sqrt(float64(maxIdx)-1)) + 1
	if selectNum > m.maxIndexNum {
		selectNum = m.maxIndexNum
	}

	var idxList []int
	idxMap := make(map[int64]bool)
	max := big.NewInt(int64(maxIdx + 1))
	for len(idxList) < selectNum {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return nil, nil, errorx.Wrap(err, "failed to rand int")
		}
		if !idxMap[idx.Int64()] && idx.Cmp(big.NewInt(0)) != 0 {
			idxList = append(idxList, int(idx.Int64()))
			idxMap[idx.Int64()] = true
		}
	}

	indices, vs, err := xchainClient.GeneratePDPChallenge(idxList)
	if err != nil {
		return nil, nil, errorx.Wrap(err, "GeneratePDPChallenge failed")
	}
	return indices, vs, nil
}

func (m *RandChallenger) Close() {}

// Setup not implemented for random challenge
func (m *RandChallenger) Setup(sliceData []byte, rangeAmount int) (c []ctype.RangeHash, err error) {
	return c, errorx.New(errorx.ErrCodeInternal, "pdp not implemented method Setup")
}

// Save not implemented for random challenge
func (m *RandChallenger) Save(cms []ctype.Material) error {
	return errorx.New(errorx.ErrCodeInternal, "pdp not implemented method Save")
}

// Take not implemented for random challenge
func (m *RandChallenger) Take(fileID string, sliceID string, nodeID []byte) (c ctype.RangeHash, err error) {
	return c, errorx.New(errorx.ErrCodeInternal, "pdp not implemented method Take")
}

// NewSetup not implemented for random challenge
func (m *RandChallenger) NewSetup(sliceData []byte, rangeAmount int, merkleMaterialQueue chan<- ctype.Material, cm ctype.Material) error {
	return errorx.New(errorx.ErrCodeInternal, "pdp not implemented method NewSetup")
}
