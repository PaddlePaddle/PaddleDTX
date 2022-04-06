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

var (
	applier string
	status  string
)

// fileAuthsListCmd represents the command to query the list of authorization applications
var fileAuthsListCmd = &cobra.Command{
	Use:   "listauth",
	Short: "list file authorization applications",
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
		opt := httpclient.ListFileAuthOptions{
			Owner:     owner,
			Status:    status,
			Applier:   applier,
			FileID:    fileID,
			TimeStart: startTime,
			TimeEnd:   endTime.UnixNano(),
			Limit:     limit,
		}

		response, err := client.ListFileAuths(context.Background(), opt)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		// Print each authorization application detailed information under the list
		for _, fa := range response {
			ctime := time.Unix(0, fa.CreateTime).Format(timeTemplate)
			atime := time.Unix(0, fa.ApprovalTime).Format(timeTemplate)
			etime := time.Unix(0, fa.ExpireTime).Format(timeTemplate)

			fmt.Printf("AuthID: %s\nFileID: %s\nName: %s\nDescription: %s\nApplier: %x\nAuthorizer: %x\nAuthKey: %x\nStatus: %v\n",
				fa.ID, fa.FileID, fa.Name, fa.Description, fa.Applier, fa.Authorizer, fa.AuthKey, fa.Status)

			fmt.Printf("RejectReason: %s\nCreateTime: %s\nApprovalTime: %s\nExpireTime: %s\n\n", fa.RejectReason, ctime, atime, etime)
		}
		if len(response) == 0 {
			fmt.Printf("\nno file authorization applications\n\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(fileAuthsListCmd)

	fileAuthsListCmd.Flags().StringVarP(&applier, "applier", "a", "", "applier's public key")
	fileAuthsListCmd.Flags().StringVarP(&owner, "owner", "o", "", "file owner")
	fileAuthsListCmd.Flags().StringVarP(&fileID, "fileID", "f", "", "file ID")
	fileAuthsListCmd.Flags().StringVar(&status, "status", "", "status of file authorization application, example 'Unapproved, Approved or Rejected'")
	fileAuthsListCmd.Flags().StringVarP(&start, "start", "s", "", "authorization applications publish after startTime, example '2022-06-10 12:00:00'")
	fileAuthsListCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate),
		"authorization applications publish before endTime, example '2022-07-10 12:00:00'")
	fileAuthsListCmd.Flags().Int64VarP(&limit, "limit", "l", blockchain.ListMaxNumber, "limit for list file authorization applications")
}
