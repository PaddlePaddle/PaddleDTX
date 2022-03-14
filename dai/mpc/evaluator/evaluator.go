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

package evaluator

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/evaluation/validation"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"

	convert "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

var (
	logger = logrus.WithField("module", "mpc.evaluator")
)

// Evaluator performs model evaluation, supports cross-validation, LOO, validation by proportional random division.
// The basic steps of evaluation:
//  Divide the dataset in some way
//  Train the model
//  Validate
//  Calculate the evaluation metric scores with prediction result obtained on the validation set
//  Calculate the average scores for each metric
type Evaluator interface {
	// Start starts model evaluation, that is to segment the training set according to a certain strategy (cross validation, proportional random division),
	//  then start the training-validation process.
	// fileRows is returned by psi.IntersectParts after sample alignment.
	Start(fileRows [][]string) error

	// Stop deletes all the leaners created by Evaluator as well as other objects
	Stop()

	// SaveModel collects the results of the training in the evaluation phase,
	// that is, the model, for Evaluation of Model.
	// If the model is successfully trained,
	// it will trigger the local creation of a Model instance for validation.
	SaveModel(*pbCom.TrainTaskResult) error

	// SavePredictOut collects the prediction results in the evaluation phase.
	// If the prediction result is obtained, it will check how many prediction results have been obtained so far,
	//  and determine whether to start calculating the average scores for each metric.
	SavePredictOut(*pbCom.PredictTaskResult) error
}

type Mpc interface {
	// StartTask starts a specific task of training or prediction
	StartTask(*pbCom.StartTaskRequest) error
	// StopTask stops a specific task of training or prediction
	StopTask(*pbCom.StopTaskRequest) error
}

type Trainer interface {
	// SavePredictAndEvaluatResult saves the training result and evaluation result for a Learner
	// and stops related task.
	SavePredictAndEvaluatResult(result *pbCom.TrainTaskResult)
}

