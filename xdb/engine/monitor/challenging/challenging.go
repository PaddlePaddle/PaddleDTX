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

package challenging

import (
	"context"
	"io"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
)

const (
	DefaultRequestInterval = time.Minute * 60
	defaultAnswerInterval  = time.Minute * 10
)

var (
	logger = logrus.WithField("monitor", "challenging")
)

type ChallengeDB interface {
	GenerateChallenge(sliceIdxList []int, interval int64) ([][]byte, [][]byte, int64, []byte, error)

	// merkle Challenge
	Setup(sliceData []byte, rangeAmount int) ([]ctype.RangeHash, error)
	Save(cms []ctype.Material) error
	Take(fileID string, sliceID string, nodeID []byte) (ctype.RangeHash, error)

	GetChallengeConf() (string, types.PairingChallengeConf)
}

type SliceStorage interface {
	Load(key string) (io.ReadCloser, error)
}

type Blockchain interface {
	ListFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error)
	ListFileNs(opt *blockchain.ListNsOptions) ([]blockchain.Namespace, error)
	ListChallengeRequests(opt *blockchain.ListChallengeOptions) ([]blockchain.Challenge, error)
	ChallengeRequest(opt *blockchain.ChallengeRequestOptions) error
	ChallengeAnswer(opt *blockchain.ChallengeAnswerOptions) ([]byte, error)
	NodeOffline(opt *blockchain.NodeOperateOptions) error
}

type NewChallengingMonitorOptions struct {
	PrivateKey ecdsa.PrivateKey

	Blockchain   Blockchain
	ChallengeDB  ChallengeDB
	SliceStorage SliceStorage
}

// ChallengingMonitor's main work is to publish challenge requests if local node is dataOwner-node,
//  otherwise is to listen challenge requests and answer them in order to prove it's storing related files
type ChallengingMonitor struct {
	PrivateKey ecdsa.PrivateKey

	AnswerInterval  time.Duration
	RequestInterval time.Duration

	blockchain   Blockchain
	challengeDB  ChallengeDB
	sliceStorage SliceStorage

	doneLoopReqC chan struct{} //will be closed when LoopRequest breaks
	doneLoopAnsC chan struct{} //will be closed when LoopAnswer breaks
}

func New(conf *config.MonitorConf, opt *NewChallengingMonitorOptions) (*ChallengingMonitor, error) {
	requestInterval := DefaultRequestInterval
	answerInterval := defaultAnswerInterval

	logger.WithFields(logrus.Fields{
		"request-interval": requestInterval.String(),
		"answer-interval":  answerInterval.String(),
	}).Info("monitor initialize...")

	cm := &ChallengingMonitor{
		PrivateKey: opt.PrivateKey,

		RequestInterval: requestInterval,
		AnswerInterval:  answerInterval,

		blockchain:   opt.Blockchain,
		challengeDB:  opt.ChallengeDB,
		sliceStorage: opt.SliceStorage,
	}

	return cm, nil
}

// StartChallengeRequest starts to publish challenge request
func (c *ChallengingMonitor) StartChallengeRequest(ctx context.Context) {
	go c.loopRequest(ctx)
}

// StopChallengeRequest breaks loop
func (c *ChallengingMonitor) StopChallengeRequest() {
	if c.doneLoopReqC == nil {
		return
	}

	logger.Info("stops listening challenge request ...")

	select {
	case <-c.doneLoopReqC:
		return
	default:
	}

	<-c.doneLoopReqC
}

// StartChallengeAnswer starts to answer request
func (c *ChallengingMonitor) StartChallengeAnswer(ctx context.Context) {
	go c.loopAnswer(ctx)
}

// StopChallengeAnswer breaks loop
func (c *ChallengingMonitor) StopChallengeAnswer() {
	if c.doneLoopAnsC == nil {
		return
	}

	logger.Info("stops answering request ...")

	select {
	case <-c.doneLoopAnsC:
		return
	default:
	}

	<-c.doneLoopAnsC
}
