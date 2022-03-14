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

package key

import (
	"fmt"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/dai/util/file"
)

var output string

// rootCmd represents the root command
var rootCmd = &cobra.Command{
	Use:   "key",
	Short: "generate the requester client private/public key pair",
}

func RootCmd() *cobra.Command {
	return rootCmd
}

// genKeyCmd randomly generates node private/public key pair
var genKeyCmd = &cobra.Command{
	Use:   "genkey",
	Short: "generate a pair of key",
	Run: func(cmd *cobra.Command, args []string) {
		prikey, pubkey, err := ecdsa.GenerateKeyPair()
		if err != nil {
			fmt.Printf("failed to GenerateKeyPair, err：%v\n", err)
			return
		}
		fmt.Println("private-key:", prikey)
		fmt.Println("public-key:", pubkey)
		err = file.WriteFile(output, file.PublicKeyFileName, []byte(pubkey.String()))
		if err != nil {
			fmt.Printf("failed to save public.key, err: %v\n", err)
			return
		}
		err = file.WriteFile(output, file.PrivateKeyFileName, []byte(prikey.String()))
		if err != nil {
			fmt.Printf("failed to save private.key, err: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(genKeyCmd)

	genKeyCmd.Flags().StringVarP(&output, "output", "o", "./keys", "output")

	genKeyCmd.MarkFlagRequired("output")
}
