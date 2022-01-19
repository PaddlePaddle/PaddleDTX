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

// listNodesCmd list storage nodes
var listNodesCmd = &cobra.Command{
	Use:   "list",
	Short: "list storage nodes",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}
		resp, err := client.ListNodes(context.Background())
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}

		for _, n := range resp {
			rtime := time.Unix(0, n.RegTime).Format(timeTemplate)
			utime := time.Unix(0, n.UpdateAt).Format(timeTemplate)
			fmt.Printf("NodeID: %s\nName: %s\nAddress: %s\nOnline: %v\nRegisterTime: %v\nUpdateTime: %v\n\n", n.ID, n.Name, n.Address, n.Online, rtime, utime)
		}
		if len(resp) == 0 {
			fmt.Printf("\nThere are no storage nodes in the network\n\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(listNodesCmd)
}
