package livaluator

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/evaluation/validation"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"

	convert "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
	pbLinearRegVl "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc/learners/linear_reg_vl"
	pbLogicRegVl "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc/learners/logic_reg_vl"
	"github.com/golang/protobuf/proto"
)

var (
	logger = logrus.WithField("module", "mpc.livaluator")
)

// LiveEvaluator performs staged evaluation during training.
// The basic steps of LiveEvaluator:
//  Divide the dataset in the way of proportional random division.
//  Initiate a learner for evaluation with training part.
//  Train the model, and pause training when the pause round is reached,
//  and instantiate the staged model for validation,
//  then, calculate the evaluation metric scores with prediction result obtained on the validation set.
//  Repeat Train-Pause-Validate until the stop signal is received.
type LiveEvaluator interface {
	// Trigger triggers model evaluation.
	// The parameter contains two types of messages.
	// One is to set the learner for evaluation with training set and start it.
	// The other is to drive the learner to continue training. When the conditions are met(reaching pause round),
	// stop training and instantiate the model for validation.
	Trigger(*pb.LiveEvaluationTriggerMsg) error

	// Stop deletes all the learners created by LiveEvaluator as well as other objects
	Stop()

	// SaveModel collects the results of the training in the evaluation phase,
	// that is, the model, for LiveEvaluation of Model.
	// If the model is successfully trained,
	// it will trigger the local creation of a Model instance for validation.
	SaveModel(*pbCom.TrainTaskResult) error

	// SavePredictOut collects the prediction results in the evaluation phase.
	// If the prediction result is obtained, it will start calculating metric scores,
	// then report the results to visualization system.
	SavePredictOut(*pbCom.PredictTaskResult) error
}

type Mpc interface {
	// StartTask starts a specific task of training or prediction
	StartTask(*pbCom.StartTaskRequest) error
	// StopTask stops a specific task of training or prediction
	StopTask(*pbCom.StopTaskRequest) error
	// Train to train out a model
	Train(*pb.TrainRequest) (*pb.TrainResponse, error)
}

type BinClassValidation interface {
	// Splitter divides data set into several subsets with some strategies (such as KFolds, LOO),
	// and hold out one subset as validation set and others as training set
	Splitter

	// SetPredictOut sets predicted probabilities from a prediction set to which `idx` refers.
	SetPredictOut(idx int, predProbas []float64) error

	// GetReport returns a json bytes of precision, recall, f1, true positive,
	// false positive, true negatives and false negatives for each class, and accuracy.
	GetReport(idx int) ([]byte, error)

	// GetROCAndAUC returns a json bytes of roc's points and auc.
	GetROCAndAUC(idx int) ([]byte, error)
}

