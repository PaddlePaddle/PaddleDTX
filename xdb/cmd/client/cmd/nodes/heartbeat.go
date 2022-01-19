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

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

// getNodeHeartbeatCmd represents the command to get number of heartbeats
var getNodeHeartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "get node heartbeat num by id",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		var ctime int64
		if start != "" {
			tamp, err := time.ParseInLocation(timeTemplate, start, time.Local)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				return
			}
			ctime = tamp.UnixNano()
		}
		if id == "" {
			pubKeyBytes, err := file.ReadFile(keyPath, file.PublicKeyFileName)
			if err != nil {
				fmt.Printf("Read publicKey failed, err: %v\n", err)
				return
			}
			id = strings.TrimSpace(string(pubKeyBytes))
		}

		hmp, err := client.GetNodeHeartbeat(context.Background(), id, ctime)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		fmt.Printf("NodeID: %s\nHeartBeatTotal: %d\nHeartBeatMaxNum: %d\n", id, hmp["heartBeatTotal"], hmp["heartBeatMax"])
	},
}

func init() {
	rootCmd.AddCommand(getNodeHeartbeatCmd)

	getNodeHeartbeatCmd.Flags().StringVarP(&id, "id", "i", "", "id")
	getNodeHeartbeatCmd.Flags().StringVarP(&keyPath, "keyPath", "", file.KeyFilePath, "node's key path")
	getNodeHeartbeatCmd.Flags().StringVarP(&start, "ctime", "c", "", "heartbeatnum of given day, example '2021-07-10 12:00:00'")
}
