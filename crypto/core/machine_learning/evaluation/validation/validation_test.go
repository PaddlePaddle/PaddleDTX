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

package validation

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

var (
	binClassFileRows = [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "label", "id"},
		[]string{"11", "12", "13", "14", "15", "yes", "3"},
		[]string{"21", "22", "23", "24", "25", "yes", "6"},
		[]string{"31", "32", "33", "34", "35", "no", "1"},
		[]string{"41", "42", "43", "44", "45", "no", "4"},
		[]string{"51", "52", "53", "54", "55", "yes", "2"},
		[]string{"61", "62", "63", "64", "65", "yes", "5"},
		[]string{"11", "12", "13", "14", "15", "no", "13"},
		[]string{"21", "22", "23", "24", "25", "yes", "16"},
		[]string{"31", "32", "33", "34", "35", "no", "11"},
		[]string{"41", "42", "43", "44", "45", "no", "14"},
		[]string{"51", "52", "53", "54", "55", "yes", "12"},
		[]string{"61", "62", "63", "64", "65", "no", "15"},
	}

	regressionFileRows = [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "label", "id"},
		[]string{"11", "12", "13", "14", "15", "1", "3"},
		[]string{"21", "22", "23", "24", "25", "2.1", "6"},
		[]string{"31", "32", "33", "34", "35", "3.2", "1"},
		[]string{"41", "42", "43", "44", "45", "4.3", "4"},
		[]string{"51", "52", "53", "54", "55", "5.4", "2"},
		[]string{"61", "62", "63", "64", "65", "6.5", "5"},
		[]string{"11", "12", "13", "14", "15", "2.0", "13"},
		[]string{"21", "22", "23", "24", "25", "3.1", "16"},
		[]string{"31", "32", "33", "34", "35", "4.22", "11"},
		[]string{"41", "42", "43", "44", "45", "5.33", "14"},
		[]string{"51", "52", "53", "54", "55", "6.44", "12"},
		[]string{"61", "62", "63", "64", "65", "6", "15"},
	}
)

func TestSimpleSplitValOnBinClass(t *testing.T) {
	bcv, err := NewBinClassValidation(binClassFileRows, "label", "id", "yes", "no", 0.5)
	checkErr(err, t)

	err = bcv.Split(40)
	checkErr(err, t)

	folds, err := bcv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after SimpleSplit is: %v", folds)

	preSet, err := bcv.GetPredictSet(0)
	checkErr(err, t)
	t.Logf("PredictSet is: %v", preSet)

	trainSet, err := bcv.GetTrainSet(0)
	checkErr(err, t)
	t.Logf("TrainSet after holding out first fold is: %v", trainSet)

	predProba := mockPredictBinClass(preSet, trainSet, 5)
	t.Logf("Mocked Prediction is: %v", predProba)

	err = bcv.SetPredictOut(0, predProba)
	checkErr(err, t)
	pos := bcv.GetAllPredictOuts()
	t.Logf("PredictOut is: %v", pos)

	acc, err := bcv.GetAccuracy(0)
	checkErr(err, t)
	t.Logf("Accuracy over PredictSet is: %v", acc)

	valReport, err := bcv.GetReport(0)
	checkErr(err, t)
	t.Logf("Validation report over PredictSet is: %v", string(valReport))

	rocAndAucJson, err := bcv.GetROCAndAUC(0)
	checkErr(err, t)
	t.Logf("Validation roc and auc report over PredictSet is: %v", string(rocAndAucJson))
}