// RegressionValidation performs validation of Regression case
type RegressionValidation interface {
	// Splitter divides data set into several subsets with some strategies (such as KFolds, LOO),
	// and hold out one subset as validation set and others as training set
	Splitter

	// SetPredictOut sets prediction outcomes for a prediction set to which `idx` refers.
	SetPredictOut(idx int, yPred []float64) error

	// GetRMSE returns RMSE over the validation set to which `idx` refers.
	GetRMSE(idx int) (float64, error)
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

type liveEvaluator struct {
	id                         string
	algo                       pbCom.Algorithm
	caseType                   pbCom.CaseType
	mpc                        Mpc
	hosts                      []string
	taskParams                 *pbCom.TaskParams
	livalParams                *pbCom.LiveEvaluationParams
	evalRule                   pbCom.EvaluationRule
	validatorCaseRegression    RegressionValidation
	validatorCaseBinClass      BinClassValidation
	splitter                   Splitter
	calMetricScoresAndCallback func(res *pbCom.PredictTaskResult)
	runLearner                 func(*pb.LiveEvaluationTriggerMsg) error
	learnerID                  string                   // indicates the evaluated learner(created by common training task)
	evalLearnerID              string                   // indicates the learner used for evaluation(created by Live Evaluator)
	pauseRound                 uint64                   // pause round of EvalLearner, set when LiveEvaluator is Triggered
	callbackPayload            []byte                   // parameters that drives the evaluated learner to continue training, set when LiveEvaluator is Triggered
	mutex                      sync.Mutex               // mutex makes sure that LiveEvaluator is triggered only once for each `PauseRound`
	trainRes                   *pbCom.TrainTaskResult   // training task result for each `PauseRound`
	predicRes                  *pbCom.PredictTaskResult // prediction task result for each `PauseRound`
}

// Trigger triggers model evaluation.
// The parameter contains two types of messages.
// One is to set the learner for evaluation with training set and start it.
// The other is to drive the learner to continue training. When the conditions are met(reaching pause round),
// stop training and instantiate the model for validation.
func (le *liveEvaluator) Trigger(msg *pb.LiveEvaluationTriggerMsg) error {
	le.mutex.Lock()
	defer le.mutex.Unlock()

	if le.pauseRound >= msg.PauseRound { // has been or being triggered
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] has been triggered more than once for pause round[%d]", le.id, msg.PauseRound)
	}

	logger.Infof("live evaluator[%s] is triggered for pause round[%d]", le.id, msg.PauseRound)
	le.pauseRound = msg.PauseRound
	le.callbackPayload = msg.CallbackPayload
	le.trainRes = nil
	le.predicRes = nil

	if msg.Type == pb.TriggerMsgType_MsgSetAndRun {

		// start a training task to create a learner for evaluation
		err := le.newLearner()
		if err != nil {
			return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to a learner for evaluation: %s", le.id, err.Error())
		}

		// set the learner for evaluation with training set and start it
		err = le.runLearner(msg)
		if err != nil {
			logger.Warnf("live evaluator[%s] failed to run learner[%s], and error is[%s].",
				le.id, le.evalLearnerID, err.Error())
			return err
		}
		logger.Infof("live evaluator[%s] run learner[%s] successfully.",
			le.id, le.evalLearnerID)
	} else { // msg.Type == pb.TriggerMsgType_MsgGoOn
		if (le.validatorCaseBinClass == nil && le.validatorCaseRegression == nil) || le.splitter == nil {
			return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] should be set with validation way and splitter first", le.id)
		}
		resp, err := le.mpc.Train(&pb.TrainRequest{
			TaskID:  le.evalLearnerID,
			Algo:    le.algo,
			Payload: msg.Payload,
		})
		if err != nil {
			logger.Warnf("live evaluator[%s] failed to run learner[%s], and error is[%s].",
				le.id, le.evalLearnerID, err.Error())
			return err
		}
		logger.Infof("live evaluator[%s] run learner[%s] successfully and response is[%v].",
			le.id, le.evalLearnerID, resp)
	}
	return nil
}

// splitAndGetTrainSetVL segments the aligned training set, and returns the training set for evaluation
func (le *liveEvaluator) splitAndGetTrainSetVL(msg *pb.LiveEvaluationTriggerMsg) ([][]string, error) {
	// get training set from message sent by learner after Sample Alignment
	var ts [][]string
	for _, r := range msg.TrainSet {
		ts = append(ts, r.GetRow())
	}
	fileRows := le.rebuildFileForShuffle(ts)

	// segment the training set
	if le.caseType == pbCom.CaseType_Regression {
		vcr, err := validation.NewRegressionValidation(fileRows, le.taskParams.TrainParams.Label, le.taskParams.TrainParams.IdName)
		if err != nil {
			return [][]string{}, errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to create RegressionValidation: %s", le.id, err.Error())
		}
		le.validatorCaseRegression = vcr
		le.splitter = vcr
	} else if le.caseType == pbCom.CaseType_BinaryClass {
		vcb, err := validation.NewBinClassValidation(fileRows, le.taskParams.TrainParams.Label, le.taskParams.TrainParams.IdName, le.taskParams.TrainParams.LabelName, "", 0.5)
		if err != nil {
			return [][]string{}, errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to create BinClassValidation: %s", le.id, err.Error())
		}
		le.validatorCaseBinClass = vcb
		le.splitter = vcb
	}

	err := le.splitter.ShuffleSplit(int(le.livalParams.RandomSplit.PercentLO), le.id)
	if err != nil {
		logger.Warnf("live evaluator[%s] failed to divide the dataset, and error is[%s].",
			le.id, err.Error())
		return [][]string{}, errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to divide the dataset, and error is: %s", le.id, err.Error())
	}
	folds, _ := le.splitter.GetAllFolds()

	// check if each subset is valid or not
	for _, fold := range folds {
		if len(fold) <= 1 { // each subset should has enough samples
			return [][]string{}, errorx.New(errcodes.ErrCodeDataSetSplit, "evaluator[%s] failed to split dataset which was too small for EvaluationRule_RandomSplit", le.id)
		}
	}
	return folds[1], nil
}

