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

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
)

// getFileSysHealthCmd represents the command to get system health of xuper db
var getFileSysHealthCmd = &cobra.Command{
	Use:   "syshealth",
	Short: "get the DataOwner's health status",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		fh, err := client.GetFileSysHealth(context.Background(), owner)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		fmt.Printf("\nNsNum:%d\nFileNum: %d\nFileExpiredNum:%d\nGreenFileNum: %d\nYellowFileNum: %d\nRedFileNum: %d\nFilesHealthRate: %.2f\nSysHealth: %s\n\n",
			fh.NsNum, fh.FileNum, fh.FileExpiredNum, fh.GreenFileNum, fh.YellowFileNum, fh.RedFileNum, fh.FilesHealthRate, fh.SysHealth)
		fmt.Printf("NodeNum: %d\nGreenNodeNum: %d\nYellowNodeNum: %d\nRedNodeNum: %d\nNodeHealthRate: %.2f\n\n",
			fh.NodeNum, fh.GreenNodeNum, fh.YellowNodeNum, fh.RedNodeNum, fh.NodeHealthRate)
	},
}

func init() {
	rootCmd.AddCommand(getFileSysHealthCmd)

	getFileSysHealthCmd.Flags().StringVarP(&owner, "owner", "o", "", "owner for files")
}