func TestShuffleSplitValOnBinClass(t *testing.T) {
	bcv, err := NewBinClassValidation(binClassFileRows, "label", "id", "yes", "", 0)
	checkErr(err, t)

	err = bcv.ShuffleSplit(40, "18f168b6-2ef2-491e-8b26-4aa6df18378a")
	checkErr(err, t)

	folds, err := bcv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after ShuffleSplit is: %v", folds)

	preSet, err := bcv.GetPredictSet(0)
	checkErr(err, t)
	t.Logf("PredictSet is: %v", preSet)

	trainSet, err := bcv.GetTrainSet(0)
	checkErr(err, t)
	t.Logf("TrainSet after holding out first fold is: %v", trainSet)

	predProba := mockPredictBinClass(preSet, trainSet, 5)
	t.Logf("Mocked Prediction is: %v", predProba)

	err = bcv.SetPredictOut(0, predProba)
	checkErr(err, t)
	pos := bcv.GetAllPredictOuts()
	t.Logf("PredictOut is: %v", pos)

	acc, err := bcv.GetAccuracy(0)
	checkErr(err, t)
	t.Logf("Accuracy over PredictSet is: %v", acc)

	valReport, err := bcv.GetReport(0)
	checkErr(err, t)
	t.Logf("Validation report over PredictSet is: %v", string(valReport))

	rocAndAucJson, err := bcv.GetROCAndAUC(0)
	checkErr(err, t)
	t.Logf("Validation roc and auc report over PredictSet is: %v", string(rocAndAucJson))
}

func TestKFoldsValOnBinClass(t *testing.T) {
	bcv, err := NewBinClassValidation(binClassFileRows, "label", "id", "yes", "no", 0)
	checkErr(err, t)

	err = bcv.KFoldsSplit(10)
	checkErr(err, t)

	folds, err := bcv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after KFoldsSplit is: %v", folds)

	for j, _ := range folds {
		preSet, err := bcv.GetPredictSet(j)
		checkErr(err, t)
		t.Logf("%dth PredictSet is: %v", j, preSet)

		trainSet, err := bcv.GetTrainSet(j)
		checkErr(err, t)
		t.Logf("TrainSet after holding out %dth fold is: %v", j, trainSet)

		predProba := mockPredictBinClass(preSet, trainSet, 5)
		t.Logf("%dth Mocked Prediction is: %v", j, predProba)

		err = bcv.SetPredictOut(j, predProba)
		checkErr(err, t)
		pos := bcv.GetAllPredictOuts()
		t.Logf("%dth PredictOut is: %v", j, pos[j])

		acc, err := bcv.GetAccuracy(j)
		checkErr(err, t)
		t.Logf("Accuracy over %dth PredictSet is: %.2f", j, acc)

		valReport, err := bcv.GetReport(j)
		checkErr(err, t)
		t.Logf("Validation report over %dth PredictSet is: %v", j, string(valReport))

		rocAndAucJson, err := bcv.GetROCAndAUC(0)
		checkErr(err, t)
		t.Logf("Validation roc and auc report over PredictSet is: %v", string(rocAndAucJson))
	}

	accs, mean, stdDev, err := bcv.GetAllAccuracy()
	checkErr(err, t)
	t.Logf("Accuracy over all split folds is: %v, and mean accuracy is %.2f with a standard deviation of %.2f", accs, mean, stdDev)

	valOverallReport, err := bcv.GetOverallReport()
	checkErr(err, t)

	var reportS string
	for k, v := range valOverallReport {
		reportS += fmt.Sprintf("%d-%s\n\a", k, string(v))
	}
	t.Logf("Validation OverallReport is:\n\a%v", reportS)

	allRocAndAuc, err := bcv.GetAllROCAndAUC()
	reportS = ""
	for k, v := range allRocAndAuc {
		reportS += fmt.Sprintf("%d-%s\n\a", k, string(v))
	}
	t.Logf("Validation Overall ROC and AUC Report is:\n\a%v", reportS)
}