// BinClassValidation performs validation of Binary Classfication case
type BinClassValidation interface {
	// Splitter divides data set into several subsets with some strategies (such as KFolds, LOO),
	// and hold out one subset as validation set and others as training set
	Splitter

	// SetPredictOut sets predicted probabilities from a prediction set to which `idx` refers.
	SetPredictOut(idx int, predProbas []float64) error

	// GetAllPredictOuts returns all prediction results has been stored.
	GetAllPredictOuts() map[int][]string

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

	// SetPredictOut sets prediction outcomes for a prediction set to which `idx` refers.
	SetPredictOut(idx int, yPred []float64) error

	// GetAllPredictOuts returns all prediction results has been stored.
	GetAllPredictOuts() map[int][]float64

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

type evaluator struct {
	id                         string
	caseType                   pbCom.CaseType
	mpc                        Mpc
	trainer                    Trainer
	hosts                      []string
	taskParams                 *pbCom.TaskParams
	evalParams                 *pbCom.EvaluationParams
	evalRule                   pbCom.EvaluationRule
	numValidates               int //number of training-validation
	validatorCaseRegression    RegressionValidation
	validatorCaseBinClass      BinClassValidation
	splitter                   Splitter
	calMetricScoresAndCallback func(index int, res *pbCom.PredictTaskResult)
	predicResults              sync.Map //if obtained prediction result for each validation set
}

// Start starts model evaluation, segment the training set according to a certain strategy (cross validation, proportional random division),
//  then starts the training-validation process.
// fileRows is returned by psi.IntersectParts after sample alignment.
func (e *evaluator) Start(fileRows [][]string) error {
	logger.WithFields(logrus.Fields{"evaluator": e.id}).Infof("start evaluation[caseType:%s, trainParams:%v], and samples are:[%v]", e.caseType, e.taskParams.TrainParams, fileRows[0:2])

	// add ID back to file, because it had been removed after Sample Alignment
	fileRows = e.rebuildFileForEvaluation(fileRows)
	logger.WithFields(logrus.Fields{"evaluator": e.id}).Infof("samples added IDs are:[%v], and total number is[%d]", fileRows[0:10], len(fileRows))

	if e.caseType == pbCom.CaseType_Regression {
		vcr, err := validation.NewRegressionValidation(fileRows, e.taskParams.TrainParams.Label, e.taskParams.TrainParams.IdName)
		if err != nil {
			return errorx.New(errcodes.ErrCodeParam, "evaluator[%s] failed to create RegressionValidation: %s", e.id, err.Error())
		}
		e.validatorCaseRegression = vcr
		e.splitter = vcr
	} else if e.caseType == pbCom.CaseType_BinaryClass {
		vcb, err := validation.NewBinClassValidation(fileRows, e.taskParams.TrainParams.Label, e.taskParams.TrainParams.IdName, e.taskParams.TrainParams.LabelName, "", 0.5)
		if err != nil {
			return errorx.New(errcodes.ErrCodeParam, "evaluator[%s] failed to create BinClassValidation: %s", e.id, err.Error())
		}
		e.validatorCaseBinClass = vcb
		e.splitter = vcb
	}

	return e.splitAndTrain()
}

// rebuildFileForEvaluation adds ID back to file, because it had been removed after Sample Alignment
func (e *evaluator) rebuildFileForEvaluation(f [][]string) [][]string {
	idName := e.taskParams.TrainParams.IdName
	rebuiltF := make([][]string, 0, len(f))
	for i, r := range f {
		newR := make([]string, 0, len(r)+1)
		if i == 0 {
			newR = append(newR, idName)
		} else {
			newR = append(newR, strconv.Itoa(i))
		}
		newR = append(newR, r...)
		rebuiltF = append(rebuiltF, newR)
	}
	return rebuiltF
}

func (e *evaluator) splitAndTrain() error {
	// divide the dataset according to `EvaluationRule`
	logger.WithFields(logrus.Fields{
		"evaluator":      e.id,
		"evaluationRule": e.evalRule,
	}).Infof("start divide the dataset")
	var folds [][][]string
	switch e.evalRule {
	case pbCom.EvaluationRule_ErCrossVal:
		var err error
		if e.evalParams.Cv.Shuffle {
			err = e.splitter.ShuffleKFoldsSplit(int(e.evalParams.Cv.Folds), e.id)
		} else {
			err = e.splitter.KFoldsSplit(int(e.evalParams.Cv.Folds))
		}
		if err != nil {
			logger.WithFields(logrus.Fields{
				"evaluator":      e.id,
				"evaluationRule": e.evalRule,
			}).Warnf("failed to divide the dataset, and error is[%v]", err.Error())
			return errorx.New(errcodes.ErrCodeDataSetSplit, "evaluator[%s] failed to split dataset: %s", e.id, err.Error())
		}

		folds, _ = e.splitter.GetAllFolds()
		e.numValidates = len(folds)

	case pbCom.EvaluationRule_ErLOO:
		err := e.splitter.LooSplit()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"evaluator":      e.id,
				"evaluationRule": e.evalRule,
			}).Warnf("failed to divide the dataset, and error is[%v]", err.Error())
			return errorx.New(errcodes.ErrCodeDataSetSplit, "evaluator[%s] failed to split dataset: %s", e.id, err.Error())
		}

		folds, _ = e.splitter.GetAllFolds()
		e.numValidates = len(folds)

	case pbCom.EvaluationRule_ErRandomSplit:
		err := e.splitter.ShuffleSplit(int(e.evalParams.RandomSplit.PercentLO), e.id)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"evaluator":      e.id,
				"evaluationRule": e.evalRule,
			}).Warnf("failed to divide the dataset, and error is[%v]", err.Error())
			return errorx.New(errcodes.ErrCodeDataSetSplit, "evaluator[%s] failed to split dataset: %s", e.id, err.Error())
		}

		folds, _ = e.splitter.GetAllFolds()
		e.numValidates = 1
	}

	// check if each subset is valid or not
	for i, fold := range folds {
		logger.WithFields(logrus.Fields{
			"evaluator":      e.id,
			"evaluationRule": e.evalRule,
		}).Infof("divided the dataset successfully, and [%d]th fold size is[%d]", i, len(fold)-1)
		if len(fold) <= 1 { // each subset should has enough samples
			return errorx.New(errcodes.ErrCodeDataSetSplit, "evaluator[%s] failed to split dataset which was too small for EvaluationRule[%s]", e.id, e.evalRule.String())
		}
	}

	logger.WithFields(logrus.Fields{"evaluator": e.id}).Infof("start to train models, and number of validations is[%d]", e.numValidates)
	err := e.train()
	if err != nil {
		return err
	}

	return nil
}

