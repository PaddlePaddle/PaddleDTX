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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"text/tabwriter"
)

// ConfusionMatrix is a nested map of actual and predicted class counts
type ConfusionMatrix map[string]map[string]int

type metricsRelatedConfusionMatrix struct {
	TP        float64
	FP        float64
	FN        float64
	TN        float64
	Precision float64
	Recall    float64
	F1Score   float64
}

type confusionMatrixSummary struct {
	Metrics  map[string]metricsRelatedConfusionMatrix
	Accuracy float64
}

// NewConfusionMatrix builds a ConfusionMatrix from a set of real class value (`realClasses')
// and a set of predicted class value (`predClasses').
// The same index of realClasses and predClasses must refer to the same sample
func NewConfusionMatrix(realClasses []string, predClasses []string) (ConfusionMatrix, error) {
	numRC := len(realClasses)
	numPC := len(predClasses)
	if numRC != numPC {
		return ConfusionMatrix{}, errors.New("RealClasses and PredClasses not match")
	}
	if numRC == 0 {
		return ConfusionMatrix{}, errors.New("RealClasses and PredClasses are empty")
	}

	cm := make(ConfusionMatrix)
	for i := 0; i < numRC; i++ {
		rc := realClasses[i]
		pc := predClasses[i]
		if _, ok := cm[rc]; ok {
			if _, ok := cm[rc][pc]; ok {
				cm[rc][pc] += 1
			} else {
				cm[rc][pc] = 1
			}
		} else {
			cm[rc] = make(map[string]int)
			cm[rc][pc] = 1
		}
	}

	return cm, nil
}

// GetTruePositives returns TP which means the number of times an entry is
// predicted correctly in the ConfusionMatrix.
func (cm ConfusionMatrix) GetTruePositives(class string) (float64, error) {
	if _, ok := cm[class]; !ok {
		return 0, errors.New("Unknown class value[" + class + "]")
	}
	return float64(cm[class][class]), nil
}

// GetFalsePositives returns FP which means the number of times an entry is
// incorrectly predicted in the ConfusionMatrix.
func (cm ConfusionMatrix) GetFalsePositives(class string) (float64, error) {
	if _, ok := cm[class]; !ok {
		return 0, errors.New("Unknown class value[" + class + "]")
	}

	ret := 0.0
	for k := range cm {
		if k == class {
			continue
		}
		if _, ok := cm[k][class]; ok {
			ret += float64(cm[k][class])
		}
	}
	return ret, nil
}

// GetFalseNegatives returns FN which means the number of times an entry is
// incorrectly predicted as something other than the given class.
func (cm ConfusionMatrix) GetFalseNegatives(class string) (float64, error) {
	if _, ok := cm[class]; !ok {
		return 0, errors.New("Unknown class value[" + class + "]")
	}

	ret := 0.0
	for k := range cm[class] {
		if k == class {
			continue
		}
		ret += float64(cm[class][k])
	}

	return ret, nil
}

// GetTrueNegatives returns TN which means the number of times an entry is
// correctly predicted as something other than the given class.
func (cm ConfusionMatrix) GetTrueNegatives(class string) (float64, error) {
	if _, ok := cm[class]; !ok {
		return 0, errors.New("Unknown class value[" + class + "]")
	}

	ret := 0.0
	for k := range cm {
		if k == class {
			continue
		}
		for l := range cm[k] {
			if l == class {
				continue
			}
			ret += float64(cm[k][l])
		}
	}

	return ret, nil
}

// GetPrecision returns Precision which means the correctly predicted fraction of the total predictions
// for a given class.
func (cm ConfusionMatrix) GetPrecision(class string) (float64, error) {
	truePositives, err := cm.GetTruePositives(class)
	if err != nil {
		return 0, err
	}
	falsePositives, err := cm.GetFalsePositives(class)
	if err != nil {
		return 0, err
	}

	if truePositives == 0 && falsePositives == 0 {
		return 0, nil
	}

	return truePositives / (truePositives + falsePositives), nil
}

// GetRecall returns Recall which means the fraction of the total occurrences of a
// given class which were predicted.
func (cm ConfusionMatrix) GetRecall(class string) (float64, error) {
	truePositives, err := cm.GetTruePositives(class)
	if err != nil {
		return 0, err
	}
	falseNegatives, err := cm.GetFalseNegatives(class)
	if err != nil {
		return 0, err
	}
	return truePositives / (truePositives + falseNegatives), nil
}

