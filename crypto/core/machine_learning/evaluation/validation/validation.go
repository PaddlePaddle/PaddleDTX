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
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"sync"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/evaluation/metrics"
)

// BinClassValidation performs validation of Binary Classfication case
type BinClassValidation interface {
	// Splitter divides data set into several subsets with some strategies (such as KFolds, LOO),
	// and hold out one subset as validation set and others as training set
	Splitter

	// SetPredictOut sets predicted probabilities from a prediction set to which `idx` refers.
	SetPredictOut(idx int, predProbas []float64) error

	// GetAllPredictOuts returns all prediction results has been stored.
	GetAllPredictOuts() map[int][]string

	// GetAccuracy returns classification accuracy.
	// idx is the index of prediction set (also of validation set) in split folds.
	GetAccuracy(idx int) (float64, error)

	// GetAllAccuracy returns scores of classification accuracy over all split folds,
	// and its Mean and Standard Deviation.
	GetAllAccuracy() (map[int]float64, float64, float64, error)

	// GetReport returns a json bytes of precision, recall, f1, true positive,
	// false positive, true negatives and false negatives for each class, and accuracy.
	GetReport(idx int) ([]byte, error)

	// GetReport returns a json bytes of precision, recall, f1, true positive,
	// false positive, true negatives and false negatives for each class, and accuracy, over all split folds.
	GetOverallReport() (map[int][]byte, error)

	// GetROCAndAUC returns a json bytes of roc's points and auc.
	GetROCAndAUC(idx int) ([]byte, error)

	// GetAllROCAndAUC returns a map contains all split folds' json bytes of roc and auc.
	GetAllROCAndAUC() (map[int][]byte, error)
}

// RegressionValidation performs validation of Regression case
type RegressionValidation interface {
	// Splitter divides data set into several subsets with some strategies (such as KFolds, LOO),
	// and hold out one subset as validation set and others as training set
	Splitter

	// SetPredictOut sets prediction outcomes from a prediction set to which `idx` refers.
	SetPredictOut(idx int, yPred []float64) error

	// GetAllPredictOuts returns all prediction results has been stored.
	GetAllPredictOuts() map[int][]float64

	// GetRMSE returns RMSE over the validation set to which `idx` refers.
	GetRMSE(idx int) (float64, error)

	// GetAllRMSE returns scores of RMSE over all split folds,
	// and its Mean and Standard Deviation.
	GetAllRMSE() (map[int]float64, float64, float64, error)
}

// Splitter divides data set into several subsets with some strategies (such as KFolds, LOO),
// and hold out one subset as validation set and others as training set
type Splitter interface {
	// Split divides the file into two parts directly
	// based on percentage which denotes the first part of divisions.
	Split(percents int) error

	// ShuffleSplit shuffles the rows with `seed`,
	// then divides the file into two parts
	// based on `percents` which denotes the first part of divisions.
	ShuffleSplit(percents int, seed string) error

	// KFoldsSplit divides the file into `k` parts directly.
	// k is the number of parts that only could be 5 or 10.
	KFoldsSplit(k int) error

	// ShuffleKFoldsSplit shuffles the sorted rows with `seed`,
	// then divides the file into `k` parts.
	// k is the number of parts that only could be 5 or 10.
	ShuffleKFoldsSplit(k int, seed string) error

	// LooSplit sorts file rows by IDs which extracted from file by `idName`,
	// then divides each row into a subset.
	LooSplit() error

	// GetAllFolds returns all folds after split.
	// And could be only called successfully after split.
	GetAllFolds() ([][][]string, error)

	// GetTrainSet holds out the subset to which refered by `idxHO`
	// and returns the remainings as training set.
	GetTrainSet(idxHO int) ([][]string, error)

	// GetPredictSet returns the subset to which refered by `idx`
	// as predicting set (without label feature).
	GetPredictSet(idx int) ([][]string, error)

	// GetPredictSet returns the subset to which refered by `idx`
	// as validation set.
	GetValidSet(idx int) ([][]string, error)
}

type binClassValidation struct {
	// Splitter divides data set into several subsets with some strategies (such as KFolds, LOO),
	// and hold out one subset as validation set and others as training set
	Splitter

	// label denotes name of lable feature
	label string

	// posClass denotes name of positive class
	posClass string

	// negClass denotes name of negtive class
	negClass string

	// classify sample as a positive class if its predicted probability exceeds threshold
	threshold float64

	// predResults stores prediction outcomes
	predResults sync.Map

	// predClasses stores predicted classes
	predClasses sync.Map
}