func TestShuffleKFoldsValOnBinClass(t *testing.T) {
	bcv, err := NewBinClassValidation(binClassFileRows, "label", "id", "yes", "no", 0)
	checkErr(err, t)

	err = bcv.ShuffleKFoldsSplit(5, "18f168b6-2ef2-491e-8b26-4aa6df18378a")
	checkErr(err, t)

	folds, err := bcv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after ShuffleKFoldsSplit is: %v", folds)

	for j, _ := range folds {
		preSet, err := bcv.GetPredictSet(j)
		checkErr(err, t)
		t.Logf("%dth PredictSet is: %v", j, preSet)

		trainSet, err := bcv.GetTrainSet(j)
		checkErr(err, t)
		t.Logf("TrainSet after holding out %dth fold is: %v", j, trainSet)

		predProba := mockPredictBinClass(preSet, trainSet, 5)
		t.Logf("%dth Mocked Prediction is: %v", j, predProba)

		err = bcv.SetPredictOut(j, predProba)
		checkErr(err, t)
		pos := bcv.GetAllPredictOuts()
		t.Logf("%dth PredictOut is: %v", j, pos[j])

		acc, err := bcv.GetAccuracy(j)
		checkErr(err, t)
		t.Logf("Accuracy over %dth PredictSet is: %.2f", j, acc)

		valReport, err := bcv.GetReport(j)
		checkErr(err, t)
		t.Logf("Validation report over %dth PredictSet is: %v", j, string(valReport))
	}

	accs, mean, stdDev, err := bcv.GetAllAccuracy()
	checkErr(err, t)
	t.Logf("Accuracy over all split folds is: %v, and mean accuracy is %.2f with a standard deviation of %.2f", accs, mean, stdDev)

	valOverallReport, err := bcv.GetOverallReport()
	checkErr(err, t)

	var reportS string
	for k, v := range valOverallReport {
		reportS += fmt.Sprintf("%d-%s\n\a", k, string(v))
	}
	t.Logf("Validation OverallReport is:\n\a%v", reportS)

	allRocAndAuc, err := bcv.GetAllROCAndAUC()
	reportS = ""
	for k, v := range allRocAndAuc {
		reportS += fmt.Sprintf("%d-%s\n\a", k, string(v))
	}
	t.Logf("Validation Overall ROC and AUC Report is:\n\a%v", reportS)
}

func TestLooValOnBinClass(t *testing.T) {
	bcv, err := NewBinClassValidation(binClassFileRows, "label", "id", "yes", "no", 0)
	checkErr(err, t)

	err = bcv.LooSplit()
	checkErr(err, t)

	folds, err := bcv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after LooSplit is: %v", folds)

	for j, _ := range folds {
		preSet, err := bcv.GetPredictSet(j)
		checkErr(err, t)
		t.Logf("%dth PredictSet is: %v", j, preSet)

		trainSet, err := bcv.GetTrainSet(j)
		checkErr(err, t)
		t.Logf("TrainSet after holding out %dth fold is: %v", j, trainSet)

		predProba := mockPredictBinClass(preSet, trainSet, 5)
		t.Logf("%dth Mocked Prediction is: %v", j, predProba)

		err = bcv.SetPredictOut(j, predProba)
		checkErr(err, t)
		pos := bcv.GetAllPredictOuts()
		t.Logf("%dth PredictOut is: %v", j, pos[j])

		acc, err := bcv.GetAccuracy(j)
		checkErr(err, t)
		t.Logf("Accuracy over %dth PredictSet is: %.2f", j, acc)

		valReport, err := bcv.GetReport(j)
		checkErr(err, t)
		t.Logf("Validation report over %dth PredictSet is: %v", j, string(valReport))
	}
	accs, mean, stdDev, err := bcv.GetAllAccuracy()
	checkErr(err, t)
	t.Logf("Accuracy over all split folds is: %v, and mean accuracy is %.2f with a standard deviation of %.2f", accs, mean, stdDev)

	valOverallReport, err := bcv.GetOverallReport()
	checkErr(err, t)

	var reportS string
	for k, v := range valOverallReport {
		reportS += fmt.Sprintf("%d-%s\n\a", k, string(v))
	}
	t.Logf("Validation OverallReport is:\n\a%v", reportS)

	allRocAndAuc, err := bcv.GetAllROCAndAUC()
	reportS = ""
	for k, v := range allRocAndAuc {
		reportS += fmt.Sprintf("%d-%s\n\a", k, string(v))
	}
	t.Logf("Validation Overall ROC and AUC Report is:\n\a%v", reportS)
}