func (e *evaluator) train() error {
	for i := 0; i < e.numValidates; i++ {
		// obtain the training sets,
		// if one of them fails to obtain, interrupt the training,
		// which means that the entire evaluation process is interrupted and fails.
		ts, err := e.splitter.GetTrainSet(i)
		if err != nil {
			return errorx.New(errcodes.ErrCodeGetTrainSet, "evaluator[%s] failed to get training set when evaluate model: %s", e.id, err.Error())
		}

		// encapsulate the request for the training task and send the request,
		// message sending is performed concurrently, no response is processed

		var b bytes.Buffer
		w := csv.NewWriter(&b)
		w.WriteAll(ts)
		file := b.Bytes()

		request := e.packParamsForTrain(i, file)
		errSt := e.mpc.StartTask(request)
		if errSt != nil {
			logger.Warnf("evaluator[%s] failed to send StartTaskRequest to start training task[%s], and error is[%s].", e.id, request.TaskID, errSt.Error())
			return errorx.New(errcodes.ErrCodeStartTask, "evaluator[%s] failed to send StartTaskRequest to start training task: %s", e.id, err.Error())
		}
		logger.WithFields(logrus.Fields{"evaluator": e.id}).Infof("sended training task request[%v]", request)
	}

	return nil
}

func (e *evaluator) packParamsForTrain(index int, file []byte) *pbCom.StartTaskRequest {
	taskParams := pbCom.TaskParams{
		Algo:        e.taskParams.Algo,
		TaskType:    e.taskParams.TaskType,
		TrainParams: e.taskParams.TrainParams,
		LivalParams: e.taskParams.LivalParams,
	}

	//if the training task is from Evaluator, the TaskID conforms such form like `{uuid}_{k}_train_Eva`,
	res := pbCom.StartTaskRequest{
		TaskID: fmt.Sprintf("%s_%d_train_Eva", e.id, index),
		File:   file,
		Hosts:  e.hosts,
		Params: &taskParams,
	}

	return &res
}

// Stop deletes all the leaners created by Evaluator as well as other objects
func (e *evaluator) Stop() {
	for i := 0; i < e.numValidates; i++ {
		if _, ok := e.predicResults.Load(i); !ok {
			go func() {
				// if the training task is from Evaluator, the TaskID conforms such form like `{uuid}_{k}_train_Eva`,
				req := &pbCom.StopTaskRequest{
					TaskID: fmt.Sprintf("%s_%d_train_Eva", e.id, i),
					Params: &pbCom.TaskParams{
						TaskType: pbCom.TaskType_LEARN,
					},
				}
				e.mpc.StopTask(req)
				logger.WithFields(logrus.Fields{"evaluator": e.id}).Infof("sended stop training task request[%v]", req)
			}()
		}
	}

}