// NewBinClassValidation creates a BinClassValidation instance to handle binary classification validation.
// file contains all rows of a file,
//  and its first row contains just names of feature, and others contain all feature values
// idName denotes which feature is ID that would be used in sample alignment
// label denotes name of lable feature
// posClass denotes name of positive class and must be one feature name in `file`
// negClass denotes name of negtive class, could be set with empty string
func NewBinClassValidation(file [][]string, label string, idName string,
	posClass string, negClass string, threshold float64) (BinClassValidation, error) {
	if len(negClass) == 0 {
		negClass = "non-" + posClass
	}
	if threshold <= 0 {
		threshold = 0.5
	}

	lf := len(file)
	newFile := make([][]string, 0, lf)
	if lf <= 1 {
		return nil, errors.New("invalid file")
	}

	// first row contains just names of feature
	// find where the label feature is
	idx := -1
	for i, v := range file[0] {
		if v == label {
			idx = i
			break
		}
	}
	if idx < 0 { // find no label feature
		for _, r := range file {
			newR := make([]string, 0, len(file[0]))
			newR = append(newR, r...)
			newFile = append(newFile, newR)
		}
	} else {
		newR := make([]string, 0, len(file[0]))
		newR = append(newR, file[0]...)

		newFile = append(newFile, newR)

		// reset value of label row by row
		for _, r := range file[1:] {
			if len(r) <= idx {
				return nil, errors.New("invalid file")
			}

			newR := make([]string, 0, len(file[0]))
			newR = append(newR, r[0:idx]...)
			if r[idx] == posClass {
				newR = append(newR, posClass)
			} else {
				newR = append(newR, negClass)
			}
			if len(r) > idx+1 {
				newR = append(newR, r[idx+1:]...)
			}
			newFile = append(newFile, newR)
		}
	}

	return &binClassValidation{
		Splitter:  NewSplitter(newFile, idName, label),
		label:     label,
		posClass:  posClass,
		negClass:  negClass,
		threshold: threshold,
	}, nil
}

// SetPredictOut sets predicted probabilities for a prediction set to which `idx` refers.
// returns error if the file hasn't been split or other errors occur.
func (bv *binClassValidation) SetPredictOut(idx int, predProbas []float64) error {
	set, err := bv.GetValidSet(idx)
	if err != nil {
		return err
	}

	lp := len(predProbas)
	if len(set)-1 != lp {
		return errors.New("there is a mismatch between the number of predicted classes and that of prediction set")
	}

	classes := make([]string, 0, lp)
	for _, p := range predProbas {
		c := bv.posClass
		if p <= bv.threshold {
			c = bv.negClass
		}
		classes = append(classes, c)
	}
	bv.predResults.Store(idx, predProbas)
	bv.predClasses.Store(idx, classes)
	return nil
}

// GetAllPredictOuts returns all prediction results has been stored.
func (bv *binClassValidation) GetAllPredictOuts() map[int][]string {
	ret := make(map[int][]string)
	bv.predClasses.Range(func(key, value interface{}) bool {
		ret[key.(int)] = value.([]string)
		return true
	})
	return ret
}

// GetAccuracy returns classification accuracy.
// idx is the index of prediction set (also of validation set) in split folds.
func (bv *binClassValidation) GetAccuracy(idx int) (float64, error) {
	predClasses, ok := bv.predClasses.Load(idx)
	if !ok {
		return 0, errors.New("not find prediction outcomes according to idx")
	}

	validSet, err := bv.GetValidSet(idx)
	if err != nil {
		return 0, err
	}

	realClasses, err := getFeaturesByName(validSet, bv.label)
	if err != nil {
		return 0, err
	}

	cm, err := metrics.NewConfusionMatrix(realClasses, predClasses.([]string))
	if err != nil {
		return 0, err
	}

	return cm.GetAccuracy(), nil
}

