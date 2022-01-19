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
	"time"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
)

// getNsCmd represents the command to get file namespace details
var getNsCmd = &cobra.Command{
	Use:   "getns",
	Short: "get the file namespace detail in XuperDB",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		nsh, err := client.GetNsByName(context.Background(), owner, namespace)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		ns := nsh.Namespace
		ctime := time.Unix(0, ns.CreateTime).Format(timeTemplate)
		utime := time.Unix(0, ns.UpdateTime).Format(timeTemplate)

		fmt.Printf("Name: %s\nFileTotalNum: %d\nFileNormalNum: %d\nFileExpiredNum: %d\nReplica: %d\nGreenFileNum: %d\nYellowFileNum: %d\nRedFileNum: %d\n",
			ns.Name, ns.FileTotalNum, nsh.FileNormalNum, nsh.FileExpiredNum, ns.Replica, nsh.GreenFileNum, nsh.YellowFileNum, nsh.RedFileNum)
		fmt.Printf("Description: %s\nUpdateTime: %s\nCreateTime: %s\n\n", ns.Description, utime, ctime)
	},
}

func init() {
	rootCmd.AddCommand(getNsCmd)

	getNsCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace for file")
	getNsCmd.Flags().StringVarP(&owner, "owner", "o", "", "owner for namespace")

	getNsCmd.MarkFlagRequired("namespace")
}
