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

package linear_reg_vl

import (
	"math/big"
	"sync"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	mlCom "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
	linearVert "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/linear_regression/gradient_descent/mpc_vertical"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	vlCom "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/linear"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

type process struct {
	round    uint64
	homoPriv *paillier.PrivateKey // homomorphic private key for parameter encryption/decryption
	params   *pbCom.TrainParams
	fileRows [][]string

	trainDataSet   *mlCom.TrainDataSet
	homoPubOfOther []byte // public key of other part

	mutex sync.Mutex

	// intermediate results
	// linear.train for more
	cost, lastCost                        float64
	thetas, nextThetas                    []float64
	rawPart                               *linearVert.RawLocalGradientPart
	partBytesForOther, partBytesFromOther []byte
	encGradForOther, encGradFromOther     []byte
	encCostForOther, encCostFromOther     []byte
	gradientNoise                         []*big.Int
	costNoise                             *big.Int
	gradBytesForOther, gradBytesFromOther []byte
	costBytesForOther, costBytesFromOther []byte

	stopped      int8 // 0 means not decided, 1 means received `Stopped`, 2 means received `NotStopped`
	otherStopped int8 // 0 means not received decision, 1 means received `Stopped`, 2 means received `NotStopped`

	calLocalGradientAndCostTimes        int
	calEncGradientAndCostTimes          int
	setEncGradientAndCostFromOtherTimes int
	decGradientAndCostTimes             int
	setGradientAndCostFromOtherTimes    int

	// cache intermediate result for next round
	partBytesFromOtherNextRound []byte
}

// init initialize Process, after PSI, before training
func (p *process) init(fileRows [][]string) error {
	// fileRows
	p.fileRows = fileRows

	// trainset
	trainDataSet, err := linear.GetTrainDataSetFromFile(p.fileRows, *p.params)

	if err != nil {
		return errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when linear_reg_vl GetTrainDataSetFromFile", err.Error())
	}

	p.trainDataSet = trainDataSet

	// thetas
	thetas := linear.InitThetas(trainDataSet, *p.params)
	p.thetas = thetas

	return nil
}

func (p *process) upRound(newRound uint64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if newRound == p.round {
		return nil
	}

	if newRound != p.round+1 {
		return errorx.New(errcodes.ErrCodeParam, "target round %d dismatch process.round %d", newRound, p.round)
	}

	p.round++

	p.lastCost = p.cost
	p.thetas = p.nextThetas
	p.cost = 0
	p.nextThetas = []float64{}

	// clean intermediate results
	p.rawPart = nil
	p.partBytesForOther = []byte{}

	if len(p.partBytesFromOtherNextRound) == 0 {
		p.partBytesFromOther = []byte{}
	} else {
		p.partBytesFromOther = p.partBytesFromOtherNextRound
		p.partBytesFromOtherNextRound = []byte{}
	}

	p.encGradForOther = []byte{}
	p.encGradFromOther = []byte{}
	p.encCostForOther = []byte{}
	p.encCostFromOther = []byte{}

	p.gradientNoise = []*big.Int{}
	p.costNoise = nil

	p.gradBytesForOther = []byte{}
	p.gradBytesFromOther = []byte{}
	p.costBytesForOther = []byte{}
	p.costBytesFromOther = []byte{}

	p.stopped = 0
	p.otherStopped = 0

	p.calLocalGradientAndCostTimes = 0
	p.calEncGradientAndCostTimes = 0
	p.setEncGradientAndCostFromOtherTimes = 0
	p.decGradientAndCostTimes = 0
	p.setGradientAndCostFromOtherTimes = 0

	return nil
}

func (p *process) calLocalGradientAndCost() ([]byte, int, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.calLocalGradientAndCostTimes != 0 {
		p.calLocalGradientAndCostTimes++
		return p.partBytesForOther, p.calLocalGradientAndCostTimes, nil
	}

	rawPart, otherPartBytes, newSet, err := linear.CalLocalGradientAndCost(p.trainDataSet, p.thetas, *p.params, &p.homoPriv.PublicKey, int(p.round))
	if err != nil {
		return []byte{}, p.calLocalGradientAndCostTimes, errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when linear_reg_vl calLocalGradientAndCost", err.Error())
	}

	p.trainDataSet.TrainSet = newSet

	p.rawPart = rawPart
	p.partBytesForOther = otherPartBytes

	p.calLocalGradientAndCostTimes++

	return p.partBytesForOther, p.calLocalGradientAndCostTimes, nil
}

func (p *process) setPartBytesFromOther(partBytesFromOther []byte, round uint64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if partBytesFromOther == nil {
		return errorx.New(errcodes.ErrCodeParam, "partBytesFromOther nil %d", p.round)
	}

	if round == p.round {
		p.partBytesFromOther = partBytesFromOther
	} else if round == p.round+1 {
		p.partBytesFromOtherNextRound = partBytesFromOther
	} else {
		return errorx.New(errcodes.ErrCodeParam, "target round [%d] should 1 greater or equal than process.round %d", round, p.round)
	}

	return nil
}

func (p *process) calEncGradientAndCost() ([]byte, []byte, int, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.calEncGradientAndCostTimes != 0 {
		p.calEncGradientAndCostTimes++
		return p.encGradForOther, p.encCostForOther, p.calEncGradientAndCostTimes, nil
	}

	if len(p.partBytesFromOther) == 0 || p.rawPart == nil {
		return []byte{}, []byte{}, p.calEncGradientAndCostTimes, nil
	}

	encGradForOther, encCostForOther, gradientNoise, costNoise, err := linear.CalEncGradientAndCost(p.rawPart, p.partBytesFromOther, p.trainDataSet, *p.params, p.homoPubOfOther, p.thetas, int(p.round))
	if err != nil {
		return []byte{}, []byte{}, p.calEncGradientAndCostTimes, errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when linear_reg_vl calEncGradientAndCost", err.Error())
	}

	p.encGradForOther = encGradForOther
	p.encCostForOther = encCostForOther
	p.gradientNoise = gradientNoise
	p.costNoise = costNoise

	p.calEncGradientAndCostTimes++

	return p.encGradForOther, p.encCostForOther, p.calEncGradientAndCostTimes, nil
}

func (p *process) setEncGradientAndCostFromOther(encGradFromOther, encCostFromOther []byte) int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.setEncGradientAndCostFromOtherTimes != 0 {
		p.setEncGradientAndCostFromOtherTimes++
		return p.setEncGradientAndCostFromOtherTimes
	}

	p.encGradFromOther = encGradFromOther
	p.encCostFromOther = encCostFromOther

	p.setEncGradientAndCostFromOtherTimes++
	return p.setEncGradientAndCostFromOtherTimes
}