// GetAllAccuracy returns scores of classification accuracy over all split folds,
// and its Mean and Standard Deviation.
func (bv *binClassValidation) GetAllAccuracy() (map[int]float64, float64, float64, error) {
	var errRet error
	accs := make(map[int]float64)
	bv.predClasses.Range(func(key, value interface{}) bool {
		i := key.(int)
		predClasses := value.([]string)

		validSet, err := bv.GetValidSet(i)
		if err != nil {
			errRet = err
			return false
		}

		realClasses, err := getFeaturesByName(validSet, bv.label)
		if err != nil {
			errRet = err
			return false
		}

		cm, err := metrics.NewConfusionMatrix(realClasses, predClasses)
		if err != nil {
			errRet = err
			return false
		}

		accs[i] = cm.GetAccuracy()
		return true
	})

	if errRet != nil {
		return map[int]float64{}, 0, 0, errRet
	}

	meanAcc, stdDevAcc := getStdDeviation(accs)

	return accs, meanAcc, stdDevAcc, nil
}

// GetReport returns a json bytes of precision, recall, f1, true positive,
// false positive, true negatives and false negatives for each class, and accuracy.
// JSON type summary is something like :
// {
//	"Metrics": {
//		"NO": {
//			"TP": 2,
//			"FP": 1,
//			"FN": 1,
//			"TN": 4,
//			"Precision": 0.6666666666666666,
//			"Recall": 0.6666666666666666,
//			"F1Score": 0.6666666666666666
//		},
//		"YES": {
//			"TP": 4,
//			"FP": 1,
//			"FN": 1,
//			"TN": 2,
//			"Precision": 0.8,
//			"Recall": 0.8,
//			"F1Score": 0.8000000000000002
//		}
//	},
//	"Accuracy": 0.75
//}
// NO and Yes are classes.
// idx is the index of prediction set (also of validation set) in split folds.
func (bv *binClassValidation) GetReport(idx int) ([]byte, error) {
	predClasses, ok := bv.predClasses.Load(idx)
	if !ok {
		return []byte{}, errors.New("not find prediction outcomes according to idx")
	}

	validSet, err := bv.GetValidSet(idx)
	if err != nil {
		return []byte{}, err
	}

	realClasses, err := getFeaturesByName(validSet, bv.label)
	if err != nil {
		return []byte{}, err
	}

	cm, err := metrics.NewConfusionMatrix(realClasses, predClasses.([]string))
	if err != nil {
		return []byte{}, err
	}

	return cm.SummaryAsJSON()
}

// GetReport returns a json bytes of precision, recall, f1, true positive,
// false positive, true negatives and false negatives for each class, and accuracy, over all split folds.
// key of return is the index of fold
// and value of return is JSON type summary, something like :
// {
//	"Metrics": {
//		"NO": {
//			"TP": 2,
//			"FP": 1,
//			"FN": 1,
//			"TN": 4,
//			"Precision": 0.6666666666666666,
//			"Recall": 0.6666666666666666,
//			"F1Score": 0.6666666666666666
//		},
//		"YES": {
//			"TP": 4,
//			"FP": 1,
//			"FN": 1,
//			"TN": 2,
//			"Precision": 0.8,
//			"Recall": 0.8,
//			"F1Score": 0.8000000000000002
//		}
//	},
//	"Accuracy": 0.75
//}
// NO and Yes are classes.
func (bv *binClassValidation) GetOverallReport() (map[int][]byte, error) {
	var errRet error
	summaries := make(map[int][]byte)
	bv.predClasses.Range(func(key, value interface{}) bool {
		i := key.(int)
		predClasses := value.([]string)

		validSet, err := bv.GetValidSet(i)
		if err != nil {
			errRet = err
			return false
		}

		realClasses, err := getFeaturesByName(validSet, bv.label)
		if err != nil {
			errRet = err
			return false
		}

		cm, err := metrics.NewConfusionMatrix(realClasses, predClasses)
		if err != nil {
			errRet = err
			return false
		}

		summary, err := cm.SummaryAsJSON()
		if err != nil {
			errRet = err
			return false
		}

		summaries[i] = summary
		return true
	})

	if errRet != nil {
		return map[int][]byte{}, errRet
	}

	return summaries, nil
}

type reportROCAndAUC struct {
	// Roc is expressed by a series of points.
	// A point of roc is represented by [3]float64, [FPR, TPR, threshold]([x,y,threshold])
	PointsOnROC [][3]float64
	// AUC is the area under curve ROC.
	AUC float64
}