// GetTPR returns TPR which means the fraction of the total occurrences of a
// given class which were predicted.
func (cm ConfusionMatrix) GetTPR(class string) (float64, error) {
	return cm.GetRecall(class)
}

// GetFPR returns FPR which means the fraction of the misclassified samples which were not given class actually
// in all samples which were not given class actually.
func (cm ConfusionMatrix) GetFPR(class string) (float64, error) {
	falsePositives, err := cm.GetFalsePositives(class)
	if err != nil {
		return 0, err
	}
	trueNegatives, err := cm.GetTrueNegatives(class)
	if err != nil {
		return 0, err
	}
	return falsePositives / (falsePositives + trueNegatives), nil
}

// GetF1Score computes the harmonic mean of Precision and Recall
// (equivalently called F-measure).
func (cm ConfusionMatrix) GetF1Score(class string) (float64, error) {
	precision, err := cm.GetPrecision(class)
	if err != nil {
		return 0, err
	}
	recall, err := cm.GetRecall(class)
	if err != nil {
		return 0, err
	}

	if precision == 0 && recall == 0 {
		return 0, nil
	}

	return 2 * (precision * recall) / (precision + recall), nil
}

// GetAccuracy computes the overall classification accuracy.
// That is (number of correctly classified instances) / total instances
func (cm ConfusionMatrix) GetAccuracy() float64 {
	correct := 0
	total := 0
	for i := range cm {
		for j := range cm[i] {
			if i == j {
				correct += cm[i][j]
			}
			total += cm[i][j]
		}
	}
	return float64(correct) / float64(total)
}

func (cm ConfusionMatrix) summary() confusionMatrixSummary {
	metrics := make(map[string]metricsRelatedConfusionMatrix, len(cm))

	for k := range cm {
		tp, _ := cm.GetTruePositives(k)
		fp, _ := cm.GetFalsePositives(k)
		tn, _ := cm.GetTrueNegatives(k)
		fn, _ := cm.GetFalseNegatives(k)
		prec, _ := cm.GetPrecision(k)
		rec, _ := cm.GetRecall(k)
		f1, _ := cm.GetF1Score(k)

		metrics[k] = metricsRelatedConfusionMatrix{
			TP:        tp,
			FP:        fp,
			FN:        fn,
			TN:        tn,
			Precision: prec,
			Recall:    rec,
			F1Score:   f1,
		}
	}
	acc := cm.GetAccuracy()

	return confusionMatrixSummary{Metrics: metrics, Accuracy: acc}
}

// Summary returns a table of precision, recall, f1, true positive,
// false positive, true negatives and false negatives for each class, and accuracy.
func (cm ConfusionMatrix) Summary() string {
	sum := cm.summary()

	var buffer bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&buffer, 0, 8, 0, '\t', 0)

	fmt.Fprintln(w, "Real Class\tTrue Positives\tFalse Positives\tTrue Negatives\tFalse Negatives\tPrecision\tRecall\tF1 Score")
	fmt.Fprintln(w, "---------------\t--------------\t---------------\t---------------\t--------------\t---------\t---------\t--------")
	for k, v := range sum.Metrics {
		fmt.Fprintf(w, "%s\t%.0f\t%.0f\t%.0f\t%.0f\t%.4f\t%.4f\t%.4f\n", k, v.TP, v.FP, v.TN, v.FN, v.Precision, v.Recall, v.F1Score)
	}
	w.Flush()
	buffer.WriteString(fmt.Sprintf("Overall accuracy: %.4f\n", sum.Accuracy))

	return buffer.String()
}

// SummaryAsJSON returns a json bytes of precision, recall, f1, true positive,
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
// NO and Yes are classes
func (cm ConfusionMatrix) SummaryAsJSON() ([]byte, error) {
	sum := cm.summary()
	return json.Marshal(&sum)
}

// String returns a human-readable version of the ConfusionMatrix.
func (cm ConfusionMatrix) String() string {
	var buffer bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&buffer, 0, 8, 0, '\t', 0)

	numClass := len(cm)
	realCs := make(map[string]bool, numClass)
	fmt.Fprintf(w, "Real Class\t")
	for k := range cm {
		fmt.Fprintf(w, "%s\t", k)
		realCs[k] = true
	}
	for _, v := range cm {
		for k := range v {
			if _, ok := realCs[k]; !ok {
				realCs[k] = false
				fmt.Fprintf(w, "%s\t", k)
			}
		}
	}

	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "----------\t")
	for k := range realCs {
		for t := 0; t < len(k); t++ {
			fmt.Fprintf(w, "-")
		}
		fmt.Fprintf(w, "\t")
	}
	fmt.Fprintf(w, "\n")

	for k, real := range realCs {
		if real {
			fmt.Fprintf(w, "%s\t", k)
			for k2 := range realCs {
				if n, ok := cm[k][k2]; ok {
					fmt.Fprintf(w, "%d\t", n)
				} else {
					fmt.Fprintf(w, "%d\t", 0)
				}
			}
			fmt.Fprintf(w, "\n")
		}
	}
	w.Flush()

	return buffer.String()
}

