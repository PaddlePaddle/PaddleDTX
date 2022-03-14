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

package metrics

// Implements several utility functions to measure regression.

import (
	"errors"
	"math"
)

// GetMSE returns Mean Squared Error which measures the average of the squares of the errors,
// that is, the average squared difference between the estimated values and the actual value.
func GetMSE(yReal []float64, yPred []float64) (float64, error) {
	cr := len(yReal)
	cp := len(yPred)
	if cr != cp {
		return 0, errors.New("yReal and yPred not match")
	}

	if cr == 0 {
		return 0, nil
	}

	deviation := 0.0
	for i := 0; i < cr; i++ {
		e := yPred[i] - yReal[i]
		deviation += math.Pow(e, 2)
	}

	return deviation / float64(cr), nil
}

// GetRMSE returns Root Mean Squared Error which takes the square root of MSE
func GetRMSE(yReal []float64, yPred []float64) (float64, error) {
	mse, err := GetMSE(yReal, yPred)
	if err != nil {
		return 0, err
	}

	return math.Sqrt(mse), nil
}
