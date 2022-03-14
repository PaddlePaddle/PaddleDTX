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

package merkle

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	fl_crypto "github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/ldbstorage"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var (
	logger       = logrus.WithField("module", "merkle-challenger")
	xchainClient = new(fl_crypto.XchainCryptoClient)

	defaultLDBRoot            = "/root/xdb/data/challenger"
	defaultShrinkSize         = 500
	defaultSegmentSize        = 5
	MerkleChallengeSetUpRange = 20000
)

// RandChallenger is the Merkle-Tree based Challenger
type RandChallenger struct {
	closeOnce sync.Once

	shrinkSize  uint64 // maximum of segment (end_idx - start_idx)
	segmentSize uint64 // number of content segments to make merkle tree

	privateKey ecdsa.PrivateKey // dataOwner node private key

	storage ctype.MaterialStorage // challenge material storage handler

	challengeAlgorithm string
}

// New new a challenger by challenge related configuration
func New(conf *config.ChallengerMerkleConf, privkey ecdsa.PrivateKey) (*RandChallenger, error) {
	ldbRoot := conf.LeveldbRoot
	if len(ldbRoot) == 0 {
		ldbRoot = defaultLDBRoot
	}
	logger.WithField("leveldb-root", ldbRoot).Info("challenger initialization")

	shrinkSize := uint64(conf.ShrinkSize)
	if shrinkSize == 0 {
		shrinkSize = uint64(defaultShrinkSize)
	}
	logger.WithField("shrink-size", shrinkSize).Info("challenger initialization")

	segmentSize := uint64(conf.SegmentSize)
	if segmentSize == 0 {
		segmentSize = uint64(defaultSegmentSize)
	}
	logger.WithField("segment-size", segmentSize).Info("challenger initialization")

	// create dir if not exist
	// only create the outer dir, for example "challenger" in "/root/xdb/data/challenger"
	// if "/root/xdb/data" is not exist, we should panic
	// because maybe the operator forgot to mount "/root/xdb/data" from host machine
	if _, err := os.Stat(ldbRoot); err != nil {
		if err := os.Mkdir(ldbRoot, 0777); err != nil {
			return nil, errorx.NewCode(err, errorx.ErrCodeConfig, "failed to mkdir for ldb")
		}
	}

	storage, err := ldbstorage.New(ldbRoot)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to create challenge storager")
	}

	rc := &RandChallenger{
		shrinkSize:         shrinkSize,
		segmentSize:        segmentSize,
		privateKey:         privkey,
		storage:            storage,
		challengeAlgorithm: types.MerkleChallengeAlgorithm,
	}

	return rc, nil
}

// GetChallengeConf get challenge configuration
func (m *RandChallenger) GetChallengeConf() (a string, conf types.PairingChallengeConf) {
	return m.challengeAlgorithm, conf
}

// GenerateChallenge not implemented for merkle challenge
func (m *RandChallenger) GenerateChallenge(sliceIdxList []int, interval int64) (i [][]byte, v [][]byte, r int64, w []byte, err error) {
	return i, v, r, w, errorx.New(errorx.ErrCodeInternal, "merkle not implemented method GenerateChallenge")
}

// Setup prepare challenge materials for challenge loop later
func (m *RandChallenger) Setup(sliceData []byte, rangeAmount int) ([]ctype.RangeHash, error) {
	length := len(sliceData)

	logger.WithFields(logrus.Fields{
		"length":       length,
		"range_amount": rangeAmount,
	}).Debug("setup started")

	curmount := math.Ceil(float64(rangeAmount) / float64(MerkleChallengeSetUpRange))

	var mRangeList []ctype.RangeHash
	cr := make(chan []ctype.RangeHash, int(curmount))
	wg := sync.WaitGroup{}

	for i := 1; i <= int(curmount); i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int) {
			defer wg.Done()
			mamount := 0
			if rangeAmount <= MerkleChallengeSetUpRange {
				mamount = rangeAmount
			} else if i*MerkleChallengeSetUpRange > rangeAmount {
				mamount = rangeAmount - (i-1)*MerkleChallengeSetUpRange
			} else {
				mamount = MerkleChallengeSetUpRange
			}
			pRanges := m.generateMerkleRanges(sliceData, mamount, length)
			cr <- pRanges
		}(&wg, i)
	}
	wg.Wait()
	close(cr)
	for er := range cr {
		mRangeList = append(mRangeList, er...)
	}
	return mRangeList, nil
}

// generateMerkleRanges generate content segments to make merkle tree
func (m *RandChallenger) generateMerkleRanges(sliceData []byte, rangeAmount, length int) []ctype.RangeHash {
	ranges := make([]ctype.RangeHash, 0, rangeAmount)
	rand.Seed(time.Now().UnixNano())

	selected := make(map[string]struct{})
	for {
		if len(ranges) == rangeAmount {
			break
		}
		sr := make([]ctype.Range, 0, m.segmentSize)
		hs := make([][]byte, 0, m.segmentSize)
		for {
			if len(sr) >= int(m.segmentSize) {
				break
			}

			first := rand.Int() % length
			second := rand.Int() % length

			start := int(math.Min(float64(first), float64(second)))
			end := int(math.Max(float64(first), float64(second)))
			start, end = m.shrink(start, end)

			selectKey := fmt.Sprintf("%d/%d", start, end)
			if _, exist := selected[selectKey]; exist {
				continue
			}
			selected[selectKey] = struct{}{}

			sr = append(sr, ctype.Range{
				Start: uint64(start),
				End:   uint64(end),
			})
			h := xchainClient.HashUsingSha256(sliceData[start:end])
			hs = append(hs, h)
		}
		// calculate merkle root
		h := xchainClient.GetMerkleRoot(hs)
		current := ctype.RangeHash{
			Ranges: sr,
			Hash:   h,
		}
		ranges = append(ranges, current)
	}
	return ranges
}

// Save save challenge material
func (m *RandChallenger) Save(cms []ctype.Material) error {
	if err := m.storage.Save(cms); err != nil {
		return errorx.Wrap(err, "failed to save challenge materials")
	}
	return nil
}

// Take take a challenge material for publishing a challenge
func (m *RandChallenger) Take(fileID string, sliceID string, nodeID []byte) (
	ctype.RangeHash, error) {

	keyList, err := m.storage.NewIterator([]byte(fmt.Sprintf("%s:%s:%x", fileID, sliceID, nodeID)))
	if err != nil {
		return ctype.RangeHash{}, errorx.Wrap(errorx.ErrNotFound, "no available keyList")
	}

	for _, key := range keyList {
		cm, err := m.storage.Load(key)
		if err != nil {
			continue
		}
		var rh *ctype.RangeHash
		newCm := ctype.Material{}
		for i := range cm.Ranges {
			if !cm.Ranges[i].Used {
				cm.Ranges[i].Used = true
				rh = &cm.Ranges[i]
				newCm.Ranges = cm.Ranges[i:]
				break
			}
		}
		if rh == nil {
			continue
		}

		if err := m.storage.Update(newCm, key); err != nil {
			continue
		}
		return *rh, nil
	}
	return ctype.RangeHash{}, errorx.Wrap(errorx.ErrNotFound, "no available challenger materials")
}

func (m *RandChallenger) Close() {
	m.closeOnce.Do(m.storage.Close)
}

func (m *RandChallenger) shrink(start, end int) (int, int) {
	if end-start > int(m.shrinkSize) {
		end = start + end%int(m.shrinkSize)
	}

	return start, end
}
