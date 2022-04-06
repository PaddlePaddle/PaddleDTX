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

package nodes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

// getMigrateRecordsCmd represents the command to get node slices migration records
var getMigrateRecordsCmd = &cobra.Command{
	Use:   "mrecords",
	Short: "get node slice migrate records",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		var startTime int64 = 0
		if start != "" {
			s, err := time.ParseInLocation(timeTemplate, start, time.Local)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				return
			}
			startTime = s.UnixNano()
		}
		endTime, err := time.ParseInLocation(timeTemplate, end, time.Local)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		if limit > blockchain.ListMaxNumber {
			fmt.Printf("invalid limit, the value must smaller than %v \n", blockchain.ListMaxNumber)
			return
		}

		if id == "" {
			pubKeyBytes, err := file.ReadFile(keyPath, file.PublicKeyFileName)
			if err != nil {
				fmt.Printf("Read publicKey failed, err: %v\n", err)
				return
			}
			id = strings.TrimSpace(string(pubKeyBytes))
		}

		resp, err := client.GetMigrateRecords(context.Background(), id, startTime, endTime.UnixNano(), limit)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		for _, v := range resp {
			cTime := time.Unix(0, int64(v["ctime"].(float64))).Format(timeTemplate)
			fmt.Printf("\nfileID: %s, SliceID: %s, migrateTime: %s\n", v["fileID"], v["sliceID"], cTime)
		}
		fmt.Printf("\nslice migrate total number: %d\n\n", len(resp))
	},
}

func init() {
	rootCmd.AddCommand(getMigrateRecordsCmd)

	getMigrateRecordsCmd.Flags().StringVarP(&id, "id", "i", "", "id")
	getMigrateRecordsCmd.Flags().StringVarP(&keyPath, "keyPath", "", file.KeyFilePath, "node's key path")
	getMigrateRecordsCmd.Flags().StringVarP(&start, "start", "s", "", "slice migrate startTime, example '2021-06-10 12:00:00'")
	getMigrateRecordsCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate), "slice migrate endTime, example '2021-06-10 12:00:00'")
	getMigrateRecordsCmd.Flags().Int64VarP(&limit, "limit", "l", blockchain.ListMaxNumber, "limit for list slice migrate records")
}