// newLearner starts a training task to create a learner for evaluation
func (le *liveEvaluator) newLearner() error {
	// create a learner without samples,
	// and this learner won't run automatically until a specific instruction is received.
	taskParams := pbCom.TaskParams{
		Algo:        le.taskParams.Algo,
		TaskType:    le.taskParams.TaskType,
		TrainParams: le.taskParams.TrainParams,
	}
	res := pbCom.StartTaskRequest{
		TaskID: le.evalLearnerID,
		Hosts:  le.hosts,
		Params: &taskParams,
	}

	err := le.mpc.StartTask(&res)
	if err != nil {
		logger.Warnf("live evaluator[%s] failed to start a training task to create learner[%s] for evaluation, and error is[%s].",
			le.id, le.evalLearnerID, err.Error())
		return err
	}
	logger.Infof("live evaluator[%s] started a training task to create learner[%s] for evaluation successfully.",
		le.id, le.evalLearnerID)

	return nil
}

func (le *liveEvaluator) runLinearRegVL(msg *pb.LiveEvaluationTriggerMsg) error {
	m := &pbLinearRegVl.Message{}
	err := proto.Unmarshal(msg.Payload, m)
	if err != nil {
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to Unmarshal payload: %s", le.id, err.Error())
	}

	ts, err := le.splitAndGetTrainSetVL(msg)
	if err != nil {
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to split LiveEvaluationTriggerMsg.TrainSet to get training set for evaluation: %s", le.id, err.Error())
	}

	// remove IDs
	ts = le.removeIDInFile(ts)

	// reset `TrainSet` in message
	var mts []*pbCom.TrainTaskResult_FileRow
	for _, r := range ts {
		mts = append(mts, &pbCom.TrainTaskResult_FileRow{Row: r})
	}
	m.TrainSet = mts

	pl, err := proto.Marshal(m)
	if err != nil {
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to Marshal message: %s", le.id, err.Error())
	}

	resp, err := le.mpc.Train(&pb.TrainRequest{
		TaskID:  le.evalLearnerID,
		Algo:    le.algo,
		Payload: pl,
	})
	if err != nil {
		logger.Warnf("live evaluator[%s] failed to run learner[%s], and error is[%s].",
			le.id, le.evalLearnerID, err.Error())
		return err
	}
	logger.Infof("live evaluator[%s] run learner[%s] successfully and response is[%v].",
		le.id, le.evalLearnerID, resp)

	return nil
}