// SaveModel collects the results of the training in the evaluation phase,
// that is, the model, for Evaluation of Model.
// If the model is successfully trained,
// it will trigger the local creation of a Model instance for validation.
func (e *evaluator) SaveModel(res *pbCom.TrainTaskResult) error {
	logger.WithFields(logrus.Fields{"evaluator": e.id}).Infof("got training result[TrainTaskID:%s, Success:%t, Model:%v] and prepare to do validation", res.TaskID, res.Success, res.Model)
	if !res.Success {
		logger.Warningf("evaluator[%s] got result of training task[%s], but it failed and error is[%s].", e.id, res.TaskID, res.ErrMsg)
		return nil
	}

	modelBytes := res.Model
	model, err := convert.TrainModelsFromBytes(modelBytes)
	if err != nil {
		logger.Warningf("evaluator[%s] failed to convert bytes to TrainModel instance and error is[%s].", e.id, res.ErrMsg)
		return errorx.New(errcodes.ErrCodeParam, "evaluator[%s] failed to convert bytes to TrainModel instance: %s", e.id, err.Error())
	}

	// parse TaskId to get index that indicates the prediction set,
	// then obtain the corresponding prediction set.

	//if the training task is from Evaluator, the TaskID conforms such form like `{uuid}_{k}_train_Eva`.
	ss := strings.SplitN(res.TaskID, "_", 3)
	if len(ss) < 3 {
		logger.Warningf("evaluator[%s] got invalid TaskID[%s].", e.id, res.TaskID)
		return errorx.New(errcodes.ErrCodeParam, "evaluator[%s] got invalid TaskID[%s]", e.id, res.TaskID)
	}
	index, err := strconv.Atoi(ss[1])
	if err != nil {
		logger.Warningf("evaluator[%s] got invalid TaskID[%s].", e.id, res.TaskID)
		return errorx.New(errcodes.ErrCodeParam, "evaluator[%s] got invalid TaskID[%s]", e.id, res.TaskID)
	}

	ps, err := e.splitter.GetPredictSet(index)
	if err != nil {
		logger.Warningf("evaluator[%s] failed to get prediction set when evaluate model and error is[%s].", e.id, err.Error())
		return errorx.New(errcodes.ErrCodeGetTrainSet, "evaluator[%s] failed to get prediction set when evaluate model: %s", e.id, err.Error())
	}

	// encapsulate the request for the prediction task and send the request,
	// no response is processed.

	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.WriteAll(ps)
	file := b.Bytes()

	request := e.packParamsForPredict(index, model, file)
	go func() {
		errSt := e.mpc.StartTask(request)
		if errSt != nil {
			logger.Warningf("evaluator[%s] failed to send StartTaskRequest to start training task[%s], and error is[%s].", e.id, request.TaskID, err.Error())
		}
	}()

	return nil
}

func (e *evaluator) packParamsForPredict(index int, model *pbCom.TrainModels, file []byte) *pbCom.StartTaskRequest {
	// set idName
	model.IdName = e.taskParams.TrainParams.IdName

	taskParams := pbCom.TaskParams{
		Algo:        e.taskParams.Algo,
		TaskType:    pbCom.TaskType_PREDICT,
		ModelParams: model,
	}

	// if the prediction is from Evaluator, the TaskID conforms such form like `{uuid}_{k}_predict_Eva`,
	res := pbCom.StartTaskRequest{
		TaskID: fmt.Sprintf("%s_%d_predict_Eva", e.id, index),
		File:   file,
		Hosts:  e.hosts,
		Params: &taskParams,
	}

	return &res
}