func TestSimpleSplitValOnRegression(t *testing.T) {
	rv, _ := NewRegressionValidation(regressionFileRows, "label", "id")

	err := rv.Split(40)
	checkErr(err, t)

	folds, err := rv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after SimpleSplit is: %v", folds)

	preSet, err := rv.GetPredictSet(0)
	checkErr(err, t)
	t.Logf("PredictSet is: %v", preSet)

	trainSet, err := rv.GetTrainSet(0)
	checkErr(err, t)
	t.Logf("TrainSet after holding out first fold is: %v", trainSet)

	yPred := mockPredictReg(preSet, trainSet, 5)
	t.Logf("Mocked Prediction is: %v", yPred)

	err = rv.SetPredictOut(0, yPred)
	checkErr(err, t)
	pos := rv.GetAllPredictOuts()
	t.Logf("PredictOut is: %v", pos)

	rmse, err := rv.GetRMSE(0)
	checkErr(err, t)
	t.Logf("RMSE over PredictSet is: %v", rmse)
}

func TestShuffleSplitValOnRegression(t *testing.T) {
	rv, _ := NewRegressionValidation(regressionFileRows, "label", "id")
	err := rv.ShuffleSplit(40, "18f168b6-2ef2-491e-8b26-4aa6df18378a")
	checkErr(err, t)

	folds, err := rv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after ShuffleSplit is: %v", folds)

	preSet, err := rv.GetPredictSet(0)
	checkErr(err, t)
	t.Logf("PredictSet is: %v", preSet)

	trainSet, err := rv.GetTrainSet(0)
	checkErr(err, t)
	t.Logf("TrainSet after holding out first fold is: %v", trainSet)

	yPred := mockPredictReg(preSet, trainSet, 5)
	t.Logf("Mocked Prediction is: %v", yPred)

	err = rv.SetPredictOut(0, yPred)
	checkErr(err, t)
	pos := rv.GetAllPredictOuts()
	t.Logf("PredictOut is: %v", pos)

	rmse, err := rv.GetRMSE(0)
	checkErr(err, t)
	t.Logf("RMSE over PredictSet is: %v", rmse)
}

func TestKFoldsValOnRegression(t *testing.T) {
	rv, _ := NewRegressionValidation(regressionFileRows, "label", "id")
	err := rv.KFoldsSplit(5)
	checkErr(err, t)

	folds, err := rv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after KFoldsSplit is: %v", folds)

	for j, _ := range folds {
		preSet, err := rv.GetPredictSet(j)
		checkErr(err, t)
		t.Logf("%dth PredictSet is: %v", j, preSet)

		trainSet, err := rv.GetTrainSet(j)
		checkErr(err, t)
		t.Logf("TrainSet after holding out %dth fold is: %v", j, trainSet)

		yPred := mockPredictReg(preSet, trainSet, 5)
		t.Logf("%dth Mocked Prediction is: %v", j, yPred)

		err = rv.SetPredictOut(j, yPred)
		checkErr(err, t)
		pos := rv.GetAllPredictOuts()
		t.Logf("%dth PredictOut is: %v", j, pos[j])

		rmse, err := rv.GetRMSE(j)
		checkErr(err, t)
		t.Logf("RMSE over %dth PredictSet is: %.2f", j, rmse)

	}

	rmses, mean, stdDev, err := rv.GetAllRMSE()
	checkErr(err, t)
	t.Logf("RMSE over all split folds is: %v, and mean RMSE is %.2f with a standard deviation of %.2f", rmses, mean, stdDev)
}

