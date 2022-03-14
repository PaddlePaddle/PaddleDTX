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

// nsListCmd represents the command to list namespaces
var nsListCmd = &cobra.Command{
	Use:   "listns",
	Short: "list file namespaces of the DataOwner",
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

		opt := httpclient.ListNsOptions{
			Owner:     owner,
			TimeStart: startTime,
			TimeEnd:   endTime.UnixNano(),
			Limit:     limit,
		}

		response, err := client.ListFileNs(context.Background(), opt)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		// Print the list of ns, such as ns name, ns description, number of files, ns creation time, file replicas
		for _, ns := range response {
			ctime := time.Unix(0, ns.CreateTime).Format(timeTemplate)
			utime := time.Unix(0, ns.UpdateTime).Format(timeTemplate)
			fmt.Printf("Name: %s\nFileTotalNum: %d\nReplica: %d\nNsDescription: %s\nUpdateTime: %s\nCreateTime: %s\n\n",
				ns.Name, ns.FileTotalNum, ns.Replica, ns.Description, utime, ctime)
		}
		if len(response) == 0 {
			fmt.Printf("\nno ns, please add first\n\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(nsListCmd)

	nsListCmd.Flags().StringVarP(&owner, "owner", "o", "", "owner for file")
	nsListCmd.Flags().StringVarP(&start, "start", "s", "", "ns create after startTime, example '2021-06-10 12:00:00'")
	nsListCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate), "ns create before endTime, example '2021-06-10 12:00:00'")
	nsListCmd.Flags().Int64VarP(&limit, "limit", "l", blockchain.ListMaxNumber, "limit for list ns")

}
