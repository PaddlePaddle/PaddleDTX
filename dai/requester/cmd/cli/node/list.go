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

package node

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	requestClient "github.com/PaddlePaddle/PaddleDTX/dai/requester/client"
)

// listNodesCmd list executor nodes
var listNodesCmd = &cobra.Command{
	Use:   "list",
	Short: "list executor nodes",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requestClient.GetRequestClient(configPath)
		if err != nil {
			fmt.Printf("GetRequestClient failed: %v\n", err)
			return
		}
		resp, err := client.ListExecutorNodes()
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return
		}

		for _, n := range resp {
			rtime := time.Unix(0, n.RegTime).Format(timeTemplate)
			fmt.Printf("NodeID: %x\nName: %s\nAddress: %s\nRegisterTime: %v\n\n", n.ID, n.Name, n.Address, rtime)
		}
		if len(resp) == 0 {
			fmt.Printf("\nThere are no executor nodes in the network\n\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(listNodesCmd)
}
