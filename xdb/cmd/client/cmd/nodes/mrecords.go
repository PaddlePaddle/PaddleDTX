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
	"time"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
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

		resp, err := client.GetMigrateRecords(context.Background(), id, startTime, endTime.UnixNano(), limit)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		for _, v := range resp {
			cTime := time.Unix(0, int64(v["ctime"].(float64))).Format(timeTemplate)
			fmt.Printf("\nFileId: %s, SliceId: %s, migrateTime: %s\n", v["fileId"], v["sliceId"], cTime)
		}
		fmt.Printf("\nslice migrate total number: %d\n\n", len(resp))
	},
}

func init() {
	rootCmd.AddCommand(getMigrateRecordsCmd)

	getMigrateRecordsCmd.Flags().StringVarP(&id, "id", "i", "", "id")
	getMigrateRecordsCmd.Flags().StringVarP(&start, "start", "s", "", "slice migrate startTime, example '2021-06-10 12:00:00'")
	getMigrateRecordsCmd.Flags().StringVarP(&end, "end", "e", time.Unix(0, time.Now().UnixNano()).Format(timeTemplate), "slice migrate endTime, example '2021-06-10 12:00:00'")
	getMigrateRecordsCmd.Flags().Uint64VarP(&limit, "limit", "l", 0, "limit for list slice migrate records")

	getMigrateRecordsCmd.MarkFlagRequired("host")
	getMigrateRecordsCmd.MarkFlagRequired("id")
}
