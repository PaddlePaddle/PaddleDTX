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

package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

// getNodeHealthCmd represents the command to get node health status
var getNodeHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "get the storage node's health status by id",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		if id == "" {
			pubKeyBytes, err := file.ReadFile(keyPath, file.PublicKeyFileName)
			if err != nil {
				fmt.Printf("Read publicKey failed, err: %v\n", err)
				return
			}
			id = strings.TrimSpace(string(pubKeyBytes))
		}
		status, err := client.GetNodeHealth(context.Background(), id)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		fmt.Printf("NodeID: %s\nHealthStatus: %s\n", id, status)
	},
}

func init() {
	rootCmd.AddCommand(getNodeHealthCmd)

	getNodeHealthCmd.Flags().StringVarP(&id, "id", "i", "", "id")
	getNodeHealthCmd.Flags().StringVarP(&keyPath, "keyPath", "", file.KeyFilePath, "node's key path")
}
