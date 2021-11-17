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

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
)

// getByIDCmd gets challenge by id
var getByIDCmd = &cobra.Command{
	Use:   "get",
	Short: "get pdp challenge by id",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		challenge, err := client.GetChallengeByID(context.Background(), id)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		fileOwner := hex.EncodeToString(challenge.FileOwner)
		targetNode := string(challenge.TargetNode)
		cTime := time.Unix(0, challenge.ChallengeTime).Format(timeTemplate)
		aTime := ""
		if challenge.AnswerTime != 0 {
			aTime = time.Unix(0, challenge.AnswerTime).Format(timeTemplate)
		}
		fmt.Printf("ID: %s\nFileOwner: %s\nTargetNode: %v\nFileID: %v\nStatus: %v\nChallengeTime: %v\nAnswerTime: %v\n\n",
			challenge.ID, fileOwner, targetNode, challenge.FileID, challenge.Status, cTime, aTime)
	},
}

func init() {
	rootCmd.AddCommand(getByIDCmd)

	getByIDCmd.Flags().StringVarP(&id, "id", "i", "", "challenge id")

	getByIDCmd.MarkFlagRequired("id")
}
