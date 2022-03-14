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
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testMetricsRelatedConfusionMatrix struct {
	TP        float64
	FP        float64
	FN        float64
	TN        float64
	Precision float64
	Recall    float64
	F1Score   float64
}

type testConfusionMatrixSummary struct {
	Metrics  map[string]testMetricsRelatedConfusionMatrix
	Accuracy float64
}

func TestBinclassConfusionMatrix(t *testing.T) {
	rcs := []string{"P", "N", "P", "P", "N", "N", "P", "P"}
	pcs := []string{"P", "N", "P", "P", "P", "N", "N", "P"}

	cm, err := NewConfusionMatrix(rcs, pcs)
	checkErr(err, t)

	t.Logf("\n\aNew ConfusionMatrix created, and it looks like: \n\a%s", cm)
	t.Logf("\n\aSummary of ConfusionMatrix is: \n\a%s", cm.Summary())

	sumJSON, err := cm.SummaryAsJSON()
	checkErr(err, t)
	t.Logf("\n\aSummary of ConfusionMatrix is also like: \n\a%v", string(sumJSON))

	var sum testConfusionMatrixSummary
	err = json.Unmarshal(sumJSON, &sum)
	checkErr(err, t)

	t.Logf("\n\aSummary of ConfusionMatrix is also like: \n\a%v", sum)
}

func TestMultclassConfusionMatrix(t *testing.T) {

	rcs := []string{"cat", "ant", "cat", "cat", "ant", "bird"}
	pcs := []string{"ant", "ant", "cat", "cat", "ant", "cat"}
	// No bird appeared in predicted class set

	// rcs := []string{"Cat", "Dog", "Rabbit", "Rabbit", "Dog", "Cat", "Rabbit", "Dog", "Cat", "Cat"}
	// pcs := []string{"Rabbit", "Dog", "Cat", "Cat", "Dog", "Cat", "Cat", "Dog", "Cat", "Cat"}
	// All Rabbits are predicted incorrectly

	cm, err := NewConfusionMatrix(rcs, pcs)
	checkErr(err, t)

	t.Logf("New ConfusionMatrix created, and it looks like: \n\a%s", cm)
	t.Logf("\n\aSummary of ConfusionMatrix is: \n\a%s", cm.Summary())

	sumJSON, err := cm.SummaryAsJSON()
	checkErr(err, t)
	t.Logf("\n\aSummary of ConfusionMatrix is also like: \n\a%v", string(sumJSON))

	var sum testConfusionMatrixSummary
	err = json.Unmarshal(sumJSON, &sum)
	checkErr(err, t)

	t.Logf("\n\aSummary of ConfusionMatrix  is also like: \n\a%v", sum)
}

func TestROCandAUC(t *testing.T) {
	rcs := []string{"P", "P", "N", "P", "P", "P", "N", "N", "P", "N", "P", "N", "P", "N", "N", "N", "P", "N", "P", "N"}
	predValues := []float64{0.9, 0.8, 0.7, 0.6, 0.55, 0.54, 0.53, 0.52, 0.51, 0.505, 0.4, 0.39, 0.38, 0.37, 0.36, 0.35, 0.34, 0.33, 0.30, 0.1}
	points, err := GetROC(rcs, predValues, "P")
	checkErr(err, t)

	for i := 0; i < len(points); i++ {
		t.Logf("ROC's point no.%d,  x: %2f , y: %2f, threshold: %2f\n", i, points[i][0], points[i][1], points[i][2])
	}

	auc, err := GetAUC(GetCoordinates(points))
	t.Logf("AUC: %f, %s", auc, err)
	assert.Equal(t, fmt.Sprintf("%.2f", auc), "0.68")
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
