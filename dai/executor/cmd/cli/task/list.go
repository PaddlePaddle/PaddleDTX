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
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	executorClient "github.com/PaddlePaddle/PaddleDTX/dai/executor/client"
	"github.com/PaddlePaddle/PaddleDTX/dai/util/file"
)

var (
	status string
	pubkey string
)

// listTasksCmd lists tasks from blockchain with specific participant public key and task status
var listTasksCmd = &cobra.Command{
	Use:   "list",
	Short: "list tasks from blockchain through executor node",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := executorClient.GetExecutorClient(host)
		if err != nil {
			fmt.Printf("GetExecutorClient failed: %v\n", err)
			return
		}
		var startTime int64 = 0
		if start != "" {
			s, err := time.ParseInLocation(timeTemplate, start, time.Local)
			if err != nil {
				fmt.Printf("ParseInLocation failed：%v\n", err)
				return
			}
			startTime = s.UnixNano()
		}
		endTime, err := time.ParseInLocation(timeTemplate, end, time.Local)
		if err != nil {
			fmt.Printf("ParseInLocation failed：%v\n", err)
			return
		}
		if limit > blockchain.TaskListMaxNum {
			fmt.Printf("invalid limit, the value must smaller than %v \n", blockchain.TaskListMaxNum)
			return
		}
		if pubkey == "" {
			pubkeyBytes, err := file.ReadFile(keyPath, file.PublicKeyFileName)
			if err != nil {
				fmt.Printf("Read publicKey failed, err: %v\n", err)
				return
			}
			pubkey = strings.TrimSpace(string(pubkeyBytes))
		}

		tasks, err := client.ListTask(context.Background(), pubkey, status, startTime, endTime.UnixNano(), limit)
		if err != nil {
			fmt.Printf("ListTask failed：%v\n", err)
			return
		}

		for _, task := range tasks.FLTasks {
			ptime := time.Unix(0, task.PublishTime).Format(timeTemplate)
			fmt.Printf("TaskID: %s\nTaskType: %s\nTaskName: %s\nDescription: %s\nTaskStatus: %s\nPublishTime: %s\n\n",
				task.ID, task.AlgoParam.TaskType, task.Name, task.Description, task.Status, ptime)
		}

		fmt.Printf("taskNum : %d\n\n", len(tasks.FLTasks))
	},
}

func init() {
	rootCmd.AddCommand(listTasksCmd)

	listTasksCmd.Flags().StringVarP(&pubkey, "pubkey", "p", "", "requester or executor public key hex string, support listing tasks a requester published or tasks an executor involved")
	listTasksCmd.Flags().StringVarP(&keyPath, "keyPath", "", "./keys", "executor's key path")
	listTasksCmd.Flags().StringVarP(&start, "start", "s", "", "start of time range during which tasks were published, example '2021-06-10 12:00:00'")
	listTasksCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate), "end of time range during which tasks were published, example '2021-06-10 12:00:00'")
	listTasksCmd.Flags().Int64VarP(&limit, "limit", "l", blockchain.TaskListMaxNum, "limit of number for listing tasks")
	listTasksCmd.Flags().StringVar(&status, "status", "", "status of task, such as Confirming, Ready, ToProcess, Processing, Finished, Failed, default for all types of status")
}