func (le *liveEvaluator) runLogicRegVL(msg *pb.LiveEvaluationTriggerMsg) error {
	// get training set from message sent by learner after Sample Alignment
	m := &pbLogicRegVl.Message{}
	err := proto.Unmarshal(msg.Payload, m)
	if err != nil {
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to Unmarshal payload: %s", le.id, err.Error())
	}

	ts, err := le.splitAndGetTrainSetVL(msg)
	if err != nil {
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to split LiveEvaluationTriggerMsg.TrainSet to get training set for evaluation: %s", le.id, err.Error())
	}

	// remove IDs
	ts = le.removeIDInFile(ts)

	// reset `TrainSet` in message
	var mts []*pbCom.TrainTaskResult_FileRow
	for _, r := range ts {
		mts = append(mts, &pbCom.TrainTaskResult_FileRow{Row: r})
	}
	m.TrainSet = mts

	pl, err := proto.Marshal(m)
	if err != nil {
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to Marshal message: %s", le.id, err.Error())
	}

	resp, err := le.mpc.Train(&pb.TrainRequest{
		TaskID:  le.evalLearnerID,
		Algo:    le.algo,
		Payload: pl,
	})
	if err != nil {
		logger.Warnf("live evaluator[%s] failed to run learner[%s], and error is[%s].",
			le.id, le.evalLearnerID, err.Error())
		return err
	}
	logger.Infof("live evaluator[%s] run learner[%s] successfully and response is[%v].",
		le.id, le.evalLearnerID, resp)

	return nil
}

// rebuildFileForEvaluation adds ID back to file in order to keep the same order when shuffle samples for parties,
// because it had been removed after Sample Alignment
func (le *liveEvaluator) rebuildFileForShuffle(f [][]string) [][]string {
	idName := le.taskParams.TrainParams.IdName
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

// removeIDInFile removes ID in the file, because it is useless after Sample Alignment
func (le *liveEvaluator) removeIDInFile(f [][]string) [][]string {
	rebuiltF := make([][]string, 0, len(f))
	for _, r := range f {
		newR := make([]string, 0, len(r)-1)
		newR = append(newR, r[1:]...) // ID is in the first place for each row, see `func rebuildFileForShuffle`
		rebuiltF = append(rebuiltF, newR)
	}
	return rebuiltF
}

// Stop deletes all the learners created by LiveEvaluator as well as other objects
func (le *liveEvaluator) Stop() {
	go func() {
		// if the training task is from Live Evaluator, the TaskID conforms such form like `{uuid}_{k}_train_LEv`,
		req := &pbCom.StopTaskRequest{
			TaskID: le.evalLearnerID,
			Params: &pbCom.TaskParams{
				TaskType: pbCom.TaskType_LEARN,
			},
		}
		le.mpc.StopTask(req)
		logger.WithFields(logrus.Fields{"live evaluator": le.id}).Infof("sended stop training task request[%v]", req)
	}()
	return
}

// SaveModel collects the results of the training in the evaluation phase,
// that is, the model, for LiveEvaluation of Model.
// If the model is successfully trained,
// it will trigger the local creation of a Model instance for validation.
func (le *liveEvaluator) SaveModel(res *pbCom.TrainTaskResult) error {
	logger.WithFields(logrus.Fields{"live evaluator": le.id}).Infof("got training result[TrainTaskID:%s, Success:%t, Model:%v] and prepare to do validation", res.TaskID, res.Success, res.Model)

	le.mutex.Lock()
	defer le.mutex.Unlock()
	if le.trainRes != nil {
		logger.Warningf("live evaluator[%s] received training result more than once for pause round[%d]", le.id, le.pauseRound)
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] received training result more than once for pause round[%d]", le.id, le.pauseRound)

	}
	le.trainRes = res

	if !res.Success {
		logger.Warningf("live evaluator[%s] got result of training task[%s], but it failed and error is[%s].", le.id, res.TaskID, res.ErrMsg)
		return nil
	}

	modelBytes := res.Model
	model, err := convert.TrainModelsFromBytes(modelBytes)
	if err != nil {
		logger.Warningf("live evaluator[%s] failed to convert bytes to TrainModel instance and error is[%s].", le.id, res.ErrMsg)
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] failed to convert bytes to TrainModel instance: %s", le.id, err.Error())
	}

	// get first set as validation set because the way of validation is RandomSplit.
	index := 0

	ps, err := le.splitter.GetPredictSet(index)
	if err != nil {
		logger.Warningf("live evaluator[%s] failed to get prediction set when evaluate model and error is[%s].", le.id, err.Error())
		return errorx.New(errcodes.ErrCodeGetTrainSet, "live evaluator[%s] failed to get prediction set when evaluate model: %s", le.id, err.Error())
	}

	// encapsulate the request for the prediction task and send the request,
	// no response is processed.

	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.WriteAll(ps)
	file := b.Bytes()

	request := le.packParamsForPredict(index, model, file)
	go func() {
		errSt := le.mpc.StartTask(request)
		if errSt != nil {
			logger.Warningf("evaluator[%s] failed to send StartTaskRequest to start prediction task[%s], and error is[%s].", le.id, request.TaskID, errSt.Error())
		}
	}()

	return nil
}

