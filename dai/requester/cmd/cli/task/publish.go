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

package task

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	requestClient "github.com/PaddlePaddle/PaddleDTX/dai/requester/client"
	"github.com/PaddlePaddle/PaddleDTX/dai/util/file"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var (
	files       string
	executors   string
	algorithm   string
	taskType    string
	taskName    string
	label       string
	labelName   string
	regMode     string
	regParam    float64
	alpha       float64
	amplitude   float64
	accuracy    uint64
	taskId      string
	description string
	psiLabel    string
	batchSize   uint64
	ev          bool  // whether perform model evaluation
	evRule      int32 // evRule is the way to evaluate model, 0 means `Random Split`, 1 means `Cross Validation`, 2 means `Leave One Out`
	percentLO   int32 // percentage to leave out as validation set when perform model evaluation in the way of `Random Split`
	folds       int32 // number of folds, 5 or 10 supported, default `10`, a optional parameter when perform model evaluation in the way of `Cross Validation`
	shuffle     bool  // whether to randomly disorder the samples before dividion, default `false`, a optional parameter when perform model evaluation in the way of `Cross Validation`

	le         bool  // whether perform live model evaluation
	lPercentLO int32 // percentage to leave out as validation set when perform live model evaluation
)

// checkTaskPublishParams check mpc task parameters
// verify if algorithm, taskType, regMode is legal
func checkTaskPublishParams() (pbCom.Algorithm, pbCom.TaskType, pbCom.RegMode, error) {
	var pAlgo pbCom.Algorithm
	var pType pbCom.TaskType
	var pRegMode pbCom.RegMode
	// task algorithm name check
	if algo, ok := blockchain.VlAlgorithmListName[algorithm]; ok {
		pAlgo = algo
	} else {
		return pAlgo, pType, pRegMode, errorx.New(errorx.ErrCodeParam, "algorithm only support linear-vl or logistic-vl")
	}
	// task type check
	if taskType, ok := blockchain.TaskTypeListName[taskType]; ok {
		pType = taskType
	} else {
		return pAlgo, pType, pRegMode, errorx.New(errorx.ErrCodeParam, "invalid task type: %s", taskType)
	}
	// task regMode check, no regularization if not set
	if mode, ok := blockchain.RegModeListName[regMode]; ok {
		pRegMode = mode
	} else {
		pRegMode = pbCom.RegMode_Reg_None
	}
	return pAlgo, pType, pRegMode, nil
}

