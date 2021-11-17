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

package linear

import (
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"testing"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	ml_common "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
	linear_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/linear_regression/gradient_descent/mpc_vertical"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common/csv"
	pb_common "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

var (
	fileRowsA [][]string
	fileRowsB [][]string

	trainDataSetA *ml_common.TrainDataSet
	trainDataSetB *ml_common.TrainDataSet
	newSetA       [][]float64
	newSetB       [][]float64

	homoPrivA *paillier.PrivateKey
	homoPrivB *paillier.PrivateKey

	rawPartA        *linear_vertical.RawLocalGradientPart
	rawPartB        *linear_vertical.RawLocalGradientPart
	otherPartBytesA []byte
	otherPartBytesB []byte

	paramsA pb_common.TrainParams
	paramsB pb_common.TrainParams

	thetasA []float64
	thetasB []float64

	encGradA []byte
	encCostA []byte
	encGradB []byte
	encCostB []byte

	gradientNoiseA []*big.Int
	gradientNoiseB []*big.Int
	costNoiseA     *big.Int
	costNoiseB     *big.Int

	lastCostA float64
	lastCostB float64
	costA     float64
	costB     float64
)

func TestLinearReg(t *testing.T) {
	// train test
	trainFileA := "../testdata/linear_boston_housing/train_dataA.csv"
	trainFileB := "../testdata/linear_boston_housing/train_dataB.csv"

	fileContentA, err := ioutil.ReadFile(trainFileA)
	checkErr(err, t)
	fileContentB, err := ioutil.ReadFile(trainFileB)
	checkErr(err, t)

	var homoPubA, homoPubB []byte
	homoPrivA, homoPubA, err = vl_common.GenerateHomoKeyPair()
	checkErr(err, t)
	homoPrivB, homoPubB, err = vl_common.GenerateHomoKeyPair()
	checkErr(err, t)

	fileRowsA, err = csv.ReadRowsFromFile(fileContentA)
	checkErr(err, t)
	fileRowsB, err = csv.ReadRowsFromFile(fileContentB)
	checkErr(err, t)

	paramsA = pb_common.TrainParams{
		Label:     "MEDV",
		RegMode:   0,
		RegParam:  0.1,
		Alpha:     0.1,
		Amplitude: 0.0001,
		Accuracy:  10,
		IsTagPart: false,
		BatchSize: 4,
	}
	paramsB = pb_common.TrainParams{
		Label:     "MEDV",
		RegMode:   0,
		RegParam:  0.1,
		Alpha:     0.1,
		Amplitude: 0.0001,
		Accuracy:  10,
		IsTagPart: true,
		BatchSize: 4,
	}

	trainDataSetA, err = GetTrainDataSetFromFile(fileRowsA, paramsA)
	checkErr(err, t)
	trainDataSetB, err = GetTrainDataSetFromFile(fileRowsB, paramsB)
	checkErr(err, t)

	thetasA = InitThetas(trainDataSetA, paramsA)
	thetasB = InitThetas(trainDataSetB, paramsB)

	round := 0
	for {
		rawPartA, otherPartBytesA, newSetA, err = CalLocalGradientAndCost(trainDataSetA, thetasA, paramsA, &homoPrivA.PublicKey, round)
		checkErr(err, t)
		rawPartB, otherPartBytesB, newSetB, err = CalLocalGradientAndCost(trainDataSetB, thetasB, paramsB, &homoPrivB.PublicKey, round)
		checkErr(err, t)
		trainDataSetA.TrainSet = newSetA
		trainDataSetB.TrainSet = newSetB

		encGradA, encCostA, gradientNoiseA, costNoiseA, err = CalEncGradientAndCost(rawPartA, otherPartBytesB, trainDataSetA, paramsA, homoPubB, thetasA, round)
		checkErr(err, t)
		encGradB, encCostB, gradientNoiseB, costNoiseB, err = CalEncGradientAndCost(rawPartB, otherPartBytesA, trainDataSetB, paramsB, homoPubA, thetasB, round)
		checkErr(err, t)

		gradBytesA, costBytesA, err := DecGradientAndCost(encGradA, encCostA, homoPrivB)
		checkErr(err, t)
		gradBytesB, costBytesB, err := DecGradientAndCost(encGradB, encCostB, homoPrivA)
		checkErr(err, t)

		if round != 0 {
			costA, err = UpdateCost(costBytesA, costNoiseA, paramsA)
			checkErr(err, t)
			costB, err = UpdateCost(costBytesB, costNoiseB, paramsB)
			checkErr(err, t)
			if StopTraining(lastCostA, costA, paramsA) && StopTraining(lastCostB, costB, paramsB) {
				break
			}

			log.Printf("round[%d], deltaA: %v, deltaB: %v", round, math.Abs(costA-lastCostA), math.Abs(costB-lastCostB))
		}

		thetasA, err = UpdateGradient(gradBytesA, gradientNoiseA, thetasA, paramsA)
		checkErr(err, t)
		thetasB, err = UpdateGradient(gradBytesB, gradientNoiseB, thetasB, paramsB)
		checkErr(err, t)

		lastCostA = costA
		lastCostB = costB
		round++
	}

	modelBytesA, err := vl_common.TrainModelsToBytes(thetasA, trainDataSetA, paramsA)
	checkErr(err, t)
	modelBytesB, err := vl_common.TrainModelsToBytes(thetasB, trainDataSetB, paramsB)
	checkErr(err, t)

	t.Logf("model A: %s\n", modelBytesA)
	t.Logf("model B: %s\n", modelBytesB)

	// predict test
	predictFileA := "../testdata/linear_boston_housing/predict_dataA.csv"
	predictFileB := "../testdata/linear_boston_housing/predict_dataB.csv"

	fileContentA, err = ioutil.ReadFile(predictFileA)
	checkErr(err, t)
	fileContentB, err = ioutil.ReadFile(predictFileB)
	checkErr(err, t)

	fileRowsA, err = csv.ReadRowsFromFile(fileContentA)
	checkErr(err, t)
	fileRowsB, err = csv.ReadRowsFromFile(fileContentB)
	checkErr(err, t)

	modelA, err := vl_common.TrainModelsFromBytes(modelBytesA)
	checkErr(err, t)
	modelB, err := vl_common.TrainModelsFromBytes(modelBytesB)
	checkErr(err, t)

	predictA, err := PredictLocalPart(fileRowsA, modelA)
	checkErr(err, t)
	predictB, err := PredictLocalPart(fileRowsB, modelB)
	checkErr(err, t)

	output := DeStandardizeOutput(modelB, predictA, predictB)
	t.Logf("predict value: %v\n", output)

	// calculate r_squared
	rss := 0.0
	tss := 0.0
	for i := 0; i < len(output); i++ {
		predictValue := output[i]

		featureNum := len(trainDataSetB.FeatureNames)
		realValue := trainDataSetB.OriginalTrainSet[i][featureNum+1]
		rss += math.Pow(realValue-predictValue, 2)
		tss += math.Pow(realValue-modelB.Xbars["price"], 2)
	}
	rSquared := 1 - rss/tss
	t.Logf("r_squared: %f\n", rSquared)
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
