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

	"github.com/spf13/cobra"

	executorClient "github.com/PaddlePaddle/PaddleDTX/dai/executor/client"
)

// confirmCmd confirms task in Ready status by taskID
var confirmCmd = &cobra.Command{
	Use:   "confirm",
	Short: "confirm task in Ready status",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := executorClient.GetExecutorClient(host)
		if err != nil {
			fmt.Printf("GetExecutorClient failed: %v\n", err)
			return
		}

		if err := client.ConfirmTask(context.Background(), privateKey, id); err != nil {
			fmt.Printf("ConfirmTask failed：%v\n", err)
			return
		}
		fmt.Println("ok")
	},
}

func init() {
	rootCmd.AddCommand(confirmCmd)

	confirmCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "private key")
	confirmCmd.Flags().StringVarP(&id, "id", "i", "", "task id")

	confirmCmd.MarkFlagRequired("id")
	confirmCmd.MarkFlagRequired("privkey")
}