func (le *liveEvaluator) packParamsForPredict(index int, model *pbCom.TrainModels, file []byte) *pbCom.StartTaskRequest {
	// set idName
	model.IdName = le.taskParams.TrainParams.IdName

	taskParams := pbCom.TaskParams{
		Algo:        le.taskParams.Algo,
		TaskType:    pbCom.TaskType_PREDICT,
		ModelParams: model,
	}

	// if the prediction is from Evaluator, the TaskID conforms such form like `{uuid}_{k}_predict_LEv`,
	res := pbCom.StartTaskRequest{
		TaskID: fmt.Sprintf("%s_%d_predict_LEv", le.id, index),
		File:   file,
		Hosts:  le.hosts,
		Params: &taskParams,
	}

	return &res
}

// SavePredictOut collects the prediction results in the evaluation phase.
// If the prediction result is obtained, it will start calculating metric scores,
// then report the results to visualization system.
func (le *liveEvaluator) SavePredictOut(res *pbCom.PredictTaskResult) error {
	le.mutex.Lock()
	defer le.mutex.Unlock()
	if le.predicRes != nil {
		logger.Warningf("live evaluator[%s] received prediction result more than once for pause round[%d]", le.id, le.pauseRound)
		return errorx.New(errcodes.ErrCodeParam, "live evaluator[%s] received prediction result more than once for pause round[%d]", le.id, le.pauseRound)

	}
	le.predicRes = res

	if !res.Success {
		logger.Warningf("evaluator[%s] got result of prediction task[%s], but it failed and error is[%s].", le.id, res.TaskID, res.ErrMsg)
		go le.callbackLearner()
		return nil
	}

	go le.calMetricScoresAndCallback(res)
	return nil
}

