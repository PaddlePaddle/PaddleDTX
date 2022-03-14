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
	"time"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

// updateExpTimeCmd represents the command to update file expiration time
var updateExpTimeCmd = &cobra.Command{
	Use:   "utime",
	Short: "update file expiretime by id",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		stamp, err := time.ParseInLocation(timeTemplate, expireTime, time.Local)
		if err != nil {
			fmt.Printf("err：%v\n", err)
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

		err = client.UpdateExpTimeByID(context.Background(), id, privateKey, stamp.UnixNano())
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		fmt.Println("OK")
	},
}

func init() {
	rootCmd.AddCommand(updateExpTimeCmd)

	updateExpTimeCmd.Flags().StringVarP(&id, "id", "i", "", "id for file")
	updateExpTimeCmd.Flags().StringVarP(&expireTime, "expireTime", "e", "", "expire time, example '2021-07-10 12:00:00'")
	updateExpTimeCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "private key")
	updateExpTimeCmd.Flags().StringVarP(&keyPath, "keyPath", "", "./ukeys", "key path")

	updateExpTimeCmd.MarkFlagRequired("id")
	updateExpTimeCmd.MarkFlagRequired("expireTime")
}
