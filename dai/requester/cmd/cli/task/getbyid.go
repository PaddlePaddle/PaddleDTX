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
	"time"

	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	requestClient "github.com/PaddlePaddle/PaddleDTX/dai/requester/client"
)

// getByIDCmd
var getByIDCmd = &cobra.Command{
	Use:   "getbyid",
	Short: "get the task by id",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requestClient.GetRequestClient(configPath)
		if err != nil {
			fmt.Printf("GetRequestClient failed: %v\n", err)
			return
		}
		task, err := client.GetTaskById(id)
		if err != nil {
			fmt.Printf("GetTaskById failed：%v\n", err)
			return
		}

		publishTime := time.Unix(0, task.PublishTime).Format(timeTemplate)

		fmt.Printf("TaskID: %s\nRequester: %x\nTaskType: %s\nTaskName: %s\nDescription: %s\nLabel: %s\nLabelName: %s\nRegMode: %v\nRegParam: %v\n",
			task.ID, task.Requester, blockchain.TaskTypeListValue[task.AlgoParam.TaskType], task.Name, task.Description, task.AlgoParam.TrainParams.Label,
			task.AlgoParam.TrainParams.LabelName, blockchain.RegModeListValue[task.AlgoParam.TrainParams.RegMode], task.AlgoParam.TrainParams.RegParam)

		fmt.Printf("Algorithm: %v\nAlpha: %f\nAmplitude: %f\nAccuracy: %v\nModelTaskID: %s\nStatus: %s\nPublishTime: %s\n\n",
			blockchain.VlAlgorithmListValue[task.AlgoParam.Algo], task.AlgoParam.TrainParams.Alpha, task.AlgoParam.TrainParams.Amplitude,
			task.AlgoParam.TrainParams.Accuracy, task.AlgoParam.ModelTaskID, task.Status, publishTime)

		if task.AlgoParam.EvalParams != nil && task.AlgoParam.EvalParams.Enable {
			fmt.Printf("ModelEvaluationRule: %s\n",
				task.AlgoParam.EvalParams.EvalRule)
			if task.AlgoParam.EvalParams.EvalRule == pbCom.EvaluationRule_ErRandomSplit {
				fmt.Printf("PercentageToLeaveOutAsValidation: %d\n\n",
					task.AlgoParam.EvalParams.RandomSplit.PercentLO)
			} else if task.AlgoParam.EvalParams.EvalRule == pbCom.EvaluationRule_ErCrossVal {
				fmt.Printf("Shuffled: %t\nFolds: %d\n\n",
					task.AlgoParam.EvalParams.Cv.Shuffle, task.AlgoParam.EvalParams.Cv.Folds)
			} else if task.AlgoParam.EvalParams.EvalRule == pbCom.EvaluationRule_ErLOO {
				fmt.Print("\n")
			}
		}

		fmt.Println("Task data sets: ")
		for _, data := range task.DataSets {
			var ct, rt string
			if data.ConfirmedAt > 0 {
				ct = time.Unix(0, data.ConfirmedAt).Format(timeTemplate)
			}
			if data.RejectedAt > 0 {
				rt = time.Unix(0, data.RejectedAt).Format(timeTemplate)
			}
			fmt.Printf("DataID: %s\nOwner: %x\nExecutor: %x\nAddress: %s\nPSILabel: %s\nConfirmedAt: %s\nRejectedAt: %s\n\n",
				data.DataID, data.Owner, data.Executor, data.Address, data.PSILabel, ct, rt)
		}

		var startTime, endTime string
		if task.StartTime != 0 {
			startTime = time.Unix(0, task.StartTime).Format(timeTemplate)
		}
		if task.EndTime != 0 {
			endTime = time.Unix(0, task.EndTime).Format(timeTemplate)
		}
		fmt.Printf("StartTime: %s\nEndTime: %s\n", startTime, endTime)

		fmt.Printf("ErrMessage: %s\nResult: %s\n\n", task.ErrMessage, task.Result)
	},
}

func init() {
	rootCmd.AddCommand(getByIDCmd)

	getByIDCmd.Flags().StringVarP(&id, "id", "i", "", "task id")

	getByIDCmd.MarkFlagRequired("id")
}
