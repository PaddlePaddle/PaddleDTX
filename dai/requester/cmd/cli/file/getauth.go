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

package file

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	requestClient "github.com/PaddlePaddle/PaddleDTX/dai/requester/client"
)

var authID string

// getAuthCmd represents the command to get file authorization application detail
var getAuthCmd = &cobra.Command{
	Use:   "getauthbyid",
	Short: "get the file authorization application detail",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requestClient.GetRequestClient(configPath)
		if err != nil {
			fmt.Printf("GetRequestClient failed: %v\n", err)
			return
		}

		fa, err := client.GetFileAuthByID(authID)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		ctime := time.Unix(0, fa.CreateTime).Format(timeTemplate)
		atime := time.Unix(0, fa.ApprovalTime).Format(timeTemplate)
		etime := time.Unix(0, fa.ExpireTime).Format(timeTemplate)

		fmt.Printf("AuthID: %s\nFileID: %s\nName: %s\nDescription: %s\nApplier: %x\nAuthorizer: %x\nAuthKey: %x\nStatus: %v\n",
			fa.ID, fa.FileID, fa.Name, fa.Description, fa.Applier, fa.Authorizer, fa.AuthKey, fa.Status)
		fmt.Printf("RejectReason: %s\nCreateTime: %s\nApprovalTime: %s\nExpireTime: %s\n\n", fa.RejectReason, ctime, atime, etime)
	},
}

func init() {
	rootCmd.AddCommand(getAuthCmd)

	getAuthCmd.Flags().StringVarP(&authID, "authID", "i", "", "id of file authorization application")

	getAuthCmd.MarkFlagRequired("authID")
}