func (p *process) decGradientAndCost() ([]byte, []byte, int, error) {

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.decGradientAndCostTimes != 0 {
		p.decGradientAndCostTimes++
		return p.gradBytesForOther, p.costBytesForOther, p.decGradientAndCostTimes, nil
	}

	gradBytesForOther, costBytesForOther, err := linear.DecGradientAndCost(p.encGradFromOther, p.encCostFromOther, p.homoPriv)
	if err != nil {
		return []byte{}, []byte{}, p.decGradientAndCostTimes, errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when linear_reg_vl decGradientAndCost", err.Error())
	}

	p.gradBytesForOther = gradBytesForOther
	p.costBytesForOther = costBytesForOther

	p.decGradientAndCostTimes++

	return p.gradBytesForOther, p.costBytesForOther, p.decGradientAndCostTimes, nil
}

func (p *process) SetGradientAndCostFromOther(gradBytesFromOther, costBytesFromOther []byte) int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.setGradientAndCostFromOtherTimes != 0 {
		p.setGradientAndCostFromOtherTimes++
		return p.setGradientAndCostFromOtherTimes
	}
	p.gradBytesFromOther = gradBytesFromOther
	p.costBytesFromOther = costBytesFromOther

	p.setGradientAndCostFromOtherTimes++

	return p.setGradientAndCostFromOtherTimes
}

func (p *process) updateCostAndGradient() (bool, error) {

	var stopped bool

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.stopped != 0 {
		if p.stopped == 1 {
			stopped = true
		}
		return stopped, nil
	}

	if len(p.gradBytesFromOther) == 0 || len(p.gradientNoise) == 0 {
		logger.Panicf("gradBytesFromOther is [%v], gradientNoise is [%v], thetas is [%v], round [%v]", p.gradBytesFromOther, p.gradientNoise, p.thetas, p.round)
	}

	nextThetas, err := linear.UpdateGradient(p.gradBytesFromOther, p.gradientNoise, p.thetas, *p.params)
	if err != nil {
		return stopped, errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when linear_reg_vl updateGradient", err.Error())
	}
	p.nextThetas = nextThetas

	if p.round > 0 {
		cost, err := linear.UpdateCost(p.costBytesFromOther, p.costNoise, *p.params)
		if err != nil {
			return stopped, errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when linear_reg_vl updateCost", err.Error())
		}

		p.cost = cost
		stopped = linear.StopTraining(p.lastCost, p.cost, *p.params)
	}
	if stopped {
		p.stopped = 1
	} else {
		p.stopped = 2
	}

	logger.Infof("updateCostAndGradient lastCost %v, cost %v, round %d", p.lastCost, p.cost, p.round)

	return stopped, nil
}

func (p *process) setOtherStatus(otherStopped bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.otherStopped != 0 {
		return
	}

	if otherStopped {
		p.otherStopped = 1
	} else {
		p.otherStopped = 2
	}
}

func (p *process) stop() (decided bool, stopped bool) {
	if p.otherStopped != 0 && p.stopped != 0 {
		logger.Infof("stop or not, otherStopped %d, stopped %d, round %d", p.otherStopped, p.stopped, p.round)
		decided = true
	}

	if decided {
		if p.otherStopped == 1 && p.stopped == 1 {
			stopped = true
		}
	}

	return
}

func (p *process) getTrainModels() ([]byte, error) {
	modelBytes, err := vlCom.TrainModelsToBytes(p.thetas, p.trainDataSet, *p.params)
	if err != nil {
		return []byte{}, errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when linear_reg_vl trainModelsToBytes", err.Error())
	}

	return modelBytes, nil
}

func (p *process) setHomoPubOfOther(homoPubOfOther []byte) {
	p.homoPubOfOther = homoPubOfOther
}

func newProcess(homoPriv *paillier.PrivateKey, params *pbCom.TrainParams) *process {
	return &process{
		round:    0,
		params:   params,
		homoPriv: homoPriv,
	}
}