func (le *liveEvaluator) calMetricScoresAndCallbackCaseRegression(res *pbCom.PredictTaskResult) {
	defer func() {
		// callback learner to go on training
		go le.callbackLearner()
	}()

	// The party who has target tag could get prediction result.
	// And the party who hasn't target tag couldn't get prediction result actually,
	// but during evaluation it cooperates with other parties for model training and prediction.

	if le.taskParams.TrainParams.IsTagPart {

		// get first set as validation set for RandomSplit
		index := 0

		pred, err := convert.PredictResultFromBytes(res.Outcomes)
		if err != nil {
			logger.Warningf("live evaluator[%s] failed to convert bytes to PredictResult instance and error is[%s].", le.id, res.ErrMsg)
			return
		}
		lout := len(pred)
		if lout <= 1 {
			logger.Warningf("live evaluator[%s] got invalid prediction result beacause it was too small.", le.id)
			return
		}
		predMap := make(map[string]float64, lout-1)
		for i := 1; i < lout; i++ {
			idValue := pred[i][0]
			labelVal, err := strconv.ParseFloat(pred[i][1], 64)
			if err != nil {
				logger.Warningf("live evaluator[%s] got invalid prediction result beacause label was not type Float64.", le.id)
				return
			}
			predMap[idValue] = labelVal
		}

		// set prediction outcomes to validator for further metric scores
		// keep the same order with samples in Validation/Prediction Set

		// to get Validation Set is more quicky than Prediction Set
		validSet, err := le.splitter.GetValidSet(index)
		if err != nil {
			logger.Warningf("live evaluator[%s] failed to get validation set and error is[%s].", le.id, res.ErrMsg)
			return
		}
		lvs := len(validSet)
		if lvs <= 1 {
			logger.Warningf("live evaluator[%s] got invalid validation set beacuse it was too small.", le.id)
			return
		}
		idIdx := fundIDIndex(validSet, le.taskParams.TrainParams.IdName)
		if idIdx < 0 {
			logger.Warningf("live evaluator[%s] got invalid validation set beacuse it had no ID.", le.id)
			return
		}

		yPreds := make([]float64, 0, lvs-1)
		for i := 1; i < lvs; i++ {
			idValue := validSet[i][idIdx]
			yPreds = append(yPreds, predMap[idValue])
		}

		err = le.validatorCaseRegression.SetPredictOut(index, yPreds)
		if err != nil {
			logger.Warningf("live evaluator[%s] failed to set prediction outcomes to validator and error is[%s].", le.id, err.Error())
			return
		}

		// calculate metric scores and report the results to visualization system
		logger.Infof("live evaluator[%s] got enough prediction outcomes, and start to calculate metric scores.", le.id)
		rmse, err := le.validatorCaseRegression.GetRMSE(index)
		if err != nil {
			logger.Warningf("live evaluator[%s] failed to calculate metric scores and error is[%s].", le.id, err.Error())
			return
		}
		logger.Infof("live evaluator[%s] finish prediction at loopRound[%d], and RMSE is[%f], PredictOut is[%v], ValidationSet is[%v].", le.id, le.pauseRound, rmse, yPreds, validSet)
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

func (le *liveEvaluator) calMetricScoresAndCallbackCaseBinClass(res *pbCom.PredictTaskResult) {
	defer func() {
		// callback learner to go on training
		go le.callbackLearner()
	}()

	// The party who has target tag could get prediction result.
	// And the party who hasn't target tag couldn't get prediction result actually,
	// but during evaluation it cooperates with other parties for model training and prediction.

	if le.taskParams.TrainParams.IsTagPart {

		// get first set as validation set for RandomSplit
		index := 0

		pred, err := convert.PredictResultFromBytes(res.Outcomes)
		if err != nil {
			logger.Warningf("live evaluator[%s] failed to convert bytes to PredictResult instance and error is[%s].", le.id, res.ErrMsg)
			return
		}
		lout := len(pred)
		if lout <= 1 {
			logger.Warningf("live evaluator[%s] got invalid prediction result beacause it was too small.", le.id)
			return
		}
		predMap := make(map[string]float64, lout-1)
		for i := 1; i < lout; i++ {
			idValue := pred[i][0]
			labelVal, err := strconv.ParseFloat(pred[i][1], 64)
			if err != nil {
				logger.Warningf("live evaluator[%s] got invalid prediction result beacause label was not type Float64.", le.id)
				return
			}
			predMap[idValue] = labelVal
		}

		// set prediction outcomes to validator for further metric scores
		// keep the same order with samples in Validation/Prediction Set

		// to get Validation Set is more quicky than Prediction Set
		validSet, err := le.splitter.GetValidSet(index)
		if err != nil {
			logger.Warningf("live evaluator[%s] failed to get validation set and error is[%s].", le.id, res.ErrMsg)
			return
		}
		lvs := len(validSet)
		if lvs <= 1 {
			logger.Warningf("live evaluator[%s] got invalid validation set beacuse it was too small.", le.id)
			return
		}
		idIdx := fundIDIndex(validSet, le.taskParams.TrainParams.IdName)
		if idIdx < 0 {
			logger.Warningf("live evaluator[%s] got invalid validation set beacuse it had no ID.", le.id)
			return
		}

		predProba := make([]float64, 0, lvs-1)
		for i := 1; i < lvs; i++ {
			idValue := validSet[i][idIdx]
			predProba = append(predProba, predMap[idValue])
		}

		err = le.validatorCaseBinClass.SetPredictOut(index, predProba)
		if err != nil {
			logger.Warningf("live evaluator[%s] failed to set prediction outcomes to validator and error is[%s].", le.id, err.Error())
			return
		}

		// calculate metric scores and report the results to visualization system

		logger.Infof("live evaluator[%s] got enough prediction outcomes, and start to calculate metric scores.", le.id)
		report, err := le.validatorCaseBinClass.GetReport(index)
		if err != nil {
			logger.Warningf("live evaluator[%s] failed to calculate metric scores and error is[%s].", le.id, err.Error())
			return
		}

		var accuracy float64
		var precision float64
		var recall float64
		var f1score float64

		var summary confusionMatrixSummary
		json.Unmarshal(report, &summary)
		// Determine whether there are positive samples
		if metr, ok := summary.Metrics[le.taskParams.TrainParams.LabelName]; ok {
			precision = metr.Precision
			recall = metr.Recall
			f1score = metr.F1Score
		}
		accuracy = summary.Accuracy
		logger.Infof("live evaluator[%s] finish prediction at loopRound[%d], and Accuracy is[%f], Precision is[%f], Recall is[%f], F1Score is[%f], and PredictOut is[%v], ValidationSet is[%v].",
			le.id, le.pauseRound, accuracy, precision, recall, f1score, predProba, validSet)

		// callback learner to go on training
		go le.callbackLearner()
	} else {
		// callback learner to go on training
		go le.callbackLearner()
	}

}

// callbackLearner calls back learner to go on training
func (le *liveEvaluator) callbackLearner() {
	resp, err := le.mpc.Train(&pb.TrainRequest{
		TaskID:  le.learnerID,
		Algo:    le.algo,
		Payload: le.callbackPayload,
	})
	if err != nil {
		logger.Warnf("live evaluator[%s] failed to call back learner to continue training at loopRound[%d], and error is[%s].",
			le.id, le.pauseRound, err.Error())
		return
	}
	logger.Infof("live evaluator[%s] called back learner to continue training at loopRound[%d] successfully and response is[%v].",
		le.id, le.pauseRound, resp)
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

func NewLiveEvaluator(req *pbCom.StartTaskRequest, mpc Mpc) (LiveEvaluator, error) {
	le := &liveEvaluator{}

	algo := req.Params.Algo
	switch algo {
	case pbCom.Algorithm_LINEAR_REGRESSION_VL:
		le.caseType = pbCom.CaseType_Regression
		le.calMetricScoresAndCallback = le.calMetricScoresAndCallbackCaseRegression
		le.runLearner = le.runLinearRegVL
	case pbCom.Algorithm_LOGIC_REGRESSION_VL:
		le.caseType = pbCom.CaseType_BinaryClass
		le.calMetricScoresAndCallback = le.calMetricScoresAndCallbackCaseBinClass
		le.runLearner = le.runLogicRegVL
	default:
		return nil, errorx.New(errcodes.ErrCodeParam, "unknown algorithm: %s", algo.String())
	}

	// check evaluation params and Evaluation Rule
	if req.Params.LivalParams == nil || !req.Params.LivalParams.Enable {
		return nil, errorx.New(errcodes.ErrCodeParam, "invalid live evaluation params")
	}
	if req.Params.LivalParams.RandomSplit == nil {
		return nil, errorx.New(errcodes.ErrCodeParam, "no RandomSplit set")
	}

	le.id = req.TaskID
	le.algo = algo
	le.mpc = mpc
	le.hosts = req.Hosts
	le.taskParams = req.Params
	le.livalParams = req.Params.LivalParams
	le.evalRule = pbCom.EvaluationRule_ErRandomSplit
	le.learnerID = req.TaskID

	//if the request to create Learner from LiveEvaluator, the TaskID conforms such form like `{uuid}_{k}_train_LEv`,
	le.evalLearnerID = fmt.Sprintf("%s_%d_train_LEv", req.TaskID, 0)

	return le, nil
}