// GetROCAndAUC returns a json bytes of roc's points and auc.
// JSON type summary is something like :
// {
//	"PointsOnROC": [
//			[0,0,1.9],
// 			[0,0.1,0.9],
//			[0,0.2,0.8],
//			[0.1,0.2,0.7],
//			[0.1,0.3,0.6],
//			[0.1,0.4,0.55]，
//			...
//		],
//	"AUC":0.68
//}
// PointsOnROC is a [3]float64, represents [FPR, TPR, threshold]([x,y,threshold])
// idx is the index of prediction set (also of validation set) in split folds.
func (bv *binClassValidation) GetROCAndAUC(idx int) ([]byte, error) {
	predResult, ok := bv.predResults.Load(idx)
	if !ok {
		return []byte{}, errors.New("not find prediction results according to idx")
	}

	validSet, err := bv.GetValidSet(idx)
	if err != nil {
		return []byte{}, err
	}

	realClasses, err := getFeaturesByName(validSet, bv.label)
	if err != nil {
		return []byte{}, err
	}

	points, err := metrics.GetROC(realClasses, predResult.([]float64), bv.posClass)
	if err != nil {
		return []byte{}, err
	}

	auc, err := metrics.GetAUC(metrics.GetCoordinates(points))
	if err != nil {
		return []byte{}, err
	}

	return json.Marshal(&reportROCAndAUC{
		PointsOnROC: points,
		AUC:         auc,
	})
}

// GetAllROCAndAUC returns a map contains all split folds' json bytes of roc and auc.
// JSON type summary is something like :
// {
//	"PointsOnROC": [
//			[0,0,1.9],
// 			[0,0.1,0.9],
//			[0,0.2,0.8],
//			[0.1,0.2,0.7],
//			[0.1,0.3,0.6],
//			[0.1,0.4,0.55]，
//			...
//		],
//	"AUC":0.68
//}
// PointsOnROC is a [3]float64, represents [FPR, TPR, threshold]([x,y,threshold])
// map's idx is the index of prediction set (also of validation set) in split folds.
func (bv *binClassValidation) GetAllROCAndAUC() (map[int][]byte, error) {
	var errRet error
	summaries := make(map[int][]byte)
	bv.predResults.Range(func(key, value interface{}) bool {
		i := key.(int)
		predResult := value.([]float64)

		validSet, err := bv.GetValidSet(i)
		if err != nil {
			errRet = err
			return false
		}

		realClasses, err := getFeaturesByName(validSet, bv.label)
		if err != nil {
			errRet = err
			return false
		}

		points, err := metrics.GetROC(realClasses, predResult, bv.posClass)
		if err != nil {
			errRet = err
			return false
		}

		auc, err := metrics.GetAUC(metrics.GetCoordinates(points))
		if err != nil {
			errRet = err
			return false
		}

		summary, _ := json.Marshal(&reportROCAndAUC{
			PointsOnROC: points,
			AUC:         auc,
		})

		summaries[i] = summary
		return true
	})

	if errRet != nil {
		return map[int][]byte{}, errRet
	}

	return summaries, nil
}

type regressionValidation struct {
	// Splitter divides data set into several subsets with some strategies (such as KFolds, LOO),
	// and hold out one subset as validation set and others as training set.
	Splitter

	// label denotes name of lable feature.
	label string

	// predResults stores prediction outcomes.
	predResults sync.Map
}

// NewRegressionValidation creates a RegressionValidation instance to handle regression validation.
// file contains all rows of a file,
//  and its first row contains just names of feature, and others contain all feature values
// idName denotes which feature is ID that would be used in sample alignment
func NewRegressionValidation(file [][]string, label string, idName string) (RegressionValidation, error) {
	return &regressionValidation{
		Splitter: NewSplitter(file, idName, label),
		label:    label,
	}, nil
}

// SetPredictOut sets prediction outcomes for a prediction set to which `idx` refers.
func (rv *regressionValidation) SetPredictOut(idx int, yPred []float64) error {
	set, err := rv.GetValidSet(idx)
	if err != nil {
		return err
	}

	if len(set)-1 != len(yPred) {
		return errors.New("there is a mismatch between the number of predicted values and that of prediction set")
	}

	rv.predResults.Store(idx, yPred)
	return nil
}

// GetAllPredictOuts returns all prediction results has been stored.
func (rv *regressionValidation) GetAllPredictOuts() map[int][]float64 {
	ret := make(map[int][]float64)
	rv.predResults.Range(func(key, value interface{}) bool {
		ret[key.(int)] = value.([]float64)
		return true
	})
	return ret
}

