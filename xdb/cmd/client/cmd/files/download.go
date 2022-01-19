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
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

var (
	fileID string
	output string
)

// downloadCmd represents the command to download file from xuper db
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download the file from XuperDB",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		if len(fileID) == 0 && len(namespace) == 0 {
			fmt.Println("use namespace+filename or fileID to locate a file")
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

		opt := httpclient.ReadOptions{
			PrivateKey: privateKey,
			Namespace:  namespace,
			FileName:   filename,
			FileID:     fileID,
		}

		reader, err := client.Read(context.Background(), opt)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		defer reader.Close()

		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0777)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		defer f.Close()

		if _, err := io.Copy(f, reader); err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}

		fmt.Println("OK")
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "private key")
	downloadCmd.Flags().StringVarP(&keyPath, "keyPath", "", "./ukeys", "key path")
	downloadCmd.Flags().StringVarP(&output, "output", "o", "", "output file path")
	downloadCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace for file")
	downloadCmd.Flags().StringVarP(&filename, "filename", "m", "", "file name")
	downloadCmd.Flags().StringVarP(&fileID, "fileid", "f", "", "file id")

	downloadCmd.MarkFlagRequired("output")
}