func TestShuffleKFoldsValOnRegression(t *testing.T) {
	rv, _ := NewRegressionValidation(regressionFileRows, "label", "id")
	err := rv.ShuffleKFoldsSplit(5, "18f168b6-2ef2-491e-8b26-4aa6df18378a")
	checkErr(err, t)

	folds, err := rv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after ShuffleKFoldsSplit is: %v", folds)

	for j, _ := range folds {
		preSet, err := rv.GetPredictSet(j)
		checkErr(err, t)
		t.Logf("%dth PredictSet is: %v", j, preSet)

		trainSet, err := rv.GetTrainSet(j)
		checkErr(err, t)
		t.Logf("TrainSet after holding out %dth fold is: %v", j, trainSet)

		yPred := mockPredictReg(preSet, trainSet, 5)
		t.Logf("%dth Mocked Prediction is: %v", j, yPred)

		err = rv.SetPredictOut(j, yPred)
		checkErr(err, t)
		pos := rv.GetAllPredictOuts()
		t.Logf("%dth PredictOut is: %v", j, pos[j])

		rmse, err := rv.GetRMSE(j)
		checkErr(err, t)
		t.Logf("RMSE over %dth PredictSet is: %.2f", j, rmse)
	}

	rmses, mean, stdDev, err := rv.GetAllRMSE()
	checkErr(err, t)
	t.Logf("RMSE over all split folds is: %v, and mean RMSE is %.2f with a standard deviation of %.2f", rmses, mean, stdDev)
}

func TestLooValOnRegression(t *testing.T) {
	rv, _ := NewRegressionValidation(regressionFileRows, "label", "id")
	err := rv.LooSplit()
	checkErr(err, t)

	folds, err := rv.GetAllFolds()
	checkErr(err, t)
	t.Logf("All subsets after LooSplit is: %v", folds)

	for j, _ := range folds {
		preSet, err := rv.GetPredictSet(j)
		checkErr(err, t)
		t.Logf("%dth PredictSet is: %v", j, preSet)

		trainSet, err := rv.GetTrainSet(j)
		checkErr(err, t)
		t.Logf("TrainSet after holding out %dth fold is: %v", j, trainSet)

		yPred := mockPredictReg(preSet, trainSet, 5)
		t.Logf("%dth Mocked Prediction is: %v", j, yPred)

		err = rv.SetPredictOut(j, yPred)
		checkErr(err, t)
		pos := rv.GetAllPredictOuts()
		t.Logf("%dth PredictOut is: %v", j, pos[j])

		rmse, err := rv.GetRMSE(j)
		checkErr(err, t)
		t.Logf("RMSE over %dth PredictSet is: %.2f", j, rmse)
	}
	rmses, mean, stdDev, err := rv.GetAllRMSE()
	checkErr(err, t)
	t.Logf("RMSE over all split folds is: %v, and mean RMSE is %.2f with a standard deviation of %.2f", rmses, mean, stdDev)
}

func mockPredictBinClass(preSet [][]string, trainSet [][]string, idx int) []float64 {

	predProba := []float64{}
	for i := 0; i < len(preSet)-1; i++ {
		if i%2 == 0 {
			predProba = append(predProba, 1.0)
		} else {
			predProba = append(predProba, 0.1)
		}
	}
	return predProba
}

func mockPredictReg(preSet [][]string, trainSet [][]string, idx int) []float64 {
	trainSet = trainSet[1:]
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(trainSet), func(i, j int) {
		trainSet[i], trainSet[j] = trainSet[j], trainSet[i]
	})

	yPred := []float64{}
	for i := 0; i < len(preSet)-1; i++ {
		v, err := strconv.ParseFloat(trainSet[0][idx], 64)
		if err != nil {
			v = 0.0
		}

		yPred = append(yPred, v)
	}
	return yPred
}
