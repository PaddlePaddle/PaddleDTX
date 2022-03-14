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

import (
	"testing"
)

func TestRegMetrics(t *testing.T) {
	yReal := []float64{3, -0.5, 2, 7}
	yPred := []float64{2.5, 0.0, 2, 8}

	mse, err := GetMSE(yReal, yPred)
	checkErr(err, t)

	t.Logf("MSE is: %.4f", mse)

	rmse, err := GetRMSE(yReal, yPred)
	checkErr(err, t)

	t.Logf("RMSE is: %.4f", rmse)
}
