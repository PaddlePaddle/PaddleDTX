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
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	executorClient "github.com/PaddlePaddle/PaddleDTX/dai/executor/client"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

// getByIDCmd get task by taskID
var getByIDCmd = &cobra.Command{
	Use:   "getbyid",
	Short: "get the task by id",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := executorClient.GetExecutorClient(host)
		if err != nil {
			fmt.Printf("GetExecutorClient failed: %v\n", err)
			return
		}
		t, err := client.GetTaskById(context.Background(), id)
		if err != nil {
			fmt.Printf("GetTaskById failed：%v\n", err)
			return
		}
		ptime := time.Unix(0, t.PublishTime).Format(timeTemplate)

		fmt.Printf("TaskID: %s\nRequester: %x\nTaskType: %s\nTaskName: %s\nDescription: %s\nLabel: %s\nLabelName: %s\nRegMode: %v\nRegParam: %v\n",
			t.ID, t.Requester, blockchain.TaskTypeListValue[t.AlgoParam.TaskType], t.Name, t.Description, t.AlgoParam.TrainParams.Label,
			t.AlgoParam.TrainParams.LabelName, blockchain.RegModeListValue[t.AlgoParam.TrainParams.RegMode], t.AlgoParam.TrainParams.RegParam)

		fmt.Printf("Algorithm: %v\nAlpha: %f\nAmplitude: %f\nAccuracy: %v\nModelTaskID: %s\nStatus: %s\nPublishTime: %s\n\n",
			blockchain.VlAlgorithmListValue[t.AlgoParam.Algo], t.AlgoParam.TrainParams.Alpha, t.AlgoParam.TrainParams.Amplitude,
			t.AlgoParam.TrainParams.Accuracy, t.AlgoParam.ModelTaskID, t.Status, ptime)

		if t.AlgoParam.EvalParams != nil && t.AlgoParam.EvalParams.Enable {
			fmt.Printf("ModelEvaluationRule: %s\n",
				t.AlgoParam.EvalParams.EvalRule)
			if t.AlgoParam.EvalParams.EvalRule == pbCom.EvaluationRule_ErRandomSplit {
				fmt.Printf("PercentageToLeaveOutAsValidation: %d\n\n",
					t.AlgoParam.EvalParams.RandomSplit.PercentLO)
			} else if t.AlgoParam.EvalParams.EvalRule == pbCom.EvaluationRule_ErCrossVal {
				fmt.Printf("Shuffled: %t\nFolds: %d\n\n",
					t.AlgoParam.EvalParams.Cv.Shuffle, t.AlgoParam.EvalParams.Cv.Folds)
			} else if t.AlgoParam.EvalParams.EvalRule == pbCom.EvaluationRule_ErLOO {
				fmt.Print("\n")
			}
		}

		fmt.Println("Task data sets: ")

		for _, d := range t.DataSets {
			var ct, rt string
			if d.ConfirmedAt > 0 {
				ct = time.Unix(0, d.ConfirmedAt).Format(timeTemplate)
			}
			if d.RejectedAt > 0 {
				rt = time.Unix(0, d.RejectedAt).Format(timeTemplate)
			}
			fmt.Printf("DataID: %s\nOwner: %x\nExecutor: %x\nAddress: %s\nPSILabel: %s\nConfirmedAt: %s\nRejectedAt: %s\n\n",
				d.DataID, d.Owner, d.Executor, d.Address, d.PSILabel, ct, rt)
		}

		startTime := time.Unix(0, t.StartTime).Format(timeTemplate)
		endTime := time.Unix(0, t.EndTime).Format(timeTemplate)
		fmt.Printf("\nStartTime: %s\nEndTime: %s\n\n", startTime, endTime)

		fmt.Printf("\nErrMessage: %s\nResult: %s\n\n", t.ErrMessage, t.Result)
	},
}

func init() {
	rootCmd.AddCommand(getByIDCmd)

	getByIDCmd.Flags().StringVarP(&id, "id", "i", "", "task id")

	getByIDCmd.MarkFlagRequired("id")
}