// GetRMSE returns RMSE over the validation set to which `idx` refers.
func (rv *regressionValidation) GetRMSE(idx int) (float64, error) {
	yPredS, ok := rv.predResults.Load(idx)
	if !ok {
		return 0, errors.New("not find prediction outcomes according to idx")
	}

	validSet, err := rv.GetValidSet(idx)
	if err != nil {
		return 0, err
	}
	yPred := yPredS.([]float64)

	yRealS, err := getFeaturesByName(validSet, rv.label)
	if err != nil {
		return 0, err
	}

	yReal := make([]float64, 0, len(yRealS))
	for _, v := range yRealS {
		v2, err := strconv.ParseFloat(v, 64)
		if err != nil {

			return 0, errors.New("failed to parse label from file, and error is:" + err.Error())
		}
		yReal = append(yReal, v2)
	}

	return metrics.GetRMSE(yReal, yPred)
}

// GetAllRMSE returns scores of RMSE over all split folds,
// and its Mean and Standard Deviation.
func (rv *regressionValidation) GetAllRMSE() (map[int]float64, float64, float64, error) {
	var errRet error
	rmses := make(map[int]float64)
	rv.predResults.Range(func(key, value interface{}) bool {
		i := key.(int)
		yPred := value.([]float64)

		validSet, err := rv.GetValidSet(i)
		if err != nil {
			errRet = err
			return false
		}

		yRealS, err := getFeaturesByName(validSet, rv.label)
		if err != nil {
			errRet = err
			return false
		}

		yReal := make([]float64, 0, len(yRealS))
		for _, v := range yRealS {
			v2, err := strconv.ParseFloat(v, 64)
			if err != nil {
				errRet = errors.New("failed to parse label from file, and error is:" + err.Error())
				return false
			}
			yReal = append(yReal, v2)
		}

		rmse, err := metrics.GetRMSE(yReal, yPred)
		if err != nil {
			errRet = err
			return false
		}

		rmses[i] = rmse
		return true
	})

	if errRet != nil {
		return map[int]float64{}, 0, 0, errRet
	}

	meanRMSE, stdDevRMSE := getStdDeviation(rmses)

	return rmses, meanRMSE, stdDevRMSE, nil
}

type splitter struct {
	//fileRows are all rows of a file
	fileRows [][]string

	//idName denotes which feature is ID that would be used in sample alignment
	idName string

	// label denotes name of lable feature
	label string

	//folds stores division result of `fileRows`
	folds [][][]string
}

// NewSplitter creates a Splitter instance.
// file contains all rows of a file,
//  and its first row contains just names of feature, and others contain all feature values.
// idName denotes which feature is ID that would be used in sample alignment.
// label denotes name of lable feature.
func NewSplitter(file [][]string, idName string, label string) Splitter {
	return &splitter{
		idName:   idName,
		fileRows: file,
		label:    label,
	}
}

// Split divides the file into two parts directly
// based on percentage which denotes the first part of divisions.
func (s *splitter) Split(percents int) error {
	if percents < 1 || percents > 100 {
		return errors.New("percents must between 1 and 100")
	}
	splitSets, err := Split(s.fileRows, percents)
	if err != nil {
		return err
	}

	s.folds = make([][][]string, 0, 2)
	s.folds = append(s.folds, splitSets[0], splitSets[1])
	return nil
}

// ShuffleSplit sorts file rows by IDs which extracted from file by `idName`,
// and shuffles the sorted rows with `seed`,
// then divides the file into two parts
// based on `percents` which denotes the first part of divisions.
func (s *splitter) ShuffleSplit(percents int, seed string) error {
	if percents < 1 || percents > 100 {
		return errors.New("percents must between 1 and 100")
	}

	splitSets, err := ShuffleSplit(s.fileRows, s.idName, percents, seed)
	if err != nil {
		return err
	}

	s.folds = make([][][]string, 0, 2)
	s.folds = append(s.folds, splitSets[0], splitSets[1])
	return nil
}

// KFoldsSplit divides the file into `k` parts directly.
// k is the number of parts that only could be 5 or 10.
func (s *splitter) KFoldsSplit(k int) error {
	splitSets, err := KFoldsSplit(s.fileRows, k)
	if err != nil {
		return err
	}

	s.folds = splitSets
	return nil
}

