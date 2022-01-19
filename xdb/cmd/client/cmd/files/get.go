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

// getByIDCmd represents the command to get file by id
var getByIDCmd = &cobra.Command{
	Use:   "getbyid",
	Short: "get the file by id from XuperDB",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		hf, err := client.GetFileByID(context.Background(), id)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		f := hf.File
		slicesMap := getFileSliceMap(f)
		ptime := time.Unix(0, f.PublishTime).Format(timeTemplate)
		etime := time.Unix(0, f.ExpireTime).Format(timeTemplate)
		fmt.Printf("FileID: %s\nOwner: %x\nFileName: %s\nFileDescription: %s\nNamespace: %s\nSlicesMap: %v\nFileLength: %v\n",
			f.ID, f.Owner, f.Name, f.Description, f.Namespace, slicesMap, f.Length)
		fmt.Printf("Health: %s\nPublishTime: %s\nExpireTime: %s\nExtra: %s\n\n", hf.Health, ptime, etime, f.Ext)
	},
}

// getByNameCmd represents the get file by ns+name command
var getByNameCmd = &cobra.Command{
	Use:   "getbyname",
	Short: "get the file by name from XuperDB",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		hf, err := client.GetFileByName(context.Background(), owner, namespace, filename)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		f := hf.File
		slicesMap := getFileSliceMap(f)
		ptime := time.Unix(0, f.PublishTime).Format(timeTemplate)
		etime := time.Unix(0, f.ExpireTime).Format(timeTemplate)
		fmt.Printf("FileID: %s\nFileDescription: %s\nSlicesMap: %v\nFileLength: %v\nHealth: %s\nPublishTimes: %s\nExpireTime: %s\nExtra: %s\n\n",
			f.ID, f.Description, slicesMap, f.Length, hf.Health, ptime, etime, f.Ext)
	},
}

func getFileSliceMap(file blockchain.File) map[string][]string {
	ret := make(map[string][]string)
	for _, slice := range file.Slices {
		ret[slice.ID] = append(ret[slice.ID], string(slice.NodeID))
	}
	return ret
}

func init() {
	rootCmd.AddCommand(getByIDCmd)
	rootCmd.AddCommand(getByNameCmd)

	getByIDCmd.Flags().StringVarP(&id, "id", "i", "", "id for file")

	getByNameCmd.Flags().StringVarP(&owner, "owner", "o", "", "owner for file")
	getByNameCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace for file")
	getByNameCmd.Flags().StringVarP(&filename, "filename", "m", "", "file name")

	getByIDCmd.MarkFlagRequired("id")

	getByNameCmd.MarkFlagRequired("namespace")
	getByNameCmd.MarkFlagRequired("filename")
}
