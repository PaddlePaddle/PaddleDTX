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

package logic

import (
	"math"
	"math/big"

	"github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	ml_common "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
	logic_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/logic_regression/mpc_vertical"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	pb_common "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

var xchainCryptoClient = new(xchain.XchainCryptoClient)

// GetTrainDataSetFromFile retrieve train dataset from file for tag/no-tag part
// fileRows is sample rows, first row is feature list, others are values for each sample
// params includes all required parameters for training
func GetTrainDataSetFromFile(fileRows [][]string, params pb_common.TrainParams) (*ml_common.TrainDataSet, error) {
	features, err := xchainCryptoClient.LogRegImportFeatures(fileRows, params.Label, params.LabelName)
	if err != nil {
		return nil, err
	}

	dataSet := &ml_common.DataSet{
		Features: features,
	}
	standardizedData := xchainCryptoClient.LogRegVLStandardizeDataSet(dataSet, params.Label)

	if params.IsTagPart {
		return xchainCryptoClient.LogRegVLPreProcessDataSetTagPart(standardizedData, params.Label), nil
	}
	return xchainCryptoClient.LogRegVLPreProcessDataSet(standardizedData), nil
}

// InitThetas initialize model for tag/no-tag part
func InitThetas(trainSet *ml_common.TrainDataSet, params pb_common.TrainParams) []float64 {
	if params.IsTagPart {
		thetas := make([]float64, len(trainSet.TrainSet[0])-2)
		return thetas
	}
	thetas := make([]float64, len(trainSet.TrainSet[0])-1)
	return thetas
}

// CalLocalGradientAndCost calculate local gradient and cost part for tag/no-tag part
// trainSet is local train set, including features list and sample values
// publicKey is local public key for encrypting rawPart to encPart
// rawPart for local calculation for gradient and cost later, encPart for transfer
func CalLocalGradientAndCost(trainSet *ml_common.TrainDataSet, thetas []float64, params pb_common.TrainParams,
	publicKey *paillier.PublicKey, round int) (*logic_vertical.RawLocalGradAndCostPart, []byte, [][]float64, error) {

	// BGD(Batch Gradient Descent), SGD(Stochastic Gradient Descent) or MBGD(Mini-Batch Gradient Descent)
	trainSetThisRound, newSet := vl_common.GetBatchSetBySize(trainSet.TrainSet, params, round, true)

	var gradAndCostPart *logic_vertical.LocalGradAndCostPart
	var err error
	if params.IsTagPart {
		gradAndCostPart, err = xchainCryptoClient.LogRegVLCalLocalGradAndCostTagPart(thetas, trainSetThisRound, int(params.Accuracy), int(params.RegMode), params.RegParam, publicKey)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		gradAndCostPart, err = xchainCryptoClient.LogRegVLCalLocalGradAndCost(thetas, trainSetThisRound, int(params.Accuracy), int(params.RegMode), params.RegParam, publicKey)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	encPartBytes, err := vl_common.LogicEncGradAndCostPartToBytes(gradAndCostPart.EncPart)
	if err != nil {
		return nil, nil, nil, err
	}
	return gradAndCostPart.RawPart, encPartBytes, newSet, nil
}

// CalEncGradientAndCost calculate own encrypted gradient and cost, encrypted by other part's public key
// rawPart is calculated locally, otherPartBytes is received from other party
// publicKeyBytes is homomorphic public key bytes received from other party, used for encryption
// thetas is calculated in last round
// return encGradient, encCost, gradient noise, cost noise
// encGradient and encCost are mixed by noise, transferred to other party
// gradient noise and cost noise are plaintext, used to recover real gradient and cost
func CalEncGradientAndCost(rawPart *logic_vertical.RawLocalGradAndCostPart, otherPartBytes []byte, trainSet *ml_common.TrainDataSet,
	params pb_common.TrainParams, publicKeyBytes []byte, thetas []float64, round int) ([]byte, []byte, []*big.Int, *big.Int, error) {

	otherEncPart, err := vl_common.LogicEncGradAndCostPartFromBytes(otherPartBytes)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	publicKey, err := vl_common.HomoPubkeyFromBytes(publicKeyBytes)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// BGD, SGD or MBGD
	trainSetThisRound, _ := vl_common.GetBatchSetBySize(trainSet.TrainSet, params, round, false)

	var encGradList []map[int]*big.Int
	var encGrad *ml_common.EncLocalGradient
	var gradientNoise []*big.Int
	for i := 0; i < len(thetas); i++ {
		if params.IsTagPart {
			encGrad, err = xchainCryptoClient.LogRegVLCalEncGradientTagPart(rawPart, otherEncPart, trainSetThisRound, i, int(params.Accuracy), publicKey)
		} else {
			encGrad, err = xchainCryptoClient.LogRegVLCalEncGradient(rawPart, otherEncPart, trainSetThisRound, i, int(params.Accuracy), publicKey)
		}
		if err != nil {
			return nil, nil, nil, nil, err
		}

		encGradList = append(encGradList, encGrad.EncGrad)
		gradientNoise = append(gradientNoise, encGrad.RandomNoise)
	}

	var encCost *ml_common.EncLocalCost
	if params.IsTagPart {
		encCost, err = xchainCryptoClient.LogRegVLEvaluateEncCostTagPart(rawPart, otherEncPart, trainSetThisRound, int(params.Accuracy), publicKey)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	} else {
		encCost, err = xchainCryptoClient.LogRegVLEvaluateEncCost(rawPart, otherEncPart, trainSetThisRound, int(params.Accuracy), publicKey)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	encGradListBytes, err := vl_common.GradListToBytes(encGradList)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	encCostBytes, err := vl_common.CostToBytes(encCost.EncCost)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	costNoise := encCost.RandomNoise

	return encGradListBytes, encCostBytes, gradientNoise, costNoise, nil
}

// DecGradientAndCost decrypt gradient list and cost for other part
// encGradsBytes and encCostBytes are ciphertext received from other party, encrypted by local homomorphic public key
// privateKey is local homomorphic private key, used to decrypt encGradsBytes and encCostBytes
func DecGradientAndCost(encGradsBytes []byte, encCostBytes []byte, privateKey *paillier.PrivateKey) ([]byte, []byte, error) {
	encGrads, err := vl_common.GradListFromBytes(encGradsBytes)
	if err != nil {
		return nil, nil, err
	}

	var decGradList []map[int]*big.Int
	for i := 0; i < len(encGrads); i++ {
		grad := xchainCryptoClient.LogRegVLDecryptGradient(encGrads[i], privateKey)
		decGradList = append(decGradList, grad)
	}

	encCost, err := vl_common.CostFromBytes(encCostBytes)
	if err != nil {
		return nil, nil, err
	}
	cost := xchainCryptoClient.LogRegVLDecryptCost(encCost, privateKey)

	decGradBytes, err := vl_common.GradListToBytes(decGradList)
	if err != nil {
		return nil, nil, err
	}
	decCostBytes, err := vl_common.CostToBytes(cost)
	if err != nil {
		return nil, nil, err
	}

	return decGradBytes, decCostBytes, nil
}

// UpdateCost retrieve and update cost
// decCostBytes is decrypted cost received from other party, with costNoise
func UpdateCost(decCostBytes []byte, costNoise *big.Int, params pb_common.TrainParams) (float64, error) {
	// retrieve real cost
	costMap, err := vl_common.CostFromBytes(decCostBytes)
	if err != nil {
		return 0, err
	}
	realCost := xchainCryptoClient.LogRegVLRetrieveRealCost(costMap, int(params.Accuracy), costNoise)
	cost := xchainCryptoClient.LogRegVLCalCost(realCost)

	return cost, nil
}

// UpdateGradient retrieve and update thetas
// decGradBytes is decrypted gradient received from other party, with gradientNoise
func UpdateGradient(decGradBytes []byte, gradientNoise []*big.Int, thetas []float64, params pb_common.TrainParams) ([]float64, error) {
	grads, err := vl_common.GradListFromBytes(decGradBytes)
	if err != nil {
		return nil, err
	}

	newThetas := make([]float64, len(thetas))
	copy(newThetas[0:], thetas)

	for i := 0; i < len(newThetas); i++ {
		realGradient := xchainCryptoClient.LogRegVLRetrieveRealGradient(grads[i], int(params.Accuracy), gradientNoise[i])
		//	grad := xchainCryptoClient.LogRegVLCalGradient(realGradient)
		grad := xchainCryptoClient.LogRegVLCalGradientWithReg(thetas, realGradient, i, int(params.RegMode), params.RegParam)
		newThetas[i] = newThetas[i] - params.Alpha*grad
	}

	return newThetas, nil
}

// StopTraining determine if train process should be stopped
func StopTraining(lastCost, cost float64, params pb_common.TrainParams) bool {
	return math.Abs(cost-lastCost) < params.Amplitude
}