// SavePredictOut collects the prediction results in the evaluation phase.
// If the prediction result is obtained, it will check how many prediction results have been obtained so far,
//  and determine whether to start calculating the average scores for each metric.
func (e *evaluator) SavePredictOut(res *pbCom.PredictTaskResult) error {
	if !res.Success {
		logger.Warningf("evaluator[%s] got result of prediction task[%s], but it failed and error is[%s].", e.id, res.TaskID, res.ErrMsg)
		return nil
	}

	//if the prediction task is from Evaluator, the TaskID conforms such form like `{uuid}_{k}_predict_Eva`.
	ss := strings.SplitN(res.TaskID, "_", 3)
	if len(ss) < 3 {
		return errorx.New(errcodes.ErrCodeParam, "evaluator[%s] got invalid TaskID[%s]", e.id, res.TaskID)
	}

	if ss[0] != e.id {
		return errorx.New(errcodes.ErrCodeParam, "evaluator[%s] got invalid TaskID[%s]", e.id, res.TaskID)
	}

	index, err := strconv.Atoi(ss[1])
	if err != nil {
		return errorx.New(errcodes.ErrCodeParam, "evaluator[%s] got invalid TaskID[%s]", e.id, res.TaskID)
	}

	go e.calMetricScoresAndCallback(index, res)

	return nil
}

func (e *evaluator) calMetricScoresAndCallbackCaseRegression(index int, res *pbCom.PredictTaskResult) {
	// The party who has target tag could get prediction result.
	// And the party who hasn't target tag couldn't get prediction result actually,
	// but during evaluation it cooperates with other parties for model training and prediction.

	if e.taskParams.TrainParams.IsTagPart {

		pred, err := convert.PredictResultFromBytes(res.Outcomes)
		if err != nil {
			logger.Warningf("evaluator[%s] failed to convert bytes to PredictResult instance and error is[%s].", e.id, res.ErrMsg)
			return
		}
		lout := len(pred)
		if lout <= 1 {
			logger.Warningf("evaluator[%s] got invalid prediction result[%d] beacause it was too small.", e.id, index)
			return
		}
		predMap := make(map[string]float64, lout-1)
		for i := 1; i < lout; i++ {
			idValue := pred[i][0]
			labelVal, err := strconv.ParseFloat(pred[i][1], 64)
			if err != nil {
				logger.Warningf("evaluator[%s] got invalid prediction result[%d] beacause label was not type Float64.", e.id, index)
				return
			}
			predMap[idValue] = labelVal
		}

		// set prediction outcomes to validator for further metric scores
		// keep the same order with samples in Validation/Prediction Set

		// to get Validation Set is more quicky than Prediction Set
		validSet, err := e.splitter.GetValidSet(index)
		if err != nil {
			logger.Warningf("evaluator[%s] failed to get validation set[%d] and error is[%s].", e.id, index, res.ErrMsg)
			return
		}
		lvs := len(validSet)
		if lvs <= 1 {
			logger.Warningf("evaluator[%s] got invalid validation set[%d] beacuse it was too small.", e.id, index)
			return
		}
		idIdx := fundIDIndex(validSet, e.taskParams.TrainParams.IdName)
		if idIdx < 0 {
			logger.Warningf("evaluator[%s] got invalid validation set[%d] beacuse it had no ID.", e.id, index)
			return
		}

		yPreds := make([]float64, 0, lvs-1)
		for i := 1; i < lvs; i++ {
			idValue := validSet[i][idIdx]
			yPreds = append(yPreds, predMap[idValue])
		}

		err = e.validatorCaseRegression.SetPredictOut(index, yPreds)
		if err != nil {
			logger.Warningf("evaluator[%s] failed to set prediction outcomes[%d] to validator and error is[%s].", e.id, index, err.Error())
			return
		}
		e.predicResults.LoadOrStore(index, true)

		// check how many prediction results have been obtained so far,
		// and determine whether to start calculating the average scores for each metric.
		posSet := e.validatorCaseRegression.GetAllPredictOuts()
		total := len(posSet)
		if total < e.numValidates {
			logger.Infof("evaluator[%s] successfully set prediction outcomes[%d] to validator and total number is[%d].", e.id, index, total)
			return
		}

		// calculate metric scores and callback trainer
		logger.Infof("evaluator[%s] got enough prediction outcomes, and start to calculate metric scores.", e.id)
		rmses, mean, stdDev, err := e.validatorCaseRegression.GetAllRMSE()
		if err != nil {
			logger.Warningf("evaluator[%s] failed to calculate metric scores and error is[%s].", e.id, err.Error())
			return
		}

		// callback trainer to notify the end of evaluation
		rRMSEs := make(map[int32]float64, len(rmses))
		for k, v := range rmses {
			rRMSEs[int32(k)] = v
		}
		metricScores := &pbCom.RegressionCaseMetricScores{
			CaseType:   pbCom.CaseType_Regression,
			RMSEs:      rRMSEs,
			MeanRMSE:   mean,
			StdDevRMSE: stdDev,
		}
		ems := &pbCom.EvaluationMetricScores{
			Payload: &pbCom.EvaluationMetricScores_RegressionCaseMetricScores{
				RegressionCaseMetricScores: metricScores,
			},
		}
		trainTaskResult := &pbCom.TrainTaskResult{
			TaskID:           e.id,
			Success:          true,
			EvalMetricScores: ems,
		}
		go e.trainer.SavePredictAndEvaluatResult(trainTaskResult)
	} else {
		e.predicResults.LoadOrStore(index, true)

		// check how many prediction results have been obtained so far,
		// and determine whether to stop the evaluation and callback trainer.
		total := 0
		e.predicResults.Range(func(k interface{}, v interface{}) bool {
			total++
			return true
		})
		if total < e.numValidates {
			logger.Infof("evaluator[%s] successfully set prediction outcomes[%d] to validator and total number is[%d].", e.id, index, total)
			return
		}

		// callback trainer to notify the end of evaluation
		trainTaskResult := &pbCom.TrainTaskResult{
			TaskID:  e.id,
			Success: true,
		}
		go e.trainer.SavePredictAndEvaluatResult(trainTaskResult)
	}

}

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
type reportROCAndAUC struct {
	// Roc is represented by a series of points.
	// A point of roc is represented by [3]float64, [FPR, TPR, threshold]([x,y,threshold])
	PointsOnROC [][3]float64
	// AUC is the area under curve ROC.
	AUC float64
}

