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
	"math/big"
	"reflect"
	"strconv"
	"testing"

	ml_common "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
	linear_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/linear_regression/gradient_descent/mpc_vertical"
	logic_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/logic_regression/mpc_vertical"

	pb_common "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

func TestPSIEncSetConvert(t *testing.T) {
	encSet := make(map[string]int)
	encSet["111"] = 1
	encSet["222"] = 2
	encSet["333"] = 3
	encSet["444"] = 4
	set := &linear_vertical.EncSet{
		EncIDs: encSet,
	}

	setBytes, err := PSIEncSetToBytes(set)
	checkErr(err, t)
	newSet, err := PSIEncSetFromBytes(setBytes)
	checkErr(err, t)

	if !reflect.DeepEqual(set, newSet) {
		t.Logf("set: %v\n", encSet)
		t.Logf("retrieved set: %v\n", newSet)
		t.Error("TestPSIEncSetConvert failed")
	}
}

func TestHomoPubkeyConvert(t *testing.T) {
	privkey, pubkeyBytes, err := GenerateHomoKeyPair()
	checkErr(err, t)

	pubkey, err := HomoPubkeyFromBytes(pubkeyBytes)
	checkErr(err, t)

	if !reflect.DeepEqual(&privkey.PublicKey, pubkey) {
		t.Logf("pubkey: %v\n", privkey.PublicKey)
		t.Logf("retrieved pubkey: %v\n", pubkey)
		t.Error("TestHomoPubkeyConvert failed")
	}
}

func TestLinearEncGradientPartConvert(t *testing.T) {
	encGradPart := make(map[int]*big.Int)
	encGradPart[1] = big.NewInt(11)
	encGradPart[2] = big.NewInt(22)
	encGradPart[3] = big.NewInt(33)

	encGradPartSquare := make(map[int]*big.Int)
	encGradPartSquare[1] = big.NewInt(111)
	encGradPartSquare[2] = big.NewInt(222)
	encGradPartSquare[3] = big.NewInt(333)
	encRegCost := big.NewInt(123)

	part := &linear_vertical.EncLocalGradientPart{
		EncGradPart:       encGradPart,
		EncGradPartSquare: encGradPartSquare,
		EncRegCost:        encRegCost,
	}

	partBytes, err := LinearEncGradientPartToBytes(part)
	checkErr(err, t)
	newPart, err := LinearEncGradientPartFromBytes(partBytes)
	checkErr(err, t)

	if !reflect.DeepEqual(part, newPart) {
		t.Logf("encGradPart: %v\n", part)
		t.Logf("retrieved encGradPart: %v\n", newPart)
		t.Error("TestLinearEncGradientPartConvert failed")
	}
}

func TestLogicEncGradAndCostPartConvert(t *testing.T) {
	encPart1 := make(map[int]*big.Int)
	encPart1[1] = big.NewInt(11)
	encPart1[2] = big.NewInt(22)
	encPart1[3] = big.NewInt(33)

	encPart2 := make(map[int]*big.Int)
	encPart2[1] = big.NewInt(111)
	encPart2[2] = big.NewInt(222)
	encPart2[3] = big.NewInt(333)

	encPart3 := make(map[int]*big.Int)
	encPart3[1] = big.NewInt(1111)
	encPart3[2] = big.NewInt(2222)
	encPart3[3] = big.NewInt(3333)

	encPart4 := make(map[int]*big.Int)
	encPart4[1] = big.NewInt(11111)
	encPart4[2] = big.NewInt(22222)
	encPart4[3] = big.NewInt(33333)

	encPart5 := make(map[int]*big.Int)
	encPart5[1] = big.NewInt(111111)
	encPart5[2] = big.NewInt(222222)
	encPart5[3] = big.NewInt(333333)
	encRegCost := big.NewInt(123)

	part := &logic_vertical.EncLocalGradAndCostPart{
		EncPart1:   encPart1,
		EncPart2:   encPart2,
		EncPart3:   encPart3,
		EncPart4:   encPart4,
		EncPart5:   encPart5,
		EncRegCost: encRegCost,
	}

	partBytes, err := LogicEncGradAndCostPartToBytes(part)
	checkErr(err, t)
	newPart, err := LogicEncGradAndCostPartFromBytes(partBytes)
	checkErr(err, t)

	if !reflect.DeepEqual(part, newPart) {
		t.Logf("encGradAndCostPart: %v\n", part)
		t.Logf("retrieved encGradAndCostPart: %v\n", newPart)
		t.Error("TestLogicEncGradAndCostPartConvert failed")
	}
}

