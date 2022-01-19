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
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

var (
	input       string
	description string
	extra       string
	expireTime  string
)

// uploadDataCmd represents the command to upload file into xuper db
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "save a file into XuperDB",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		f, err := os.OpenFile(input, os.O_RDONLY, 0600)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		defer f.Close()

		stamp, err := time.ParseInLocation(timeTemplate, expireTime, time.Local)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		if privateKey == "" {
			privateKeyBytes, err := file.ReadFile(keyPath, file.PrivateKeyFileName)
			if err != nil {
				fmt.Printf("Read privateKey failed, err: %v\n", err)
				return
			}
			privateKey = strings.TrimSpace(string(privateKeyBytes))
		}

		opt := httpclient.WriteOptions{
			PrivateKey:  privateKey,
			Namespace:   namespace,
			FileName:    filename,
			ExpireTime:  stamp.UnixNano(),
			Description: description,
			Extra:       extra,
		}

		resp, err := client.Write(context.Background(), f, opt)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		fmt.Println("FileID:", resp.FileID)
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	uploadCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "private key")
	uploadCmd.Flags().StringVarP(&keyPath, "keyPath", "", "./ukeys", "key path")
	uploadCmd.Flags().StringVarP(&input, "input", "i", "", "input file path")
	uploadCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace for file")
	uploadCmd.Flags().StringVarP(&filename, "filename", "m", "", "file name")
	uploadCmd.Flags().StringVarP(&description, "description", "d", "", "file description")
	uploadCmd.Flags().StringVarP(&expireTime, "expireTime", "e", "", "expire time, example '2021-06-10 12:00:00'")
	uploadCmd.Flags().StringVar(&extra, "ext", "", "file extra info")

	uploadCmd.MarkFlagRequired("input")
	uploadCmd.MarkFlagRequired("namespace")
	uploadCmd.MarkFlagRequired("filename")
	uploadCmd.MarkFlagRequired("expireTime")
}