/*
* GetROC
* Compute Receiver operating characteristic(roc).
* PARAMS:
*   - realClasses []string: Real labels of sample set
*	- predValues  []float64: Predicted Target scores, which corresponds to the sample in 'realClasses',
                             are probability estimates of the next param 'class'.
*	- class       string: The positive label in roc.
* RERURNS:
*	- [][3]float64:  a series of points on roc, the point is represented as [FPR, TPR, threshold]([x,y,threshold])
* 					 FPR = FP / (TN + FP),  TPR = TP / (TP + FN)
*	- error: nil if succeed, error if fail
*/
func GetROC(realClasses []string, predValues []float64, class string) ([][3]float64, error) {
	if len(realClasses) != len(predValues) {
		return nil, errors.New("The number of samples should be consistent with the predicted number")
	}

	// Use the predicted value as the threshold to calculate each indicator
	stepThresholds := make([]float64, len(predValues))
	copy(stepThresholds, predValues)
	sort.Sort(sort.Reverse(sort.Float64Slice(stepThresholds)))

	ret := make([][3]float64, 0, len(stepThresholds))
	// ret[0] represents no instances being predicted positive.
	// FPR and TPR are zero, and the threshold is arbitrarily set to `max(predValue)+1`
	ret = append(ret, [3]float64{0, 0, stepThresholds[0]+1})

	// calculate each point of the roc, according to the threshold array
	for _, threshold := range stepThresholds {
		predTruePNum := 0  //TP
		predFalsePNum := 0 //FP
		realPNum := 0      //TP+FN
		for i := 0; i < len(realClasses); i++ {
			if class == realClasses[i] {
				realPNum += 1
				if predValues[i] >= threshold {
					predTruePNum += 1
				}
			} else {
				if predValues[i] >= threshold {
					predFalsePNum += 1
				}
			}
		}

		var FPR, TPR float64
		// FPR = FP/(TN+FP) = TP/[sum(samples)-(TP+FN)],  TPR = TP/(TP+FN)
		if realPNum == len(realClasses) {
			TPR = float64(predTruePNum) / float64(realPNum)
			FPR = 1
		} else if realPNum == 0 {
			TPR = 1
			FPR = float64(predFalsePNum) / float64(len(realClasses)-realPNum)
		} else {
			TPR = float64(predTruePNum) / float64(realPNum)
			FPR = float64(predFalsePNum) / float64(len(realClasses)-realPNum)
		}
		point := [3]float64{FPR, TPR, threshold}
		ret = append(ret, point)
	}


	return ret, nil
}

// GetCoordinates get the abscissa and ordinate of the point represented as [Xi,Yi,Tag], return [[Xi,Yi],...]
func GetCoordinates(points [][3]float64) [][2]float64 {
	ret := make([][2]float64, 0, len(points))
	for _, point := range points {
		ret = append(ret, [2]float64{point[0], point[1]})
	}
	return ret
}

// GetAUC returns auc of the roc which is expressed by a series of points,
// points is sorted by FPR in monotonic increasing or monotonic order.
func GetAUC(points [][2]float64) (float64, error) {
	var auc float64
	if len(points) < 2 {
		return 0, errors.New("The number of points needs to be greater than 1")
	}

	// TA(trapezoid area)i = 1/2 * [Yi+Y(i+1)] * [X(i+1)-X(i)]
	// AUC = SUM(TAi)
	var flag bool
	for i := 0; i < len(points)-1; i++ {
		if i == 0 {
			flag = points[i][0] > points[i+1][0]
		}
		width := points[i][0] - points[i+1][0]
		if (width < 0 && flag) || (width > 0 && !flag) {
			return 0, errors.New("The points' FPRs must be either monotonic increasing or monotonic")
		}
		height := (points[i][1] + points[i+1][1]) / 2
		auc = auc + width*height
	}
	if auc == 0 {
		return auc, nil
	}
	if flag {
		return auc, nil
	} else {
		return -auc, nil
	}

}
