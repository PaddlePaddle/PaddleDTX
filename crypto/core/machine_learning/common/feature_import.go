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
	"fmt"
	"strconv"
)

// ImportFeaturesForLinReg import linear regression features from file
func ImportFeaturesForLinReg(fileRows [][]string) ([]*DataFeature, error) {
	if fileRows == nil {
		return nil, fmt.Errorf("empty file content")
	}

	// read the first row to get all features
	featureNum := len(fileRows[0])
	features := make([]*DataFeature, featureNum)
	for i := 0; i < featureNum; i++ {
		features[i] = new(DataFeature)
		features[i].Sets = make(map[int]float64)
		features[i].FeatureName = fileRows[0][i]
	}

	// read from all rows to get feature values
	sample := 0
	for row := 1; row < len(fileRows); row++ {
		for i := 0; i < featureNum; i++ {
			value, err := strconv.ParseFloat(fileRows[row][i], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value, err: %v", err)
			}
			features[i].Sets[sample] = value
		}
		sample++
	}

	return features, nil
}

// ImportFeaturesForLogReg import logic regression features from file, target variable imported as 1 or 0
// - fileRows file rows, first row is feature list
// - label target feature
// - labelName target variable
func ImportFeaturesForLogReg(fileRows [][]string, label, labelName string) ([]*DataFeature, error) {
	if fileRows == nil {
		return nil, fmt.Errorf("empty file content")
	}

	// read the first row to get all features
	featureNum := len(fileRows[0])
	features := make([]*DataFeature, featureNum)
	for i := 0; i < featureNum; i++ {
		features[i] = new(DataFeature)
		features[i].Sets = make(map[int]float64)
		features[i].FeatureName = fileRows[0][i]
	}

	// read from all rows to get feature values
	sample := 0
	for row := 1; row < len(fileRows); row++ {
		for i := 0; i < featureNum; i++ {
			if features[i].FeatureName == label {
				// parse target feature variable to 0 or 1
				if fileRows[row][i] == labelName {
					features[i].Sets[sample] = 1.0
				} else {
					features[i].Sets[sample] = 0.0
				}
			} else {
				value, err := strconv.ParseFloat(fileRows[row][i], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse value, err: %v", err)
				}
				features[i].Sets[sample] = value
			}
		}

		sample++
	}

	return features, nil
}
