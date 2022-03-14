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
	"time"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

// getNodeCmd gets storage node by id
var getNodeCmd = &cobra.Command{
	Use:   "get",
	Short: "get the storage node by id",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err: %v\n", err)
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

		n, err := client.GetNode(context.Background(), id)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		rtime := time.Unix(0, n.RegTime).Format(timeTemplate)
		utime := time.Unix(0, n.UpdateAt).Format(timeTemplate)
		fmt.Printf("NodeID: %s\nName: %s\nAddress: %s\nOnline: %v\nRegisterTime: %v\nUpdateTime: %v\n", n.ID, n.Name, n.Address, n.Online, rtime, utime)
	},
}

func init() {
	rootCmd.AddCommand(getNodeCmd)
	getNodeCmd.Flags().StringVarP(&id, "id", "i", "", "id")
	getNodeCmd.Flags().StringVarP(&keyPath, "keyPath", "", file.KeyFilePath, "node's key path")
}
