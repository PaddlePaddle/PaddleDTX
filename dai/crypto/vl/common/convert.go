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
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strconv"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	ml_common "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
	linear_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/linear_regression/gradient_descent/mpc_vertical"
	logic_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/logic_regression/mpc_vertical"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	pb_common "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

// PSIEncSetToBytes convert enc set to bytes
func PSIEncSetToBytes(set *linear_vertical.EncSet) ([]byte, error) {
	encSet := make(map[string]int)
	for key, value := range set.EncIDs {
		newKey := hex.EncodeToString([]byte(key))
		encSet[newKey] = value
	}
	return json.Marshal(encSet)
}

// PSIEncSetFromBytes retrieve enc set from bytes
func PSIEncSetFromBytes(setBytes []byte) (*linear_vertical.EncSet, error) {
	var set map[string]int
	err := json.Unmarshal(setBytes, &set)
	if err != nil {
		return nil, err
	}

	originalSet := make(map[string]int)
	for key, value := range set {
		newKey, err := hex.DecodeString(key)
		if err != nil {
			return nil, err
		}
		originalSet[string(newKey)] = value
	}

	encSet := &linear_vertical.EncSet{
		EncIDs: originalSet,
	}
	return encSet, nil
}

// HomoPubkeyToBytes convert homomorphic public key to bytes
func HomoPubkeyToBytes(key *paillier.PublicKey) ([]byte, error) {
	return json.Marshal(key)
}

// HomoPubkeyFromBytes retrieve homomorphic public key from bytes
func HomoPubkeyFromBytes(keyBytes []byte) (*paillier.PublicKey, error) {
	var key paillier.PublicKey
	err := json.Unmarshal(keyBytes, &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// LinearEncGradientPartToBytes convert enc gradient part to bytes
func LinearEncGradientPartToBytes(encPart *linear_vertical.EncLocalGradientPart) ([]byte, error) {
	return json.Marshal(encPart)
}

// LinearEncGradientPartFromBytes retrieve enc gradient part from bytes
func LinearEncGradientPartFromBytes(encPartBytes []byte) (*linear_vertical.EncLocalGradientPart, error) {
	var encPart linear_vertical.EncLocalGradientPart
	err := json.Unmarshal(encPartBytes, &encPart)
	if err != nil {
		return nil, err
	}
	return &encPart, nil
}

// LogicEncGradAndCostPartToBytes convert enc gradientAndCost part to bytes
func LogicEncGradAndCostPartToBytes(encPart *logic_vertical.EncLocalGradAndCostPart) ([]byte, error) {
	return json.Marshal(encPart)
}

// LogicEncGradAndCostPartFromBytes retrieve enc gradientAndCost part from bytes
func LogicEncGradAndCostPartFromBytes(encPartBytes []byte) (*logic_vertical.EncLocalGradAndCostPart, error) {
	var encPart logic_vertical.EncLocalGradAndCostPart
	err := json.Unmarshal(encPartBytes, &encPart)
	if err != nil {
		return nil, err
	}
	return &encPart, nil
}

// GradListToBytes convert gradient list to bytes
func GradListToBytes(grads []map[int]*big.Int) ([]byte, error) {
	return json.Marshal(grads)
}

// GradListFromBytes retrieve gradient list from bytes
func GradListFromBytes(gradsBytes []byte) ([]map[int]*big.Int, error) {
	var grads []map[int]*big.Int
	err := json.Unmarshal(gradsBytes, &grads)
	if err != nil {
		return nil, err
	}
	return grads, nil
}

// CostToBytes convert costs to bytes
func CostToBytes(encCost map[int]*big.Int) ([]byte, error) {
	return json.Marshal(encCost)
}

// CostFromBytes retrieve costs from bytes
func CostFromBytes(encCostByes []byte) (map[int]*big.Int, error) {
	encCost := make(map[int]*big.Int)
	err := json.Unmarshal(encCostByes, &encCost)
	if err != nil {
		return nil, err
	}
	return encCost, nil
}

// TrainModelsToBytes convert train models to bytes for transfer and save
func TrainModelsToBytes(thetas []float64, trainDataSet *ml_common.TrainDataSet, params pb_common.TrainParams) ([]byte, error) {
	thetaMap := thetasToMap(thetas, trainDataSet, params.IsTagPart)
	trainModels := pb_common.TrainModels{
		Thetas:    thetaMap,
		Xbars:     trainDataSet.XbarParams,
		Sigmas:    trainDataSet.SigmaParams,
		Label:     params.Label,
		IsTagPart: params.IsTagPart,
	}
	return json.Marshal(trainModels)
}

// thetasToMap save train model as map
func thetasToMap(thetas []float64, trainDataSet *ml_common.TrainDataSet, isTagPart bool) map[string]float64 {
	params := make(map[string]float64)
	if isTagPart {
		params["Intercept"] = thetas[0]
		for i := 0; i < len(trainDataSet.FeatureNames)-1; i++ {
			params[trainDataSet.FeatureNames[i]] = thetas[i+1]
		}
	} else {
		for i := 0; i < len(trainDataSet.FeatureNames); i++ {
			params[trainDataSet.FeatureNames[i]] = thetas[i]
		}
	}
	return params
}

// TrainModelsFromBytes retrieve train models from bytes
func TrainModelsFromBytes(modelsBytes []byte) (*pb_common.TrainModels, error) {
	var model pb_common.TrainModels
	err := json.Unmarshal(modelsBytes, &model)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// PredictResultToBytes convert ID and predict values to bytes for storage
func PredictResultToBytes(idName string, IDs []string, values []float64) ([]byte, error) {
	if len(IDs) != len(values) {
		return nil, errorx.New(errcodes.ErrCodeParam, "ID and predict values numbers are not equal")
	}

	fileRows := make([][]string, len(IDs)+1)
	// first row [idName, value], others are predict result
	fileRows[0] = []string{idName, "value"}
	for i := 0; i < len(IDs); i++ {
		fileRows[i+1] = []string{IDs[i], strconv.FormatFloat(values[i], 'g', -1, 64)}
	}

	content, err := json.Marshal(fileRows)
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeEncoding, "encode predict results failed: %s", err.Error())
	}
	return content, nil
}

// PredictResultFromBytes retrieve predict values from bytes
func PredictResultFromBytes(resultBytes []byte) ([][]string, error) {
	if len(resultBytes) == 0 {
		return nil, nil
	}

	var result [][]string
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, errorx.New(errcodes.ErrCodeEncoding, "decode predict results failed: %s", err.Error())
	}
	return result, nil
}