func TestGradListConvert(t *testing.T) {
	gradMap1 := make(map[int]*big.Int)
	gradMap1[1] = big.NewInt(11)
	gradMap1[2] = big.NewInt(22)
	gradMap1[3] = big.NewInt(33)

	gradMap2 := make(map[int]*big.Int)
	gradMap2[1] = big.NewInt(111)
	gradMap2[2] = big.NewInt(222)
	gradMap2[3] = big.NewInt(333)

	gradMap3 := make(map[int]*big.Int)
	gradMap3[1] = big.NewInt(1111)
	gradMap3[2] = big.NewInt(2222)
	gradMap3[3] = big.NewInt(3333)

	gradList := []map[int]*big.Int{gradMap1, gradMap2, gradMap3}

	gradBytes, err := GradListToBytes(gradList)
	checkErr(err, t)
	newGradList, err := GradListFromBytes(gradBytes)
	checkErr(err, t)

	if !reflect.DeepEqual(gradList, newGradList) {
		t.Logf("gradList: %v\n", gradList)
		t.Logf("retrieved gradList: %v\n", newGradList)
		t.Error("TestGradListConvert failed")
	}
}

func TestCostConvert(t *testing.T) {
	cost := make(map[int]*big.Int)
	cost[1] = big.NewInt(111)
	cost[2] = big.NewInt(222)
	cost[3] = big.NewInt(333)
	cost[4] = big.NewInt(444)
	cost[5] = big.NewInt(555)

	costBytes, err := CostToBytes(cost)
	checkErr(err, t)
	newCost, err := CostFromBytes(costBytes)
	checkErr(err, t)

	if !reflect.DeepEqual(cost, newCost) {
		t.Logf("cost: %v\n", cost)
		t.Logf("retrieved cost: %v\n", newCost)
		t.Error("TestCostConvert failed")
	}
}

func TestTrainModelsConvert(t *testing.T) {
	thetas := []float64{0.1, 0.2}
	trainDataSet := &ml_common.TrainDataSet{
		FeatureNames: []string{"size", "floor"},
		XbarParams:   map[string]float64{"size": 111, "floor": 222},
		SigmaParams:  map[string]float64{"size": 11, "floor": 22},
	}
	params := pb_common.TrainParams{
		Label:     "price",
		RegMode:   0,
		RegParam:  0.0,
		Alpha:     0.1,
		Amplitude: 0.001,
		Accuracy:  10,
		IsTagPart: false,
	}
	modelsBytes, err := TrainModelsToBytes(thetas, trainDataSet, params)
	checkErr(err, t)

	newModels, err := TrainModelsFromBytes(modelsBytes)
	checkErr(err, t)

	thetaMap := thetasToMap(thetas, trainDataSet, params.IsTagPart)
	if newModels.Label != params.Label || newModels.IsTagPart != params.IsTagPart ||
		!reflect.DeepEqual(trainDataSet.XbarParams, newModels.Xbars) ||
		!reflect.DeepEqual(trainDataSet.SigmaParams, newModels.Sigmas) ||
		!reflect.DeepEqual(thetaMap, newModels.Thetas) {
		t.Errorf("TestTrainModelsConvert failed")
	}
}

func TestPredictResultConvert(t *testing.T) {
	idName := "testIDName"
	IDs := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}
	values := []float64{1.2, 2.4, 3.5, 4.2, 5.1, 6.4, 7.9, 8.5, 9.3, 10.2, 11.1}
	content, err := PredictResultToBytes(idName, IDs, values)
	checkErr(err, t)

	rows, err := PredictResultFromBytes(content)
	checkErr(err, t)
	if rows[0][0] != idName || rows[0][1] != "value" {
		t.Errorf("first row dis-matched, supposed to be [id, value], got %v\n", rows[0])
	}
	for i := 1; i < len(rows); i++ {
		value := strconv.FormatFloat(values[i-1], 'g', -1, 64)
		if rows[i][0] != IDs[i-1] || rows[i][1] != value {
			t.Errorf("row-%d dis-matched, supposed to be [%s, %s], got %v\n", i, IDs[i-1], value, rows[i])
		}
	}
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
