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
	"strings"
	"time"

	"github.com/spf13/cobra"

	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

var (
	rejectReason string
)

// confirmAuthCmd represents the command to confirm applier's file authorization application
var confirmAuthCmd = &cobra.Command{
	Use:   "confirmauth",
	Short: "confirm the applier's file authorization application",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
		if err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
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
		opt := httpclient.ConfirmAuthOptions{
			PrivateKey: privateKey,
			AuthID:     authID,
			ExpireTime: stamp.UnixNano(),
			Status:     true,
		}
		if err := client.ConfirmOrRejectAuth(context.Background(), opt); err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		fmt.Println("OK")
	},
}

// rejectAuthCmd represents the command to reject applier's file authorization application
var rejectAuthCmd = &cobra.Command{
	Use:   "rejectauth",
	Short: "reject the applier's file authorization application",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := httpclient.New(host)
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
		opt := httpclient.ConfirmAuthOptions{
			PrivateKey:   privateKey,
			AuthID:       authID,
			RejectReason: rejectReason,
			Status:       false,
		}
		if err := client.ConfirmOrRejectAuth(context.Background(), opt); err != nil {
			fmt.Printf("err：%v\n", err)
			return
		}
		fmt.Println("OK")
	},
}

func init() {
	rootCmd.AddCommand(confirmAuthCmd)
	rootCmd.AddCommand(rejectAuthCmd)

	confirmAuthCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "private key")
	confirmAuthCmd.Flags().StringVarP(&keyPath, "keyPath", "", "./ukeys", "key path")
	confirmAuthCmd.Flags().StringVarP(&authID, "authID", "i", "", "id for file authorization application")
	confirmAuthCmd.Flags().StringVarP(&expireTime, "expireTime", "e", "", "expire time, example '2022-07-10 12:00:00'")

	rejectAuthCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "private key")
	rejectAuthCmd.Flags().StringVarP(&keyPath, "keyPath", "", "./ukeys", "key path")
	rejectAuthCmd.Flags().StringVarP(&authID, "authID", "i", "", "id for file authorization application")
	rejectAuthCmd.Flags().StringVarP(&rejectReason, "rejectReason", "r", "", "reason for reject the authorization")

	confirmAuthCmd.MarkFlagRequired("authID")
	confirmAuthCmd.MarkFlagRequired("expireTime")

	rejectAuthCmd.MarkFlagRequired("authID")
	rejectAuthCmd.MarkFlagRequired("rejectReason")
}
