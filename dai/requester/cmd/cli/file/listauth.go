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
	"encoding/hex"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	requestClient "github.com/PaddlePaddle/PaddleDTX/dai/requester/client"
	xdbchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var (
	applier string
	owner   string
	fileId  string
	status  string
	start   string
	end     string
	limit   int64
)

// fileAuthsListCmd represents the command to query the list of authorization applications
var fileAuthsListCmd = &cobra.Command{
	Use:   "listauth",
	Short: "list file authorization applications",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requestClient.GetRequestClient(configPath)
		if err != nil {
			fmt.Printf("GetRequestClient failed: %v\n", err)
			return
		}
		// Get query file authorization application list parameters
		opt, err := getListAuthOptions()
		if err != nil {
			fmt.Printf("getListAuthOptions err：: %v\n", err)
			return
		}

		response, err := client.ListFileAuthApplications(opt)
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

// getListAuthOptions get query file authorization application list parameters
// Verify and return the parameters required for the contract call
func getListAuthOptions() (*xdbchain.ListFileAuthOptions, error) {
	var startTime int64 = 0
	if start != "" {
		s, err := time.ParseInLocation(timeTemplate, start, time.Local)
		if err != nil {
			return nil, err
		}
		startTime = s.UnixNano()
	}
	endTime, err := time.ParseInLocation(timeTemplate, end, time.Local)
	if err != nil {
		return nil, err
	}
	opt := xdbchain.ListFileAuthOptions{
		Status:    status,
		FileID:    fileId,
		TimeStart: startTime,
		TimeEnd:   endTime.UnixNano(),
		Limit:     limit,
	}
	// Applier and file's owner can not be empty at the same time
	if applier == "" && owner == "" {
		return nil, errorx.New(errorx.ErrCodeParam, "owner or applier can not by empty")
	}
	if applier != "" {
		pubkey, err := hex.DecodeString(applier)
		if err != nil {
			return nil, errorx.New(errorx.ErrCodeParam, "failed to decode file applier's public key, err: %v", err)
		}
		opt.Applier = pubkey[:]
	}
	if owner != "" {
		pubkey, err := hex.DecodeString(owner)
		if err != nil {
			return nil, errorx.New(errorx.ErrCodeParam, "failed to decode file owner's public key, err: %v", err)
		}
		opt.Authorizer = pubkey[:]
	}
	// Limit cannot exceed the maximum limit of 100
	if opt.Limit > xdbchain.ListMaxNumber {
		fmt.Printf("invalid limit, the value must less than %v \n", xdbchain.ListMaxNumber)
		return nil, errorx.New(errorx.ErrCodeParam, "invalid limit, the value must smaller than %v", xdbchain.ListMaxNumber)
	}
	return &opt, nil
}

func init() {
	rootCmd.AddCommand(fileAuthsListCmd)

	fileAuthsListCmd.Flags().StringVarP(&applier, "applier", "a", "", "applier's public key, often known as executor's public key")
	fileAuthsListCmd.Flags().StringVarP(&owner, "owner", "o", "", "file owner")
	fileAuthsListCmd.Flags().StringVarP(&fileId, "fileID", "f", "", "sample file ID")
	fileAuthsListCmd.Flags().StringVar(&status, "status", "", "status of file authorization application")
	fileAuthsListCmd.Flags().StringVarP(&start, "start", "s", "", "authorization applications publish after startTime, example '2022-06-10 12:00:00'")
	fileAuthsListCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate),
		"authorization applications publish before endTime, example '2022-07-10 12:00:00'")
	fileAuthsListCmd.Flags().Int64VarP(&limit, "limit", "l", xdbchain.ListMaxNumber, "limit for list file authorization applications")
}
