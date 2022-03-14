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

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

// nodeOnlineCmd represents the command to get node of xuper db online
var nodeOnlineCmd = &cobra.Command{
	Use:   "online",
	Short: "set a storage node online",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		if privateKey == "" {
			privateKeyBytes, err := file.ReadFile(keyPath, file.PrivateKeyFileName)
			if err != nil {
				fmt.Printf("Read privateKey failed, err: %v\n", err)
				return
			}
			privateKey = strings.TrimSpace(string(privateKeyBytes))
		}
		privKey, err := ecdsa.DecodePrivateKeyFromString(privateKey)
		if err != nil {
			fmt.Printf("failed to DecodePrivateKeyFromString, err: %v\n", err)
			return
		}
		if err := client.NodeOnline(context.Background(), privKey.String()); err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		fmt.Println("node online")
	},
}

func init() {
	rootCmd.AddCommand(nodeOnlineCmd)

	nodeOnlineCmd.Flags().StringVarP(&privateKey, "privateKey", "k", "", "privatekey")
	nodeOnlineCmd.Flags().StringVarP(&keyPath, "keyPath", "", file.KeyFilePath, "node's key path")
}
