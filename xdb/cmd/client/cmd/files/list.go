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

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
)

// listFilesCmd represents the command to list files by namespace
var listFilesCmd = &cobra.Command{
	Use:   "list",
	Short: "list files in XuperDB",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		var startTime int64 = 0
		if start != "" {
			s, err := time.ParseInLocation(timeTemplate, start, time.Local)
			if err != nil {
				fmt.Printf("err：%v\n", err)
				return
			}
			startTime = s.UnixNano()
		}
		endTime, err := time.ParseInLocation(timeTemplate, end, time.Local)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		if limit > blockchain.ListMaxNumber {
			fmt.Printf("invalid limit, the value must smaller than %v \n", blockchain.ListMaxNumber)
			return
		}

		opt := httpclient.ListFileOptions{
			Owner:     owner,
			Namespace: namespace,
			TimeStart: startTime,
			TimeEnd:   endTime.UnixNano(),
			Limit:     limit,
		}
		resp, err := client.ListFiles(context.Background(), opt)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		for _, f := range resp {
			ptime := time.Unix(0, f.PublishTime).Format(timeTemplate)
			etime := time.Unix(0, f.ExpireTime).Format(timeTemplate)
			fmt.Printf("FileID: %s\nFileName: %s\nFileDescription: %s\nNamespace: %s\nFileLength: %v\nPublishTimes: %s\nExpireTime: %s\n\n",
				f.ID, f.Name, f.Description, f.Namespace, f.Length, ptime, etime)
		}
		if len(resp) == 0 {
			fmt.Printf("\nno files\n\n")
		} else {
			fmt.Printf("\nfiles num from %s to %s: %d\n\n", start, end, len(resp))
		}
	},
}

// listExpFilesCmd represents the list expired but valid files by namespace command
var listExpFilesCmd = &cobra.Command{
	Use:   "listexp",
	Short: "list expired but valid files in XuperDB",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		var startTime int64 = 0
		if start != "" {
			s, err := time.ParseInLocation(timeTemplate, start, time.Local)
			if err != nil {
				fmt.Printf("err：%v\n", err)
				return
			}
			startTime = s.UnixNano()
		}
		endTime, err := time.ParseInLocation(timeTemplate, end, time.Local)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		if limit > blockchain.ListMaxNumber {
			fmt.Printf("invalid limit, the value must smaller than %v \n", blockchain.ListMaxNumber)
			return
		}

		opt := httpclient.ListFileOptions{
			Owner:     owner,
			Namespace: namespace,
			TimeStart: startTime,
			TimeEnd:   endTime.UnixNano(),
			Limit:     limit,
		}
		resp, err := client.ListExpiredFiles(context.Background(), opt)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		for _, f := range resp {
			ptime := time.Unix(0, f.PublishTime).Format(timeTemplate)
			etime := time.Unix(0, f.ExpireTime).Format(timeTemplate)
			fmt.Printf("FileID: %s\nFileName: %s\nFileDescription: %s\nNamespace: %s\nFileLength: %v\nPublishTimes: %s\nExpireTime: %s\n\n",
				f.ID, f.Name, f.Description, f.Namespace, f.Length, ptime, etime)
		}
		if len(resp) == 0 {
			fmt.Printf("\nno files\n\n")
		} else {
			fmt.Printf("\nfiles num from %s to %s: %d\n\n", start, end, len(resp))
		}
	},
}

func init() {
	rootCmd.AddCommand(listFilesCmd)
	rootCmd.AddCommand(listExpFilesCmd)

	listFilesCmd.Flags().StringVarP(&owner, "owner", "o", "", "owner for file")
	listFilesCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace for file")
	listFilesCmd.Flags().StringVarP(&start, "start", "s", "", "file publish after startTime, example '2021-06-10 12:00:00'")
	listFilesCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate), "file publish before endTime, example '2021-06-10 12:00:00'")
	listFilesCmd.Flags().Int64VarP(&limit, "limit", "l", blockchain.ListMaxNumber, "limit for list files")

	listFilesCmd.MarkFlagRequired("namespace")

	listExpFilesCmd.Flags().StringVarP(&owner, "owner", "o", "", "owner for file")
	listExpFilesCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace for file")
	listExpFilesCmd.Flags().StringVarP(&start, "start", "s", "", "file publish after startTime, example '2021-06-10 12:00:00'")
	listExpFilesCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate), "file publish before endTime, example '2021-06-10 12:00:00'")
	listExpFilesCmd.Flags().Int64VarP(&limit, "limit", "l", blockchain.ListMaxNumber, "limit for list expired files")

	listExpFilesCmd.MarkFlagRequired("namespace")
}