func (e *evaluator) calMetricScoresAndCallbackCaseBinClass(index int, res *pbCom.PredictTaskResult) {
	// The party who has target tag could get prediction result.
	// And the party who hasn't target tag couldn't get prediction result actually,
	// but during evaluation it cooperates with other parties for model training and prediction.

	if e.taskParams.TrainParams.IsTagPart {

		pred, err := convert.PredictResultFromBytes(res.Outcomes)
		if err != nil {
			logger.Warningf("evaluator[%s] failed to convert bytes to PredictResult instance and error is[%s].", e.id, res.ErrMsg)
			return
		}
		lout := len(pred)
		if lout <= 1 {
			logger.Warningf("evaluator[%s] got invalid prediction result[%d] beacause it was too small.", e.id, index)
			return
		}
		predMap := make(map[string]float64, lout-1)
		for i := 1; i < lout; i++ {
			idValue := pred[i][0]
			labelVal, err := strconv.ParseFloat(pred[i][1], 64)
			if err != nil {
				logger.Warningf("evaluator[%s] got invalid prediction result[%d] beacause label was not type Float64.", e.id, index)
				return
			}
			predMap[idValue] = labelVal
		}

		// set prediction outcomes to validator for further metric scores
		// keep the same order with samples in Validation/Prediction Set

		// to get Validation Set is more quicky than Prediction Set
		validSet, err := e.splitter.GetValidSet(index)
		if err != nil {
			logger.Warningf("evaluator[%s] failed to get validation set[%d] and error is[%s].", e.id, index, res.ErrMsg)
			return
		}
		lvs := len(validSet)
		if lvs <= 1 {
			logger.Warningf("evaluator[%s] got invalid validation set[%d] beacuse it was too small.", e.id, index)
			return
		}
		idIdx := fundIDIndex(validSet, e.taskParams.TrainParams.IdName)
		if idIdx < 0 {
			logger.Warningf("evaluator[%s] got invalid validation set[%d] beacuse it had no ID.", e.id, index)
			return
		}

		predProba := make([]float64, 0, lvs-1)
		for i := 1; i < lvs; i++ {
			idValue := validSet[i][idIdx]
			predProba = append(predProba, predMap[idValue])
		}

		err = e.validatorCaseBinClass.SetPredictOut(index, predProba)
		if err != nil {
			logger.Warningf("evaluator[%s] failed to set prediction outcomes[%d] to validator and error is[%s].", e.id, index, err.Error())
			return
		}
		e.predicResults.LoadOrStore(index, true)

		// check how many prediction results have been obtained so far,
		// and determine whether to start calculating the average scores for each metric.
		posSet := e.validatorCaseBinClass.GetAllPredictOuts()
		total := len(posSet)
		if total < e.numValidates {
			logger.Infof("evaluator[%s] successfully set prediction outcomes[%d] to validator and total number is[%d].", e.id, index, total)
			return
		}

		// calculate metric scores and callback trainer
		logger.Infof("evaluator[%s] got enough prediction outcomes, and start to calculate metric scores.", e.id)
		reports, err := e.validatorCaseBinClass.GetOverallReport()
		if err != nil {
			logger.Warningf("evaluator[%s] failed to calculate metric scores and error is[%s].", e.id, err.Error())
			return
		}
		repsRocAuc, err := e.validatorCaseBinClass.GetAllROCAndAUC()
		if err != nil {
			logger.Warningf("evaluator[%s] failed to calculate AUC scores and error is[%s].", e.id, err.Error())
			return
		}

		// callback trainer to notify the end of evaluation
		lreps := len(reports)
		metricsPerFold := make(map[int32]*pbCom.BinaryClassCaseMetricScores_MetricsPerFold, lreps)
		var avgAccuracy float64
		var avgPrecision float64
		var avgRecall float64
		var avgF1Score float64
		var countP int

		for k, r := range reports {
			var report confusionMatrixSummary
			json.Unmarshal(r, &report)
			// Determine whether there are positive samples
			if metr, ok := report.Metrics[e.taskParams.TrainParams.LabelName]; ok {
				countP++
				metricsPerFold[int32(k)] = &pbCom.BinaryClassCaseMetricScores_MetricsPerFold{
					Accuracy:  report.Accuracy,
					Precision: metr.Precision,
					Recall:    metr.Recall,
					F1Score:   metr.F1Score,
				}
				avgPrecision += metr.Precision
				avgRecall += metr.Recall
				avgF1Score += metr.F1Score
			} else {
				metricsPerFold[int32(k)] = &pbCom.BinaryClassCaseMetricScores_MetricsPerFold{
					Accuracy: report.Accuracy,
				}
			}

			avgAccuracy += report.Accuracy

		}
		avgAccuracy /= float64(lreps)
		if countP > 0 {
			avgPrecision /= float64(countP)
			avgRecall /= float64(countP)
			avgF1Score /= float64(countP)
		}

		var avgAUC float64
		for k, r := range repsRocAuc {
			var report reportROCAndAUC
			json.Unmarshal(r, &report)
			roc := make([]*pbCom.BinaryClassCaseMetricScores_Point, 0, len(report.PointsOnROC))
			for _, p := range report.PointsOnROC {
				roc = append(roc, &pbCom.BinaryClassCaseMetricScores_Point{P: []float64{p[0], p[1], p[2]}})
			}
			metricsPerFold[int32(k)].AUC = report.AUC
			metricsPerFold[int32(k)].ROC = roc

			avgAUC += report.AUC
		}
		avgAUC /= float64(lreps)

		metricScores := &pbCom.BinaryClassCaseMetricScores{
			CaseType:       pbCom.CaseType_BinaryClass,
			AvgAccuracy:    avgAccuracy,
			AvgPrecision:   avgPrecision,
			AvgRecall:      avgRecall,
			AvgF1Score:     avgF1Score,
			AvgAUC:         avgAUC,
			MetricsPerFold: metricsPerFold,
		}
		ems := &pbCom.EvaluationMetricScores{
			Payload: &pbCom.EvaluationMetricScores_BinaryClassCaseMetricScores{
				BinaryClassCaseMetricScores: metricScores,
			},
		}
		trainTaskResult := &pbCom.TrainTaskResult{
			TaskID:           e.id,
			Success:          true,
			EvalMetricScores: ems,
		}
		go e.trainer.SavePredictAndEvaluatResult(trainTaskResult)
	} else {
		e.predicResults.LoadOrStore(index, true)

		// check how many prediction results have been obtained so far,
		// and determine whether to stop the evaluation and callback trainer.
		total := 0
		e.predicResults.Range(func(k interface{}, v interface{}) bool {
			total++
			return true
		})
		if total < e.numValidates {
			logger.Infof("evaluator[%s] successfully set prediction outcomes[%d] to validator and total number is[%d].", e.id, index, total)
			return
		}

		// callback trainer to notify the end of evaluation
		trainTaskResult := &pbCom.TrainTaskResult{
			TaskID:  e.id,
			Success: true,
		}
		go e.trainer.SavePredictAndEvaluatResult(trainTaskResult)
	}

}

