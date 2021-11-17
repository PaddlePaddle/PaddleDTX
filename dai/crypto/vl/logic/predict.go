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
	"fmt"
	"math"
	"strconv"

	pb_common "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

// PredictLocalPart calculate predict values for local part
// fileRows is sample rows, first row is feature list, others are values for each sample
func PredictLocalPart(fileRows [][]string, params *pb_common.TrainModels) ([]float64, error) {
	featureList := fileRows[0]

	var localPredictValues []float64
	for i := 1; i < len(fileRows); i++ {
		input := make(map[string]float64)
		for j := 0; j < len(featureList); j++ {
			featureName := featureList[j]
			value, err := strconv.ParseFloat(fileRows[i][j], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value, err: %v", err)
			}
			input[featureName] = value
		}

		var predictValue float64
		standardizedInput := xchainCryptoClient.LogRegVLStandardizeLocalInput(params.Xbars, params.Sigmas, input)
		if params.IsTagPart {
			predictValue = xchainCryptoClient.LogRegVLPredictLocalTagPart(params.Thetas, standardizedInput)
		} else {
			predictValue = xchainCryptoClient.LogRegVLPredictLocalPart(params.Thetas, standardizedInput)
		}

		localPredictValues = append(localPredictValues, predictValue)
	}

	return localPredictValues, nil
}

// CalRealPredictValue calculate final predict value by sum of predict parts
func CalRealPredictValue(localPredict, otherPredict []float64) []float64 {
	var realPredictValue []float64

	for i := 0; i < len(localPredict); i++ {
		predictSum := localPredict[i] + otherPredict[i]
		realValue := 1 / (1 + math.Exp(-1*predictSum))
		realPredictValue = append(realPredictValue, realValue)
	}
	return realPredictValue
}