// ShuffleKFoldsSplit sorts file rows by IDs which extracted from file by `idName`,
// and shuffles the sorted rows with `seed`,
// then divides the file into `k` parts.
// k is the number of parts that only could be 5 or 10.
func (s *splitter) ShuffleKFoldsSplit(k int, seed string) error {
	splitSets, err := ShuffleKFoldsSplit(s.fileRows, s.idName, k, seed)
	if err != nil {
		return err
	}

	s.folds = splitSets
	return nil
}

// LooSplit sorts file rows by IDs which extracted from file by `idName`,
// then divides each row into a subset.
func (s *splitter) LooSplit() error {
	splitSets, err := LooSplit(s.fileRows, s.idName)
	if err != nil {
		return err
	}

	s.folds = splitSets
	return nil
}

// GetAllFolds returns all folds after split.
// And could be only called successfully after split.
func (s *splitter) GetAllFolds() ([][][]string, error) {
	if len(s.folds) == 0 {
		return [][][]string{}, errors.New("the file has not been split")
	}
	return s.folds, nil
}

// GetTrainSet holds out the subset to which refered by `idxHO`
// and returns the remainings as training set.
func (s *splitter) GetTrainSet(idxHO int) ([][]string, error) {
	l := len(s.folds)
	if l == 0 {
		return [][]string{}, errors.New("the file has not been split")
	}

	if idxHO >= l {
		return [][]string{}, errors.New("invalid index referring to subset held out")
	}

	lHO := len(s.folds[idxHO])
	lTrain := len(s.fileRows) - lHO + 1 // each subset has a row containing the names of feature, so take it out

	trainSet := make([][]string, 0, lTrain)
	trainSet = append(trainSet, s.fileRows[0])

	for i, fold := range s.folds {
		if i == idxHO {
			continue
		}

		trainSet = append(trainSet, fold[1:]...)
	}

	return trainSet, nil
}

// GetPredictSet returns the subset to which refered by `idx`
// as predicting set (without label feature).
func (s *splitter) GetPredictSet(idx int) ([][]string, error) {
	validSet, err := s.GetValidSet(idx)
	if err != nil {
		return validSet, err
	}

	// find label feature and remove it row by row.
	idxL := -1
	for i, v := range validSet[0] {
		if v == s.label {
			idxL = i
			break
		}
	}

	lenFile := len(validSet)
	predictFile := make([][]string, 0, lenFile)

	if idxL < 0 { //no label feature found, no need to remove it
		predictFile = validSet
	} else {
		for _, r := range validSet {
			lR := len(r)
			if lR <= idxL {
				return [][]string{}, errors.New("invalid file")
			}

			newR := make([]string, 0, lR-1)
			for i, s := range r {
				if i == idxL {
					continue
				}
				newR = append(newR, s)
			}

			predictFile = append(predictFile, newR)
		}
	}

	return predictFile, nil
}

// GetPredictSet returns the subset to which refered by `idx`
// as validation set.
func (s *splitter) GetValidSet(idx int) ([][]string, error) {
	l := len(s.folds)
	if l == 0 {
		return [][]string{}, errors.New("the file has not been split")
	}

	if idx >= l {
		return [][]string{}, errors.New("invalid index referring to validation set ")
	}

	return s.folds[idx], nil
}

// getFeaturesByName abstracts features from file according to `name`,
// and order of returns is the same as that of rows.
func getFeaturesByName(fileRows [][]string, name string) ([]string, error) {
	lenFile := len(fileRows)
	if lenFile < 1 {
		return []string{}, errors.New("invalid file")
	}

	// find where the target feature is
	idx := -1
	for i, v := range fileRows[0] {
		if v == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []string{}, errors.New("not find name")
	}

	features := make([]string, 0, lenFile-1)

	for _, r := range fileRows[1:] { // first row contains just names of feature, skip it
		if len(r) <= idx {
			return []string{}, errors.New("invalid file")
		}

		features = append(features, r[idx])
	}

	return features, nil
}

// getStdDeviation return mean and standard deviation.
func getStdDeviation(score map[int]float64) (float64, float64) {
	l := len(score)
	if l == 0 {
		return 0, 0
	}

	mean := 0.0
	for _, v := range score {
		mean += v
	}
	mean /= float64(l)

	if l == 1 {
		return mean, 0
	}

	deviation := 0.0
	for _, v := range score {
		deviation += math.Pow(v-mean, 2)
	}

	deviation /= float64(l - 1)

	stdDeviation := math.Sqrt(deviation)
	return mean, stdDeviation
}
