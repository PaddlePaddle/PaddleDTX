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

package challenge

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
)

// toProveCmd gets toProve challenge by filters
var toProveCmd = &cobra.Command{
	Use:   "toprove",
	Short: "get ToProve challenges by filters",
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

		opt := httpclient.GetChallengesOptions{
			Owner:      owner,
			TargetNode: storageNode,
			FileID:     fileID,
			TimeStart:  startTime,
			TimeEnd:    endTime.UnixNano(),
			Limit:      limit,
		}
		challenges, err := client.GetToProveChallenges(context.Background(), opt)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		if list != 0 {
			for _, c := range challenges {
				fileOwner := hex.EncodeToString(c.FileOwner)
				cTime := time.Unix(0, c.ChallengeTime).Format(timeTemplate)
				fmt.Printf("ChallengeID: %s\nFileID: %s\nOwner: %s\nStorageNode: %s\nChallengeTime: %s\n\n",
					c.ID, c.FileID, fileOwner, string(c.TargetNode), cTime)
			}
		}
		fmt.Printf("ToProve challenges from %s to %s\nNum: %d\n\n", start, end, len(challenges))
	},
}

func init() {
	rootCmd.AddCommand(toProveCmd)

	toProveCmd.Flags().StringVarP(&owner, "owner", "o", "", "file owner")
	toProveCmd.Flags().StringVarP(&storageNode, "node", "n", "", "storage node")
	toProveCmd.Flags().StringVarP(&fileID, "file", "f", "", "file ID")
	toProveCmd.Flags().StringVarP(&start, "start", "s", "", "challenge before startTime, example '2021-06-10 12:00:00'")
	toProveCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate), "challenge after endTime, example '2021-06-10 12:00:00'")
	toProveCmd.Flags().Int64VarP(&limit, "limit", "l", blockchain.ListMaxNumber, "limit")
	toProveCmd.Flags().Int8VarP(&list, "list", "", 1, "show challenges list or not, 0 not to show")

	toProveCmd.MarkFlagRequired("node")
}