func fundIDIndex(fileRows [][]string, idName string) int {
	// find where the IDs are
	idx := -1
	for i, v := range fileRows[0] {
		if v == idName {
			idx = i
			break
		}
	}
	return idx
}

func NewEvaluator(req *pbCom.StartTaskRequest, mpc Mpc, trainer Trainer) (Evaluator, error) {
	getCaseType := func(algo pbCom.Algorithm) (pbCom.CaseType, error) {
		switch algo {
		case pbCom.Algorithm_LINEAR_REGRESSION_VL:
			return pbCom.CaseType_Regression, nil
		case pbCom.Algorithm_LOGIC_REGRESSION_VL:
			return pbCom.CaseType_BinaryClass, nil
		default:
			return 0, errorx.New(errcodes.ErrCodeParam, "unknown algorithm: %s", algo.String())
		}
	}

	// check CaseType
	caseType, err := getCaseType(req.Params.Algo)
	if err != nil {
		return nil, err
	}

	// check evaluation params and Evaluation Rule
	if req.Params.EvalParams == nil || !req.Params.EvalParams.Enable {
		return nil, errorx.New(errcodes.ErrCodeParam, "invalid evaluation params")
	}
	switch req.Params.EvalParams.EvalRule {
	case pbCom.EvaluationRule_ErRandomSplit:
		if req.Params.EvalParams.RandomSplit == nil {
			return nil, errorx.New(errcodes.ErrCodeParam, "invalid evaluation rule: %s", req.Params.EvalParams.EvalRule)
		}
	case pbCom.EvaluationRule_ErCrossVal:
		if req.Params.EvalParams.Cv == nil {
			return nil, errorx.New(errcodes.ErrCodeParam, "invalid evaluation rule: %s", req.Params.EvalParams.EvalRule)
		}
	case pbCom.EvaluationRule_ErLOO:
	default:
		return nil, errorx.New(errcodes.ErrCodeParam, "unknown evaluation rule: %s", req.Params.EvalParams.EvalRule)
	}

	e := &evaluator{
		id:         req.TaskID,
		caseType:   caseType,
		mpc:        mpc,
		trainer:    trainer,
		hosts:      req.Hosts,
		taskParams: req.Params,
		evalParams: req.Params.EvalParams,
		evalRule:   req.Params.EvalParams.EvalRule,
	}
	if caseType == pbCom.CaseType_Regression {
		e.calMetricScoresAndCallback = e.calMetricScoresAndCallbackCaseRegression
	} else {
		e.calMetricScoresAndCallback = e.calMetricScoresAndCallbackCaseBinClass
	}
	return e, nil
}