// publishCmd publishes FL task
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "publish a task, can be a training task or a prediction task",
	Run: func(cmd *cobra.Command, args []string) {

		client, err := requestClient.GetRequestClient(configPath)
		if err != nil {
			fmt.Printf("GetRequestClient failed: %v\n", err)
			return
		}
		algo, taskType, regMode, err := checkTaskPublishParams()
		if err != nil {
			fmt.Printf("failed to check task publish algoParam : %v\n", err)
			return
		}
		// check params about evaluation
		if evRule < 0 || evRule > 2 {
			fmt.Printf("invalid `evRule`, it should be 0 or 1 or 2")
			return
		}
		if percentLO <= 0 || percentLO >= 100 {
			fmt.Printf("invalid `plo`, it should in the range of (0,100)")
			return
		}
		if folds != 5 && folds != 10 {
			fmt.Printf("invalid `folds`, it should be 5 or 10")
			return
		}
		if lPercentLO <= 0 || lPercentLO >= 100 {
			fmt.Printf("invalid `lplo`, it should in the range of (0,100)")
			return
		}

		// pack `pbCom.TaskParams`
		algorithmParams := pbCom.TaskParams{
			Algo:        algo,
			TaskType:    taskType,
			ModelTaskID: taskId,
			TrainParams: &pbCom.TrainParams{
				Label:     label,
				LabelName: labelName,
				RegMode:   regMode,
				RegParam:  regParam,
				Alpha:     alpha,
				Amplitude: amplitude,
				Accuracy:  int64(accuracy),
				BatchSize: int64(batchSize),
			},
		}
		// set `Evaluation` part
		if ev {
			algorithmParams.EvalParams = &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule(evRule),
			}
			if algorithmParams.EvalParams.EvalRule == pbCom.EvaluationRule_ErRandomSplit {
				algorithmParams.EvalParams.RandomSplit = &pbCom.RandomSplit{PercentLO: percentLO}
			} else if algorithmParams.EvalParams.EvalRule == pbCom.EvaluationRule_ErCrossVal {
				algorithmParams.EvalParams.Cv = &pbCom.CrossVal{
					Folds:   folds,
					Shuffle: shuffle,
				}
			}
		}
		// set `LiveEvaluation` part
		if le {
			algorithmParams.LivalParams = &pbCom.LiveEvaluationParams{
				Enable: true,
				RandomSplit: &pbCom.RandomSplit{
					PercentLO: lPercentLO,
				},
			}
		}

		if privateKey == "" {
			privateKeyBytes, err := file.ReadFile(keyPath, file.PrivateKeyFileName)
			if err != nil {
				fmt.Printf("Read privateKey failed, err: %v\n", err)
				return
			}
			privateKey = strings.TrimSpace(string(privateKeyBytes))
		}

		taskID, err := client.Publish(requestClient.PublishOptions{
			PrivateKey:  privateKey,
			Files:       files,
			Executors:   executors,
			TaskName:    taskName,
			AlgoParam:   algorithmParams,
			Description: description,
			PSILabels:   psiLabel,
		})
		if err != nil {
			fmt.Printf("Publish task failed: %v\n", err)
			return
		}
		fmt.Println("TaskID:", taskID)
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVarP(&taskName, "name", "n", "", "task's name")
	publishCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "requester's private key hex string")
	publishCmd.Flags().StringVarP(&keyPath, "keyPath", "", "./keys", "requester's key path")
	publishCmd.Flags().StringVarP(&taskType, "type", "t", "", "task type, 'train' or 'predict'")
	publishCmd.Flags().StringVarP(&algorithm, "algorithm", "a", "", "algorithm assigned to task, 'linear-vl' and 'logistic-vl' are supported")
	publishCmd.Flags().StringVarP(&files, "files", "f", "", "sample files IDs with ',' as delimiter, like '123,456'")
	publishCmd.Flags().StringVarP(&executors, "executors", "e", "", "executor node names with ',' as delimiter, like 'executor1,executor2'")

	// optional params
	publishCmd.Flags().StringVarP(&label, "label", "l", "", "target feature for training task")
	publishCmd.Flags().StringVar(&labelName, "labelName", "", "target variable required in logistic-vl training")
	publishCmd.Flags().StringVarP(&psiLabel, "PSILabel", "p", "", "ID feature name list with ',' as delimiter, like 'id,id', required in vertical task")
	publishCmd.Flags().StringVarP(&taskId, "taskId", "i", "", "finished train task ID from which obtain the model, required for predict task")
	publishCmd.Flags().StringVar(&regMode, "regMode", "", "regularization mode required in train task, no regularization if not set, options are l1(L1-norm) and l2(L2-norm)")
	publishCmd.Flags().Float64Var(&regParam, "regParam", 0.1, "regularization parameter required in train task if set regMode")
	publishCmd.Flags().Float64Var(&alpha, "alpha", 0.1, "learning rate required in train task")
	publishCmd.Flags().Float64Var(&amplitude, "amplitude", 0.0001, "target difference of costs in two contiguous rounds that determines whether to stop training")
	publishCmd.Flags().Uint64Var(&accuracy, "accuracy", 10, "accuracy of homomorphic encryption")
	publishCmd.Flags().StringVarP(&description, "description", "d", "", "task description")
	publishCmd.Flags().Uint64VarP(&batchSize, "batchSize", "b", 4,
		"size of samples for one round of training loop, 0 for BGD(Batch Gradient Descent), non-zero for SGD(Stochastic Gradient Descent) or MBGD(Mini-Batch Gradient Descent)")
	// optional params about evaluation
	publishCmd.Flags().BoolVar(&ev, "ev", false, "perform model evaluation")
	publishCmd.Flags().Int32Var(&evRule, "evRule", 0, "the way to evaluate model, 0 means 'Random Split', 1 means 'Cross Validation', 2 means 'Leave One Out'")
	publishCmd.Flags().Int32Var(&folds, "folds", 10, "number of folds, 5 or 10 supported, a optional parameter when perform model evaluation in the way of 'Cross Validation'")
	publishCmd.Flags().BoolVar(&shuffle, "shuffle", false, "shuffle the samples before division when perform model evaluation in the way of 'Cross Validation'")
	publishCmd.Flags().Int32Var(&percentLO, "plo", 30, "percentage to leave out as validation set when perform model evaluation in the way of 'Random Split'")

	// optional params about live evaluation
	publishCmd.Flags().BoolVar(&le, "le", false, "perform live model evaluation")
	publishCmd.Flags().Int32Var(&lPercentLO, "lplo", 30, "percentage to leave out as validation set when perform live model evaluation")

	publishCmd.MarkFlagRequired("name")
	publishCmd.MarkFlagRequired("type")
	publishCmd.MarkFlagRequired("algorithm")
	publishCmd.MarkFlagRequired("files")
	publishCmd.MarkFlagRequired("executors")
}
