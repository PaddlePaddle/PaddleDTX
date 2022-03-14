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

package files

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

// updateNsReplicaCmd represents the command to update replica
var updateNsReplicaCmd = &cobra.Command{
	Use:   "ureplica",
	Short: "update file replica of XuperDB",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		if replica <= 0 || len(namespace) == 0 {
			fmt.Printf("err: bad param, replica and ns length must greater than 0")
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

		err = client.UpdateFileNsReplica(context.Background(), privateKey, namespace, replica)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		fmt.Println("OK")
	},
}

func init() {
	rootCmd.AddCommand(updateNsReplicaCmd)

	updateNsReplicaCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "private key")
	updateNsReplicaCmd.Flags().StringVarP(&keyPath, "keyPath", "", "./ukeys", "key path")
	updateNsReplicaCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace for file")
	updateNsReplicaCmd.Flags().IntVarP(&replica, "replica", "r", 0, "replica")

	updateNsReplicaCmd.MarkFlagRequired("namespace")
	updateNsReplicaCmd.MarkFlagRequired("replica")
}
