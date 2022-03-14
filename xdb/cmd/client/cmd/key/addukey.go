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
	"encoding/hex"
	"fmt"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/spf13/cobra"

	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

var userPubkey string

// addUKeyCmd adds the user's public key into the whitelist
var addUKeyCmd = &cobra.Command{
	Use:   "addukey",
	Short: "used for the dataOwner node to add client's public key into the whitelist",
	Run: func(cmd *cobra.Command, args []string) {
		pubkey, err := ecdsa.DecodePublicKeyFromString(userPubkey)
		if err != nil {
			fmt.Printf("failed to save private.key, err: %v\n", err)
			return
		}
		userKeyFileName := hash.HashUsingSha256([]byte(pubkey.String()))
		err = file.WriteFile(output, hex.EncodeToString(userKeyFileName), []byte(pubkey.String()))
		if err != nil {
			fmt.Printf("failed to grant authorization of the ukeys, err: %v\n", err)
			return
		}
		fmt.Println("OK")
	},
}

func init() {
	rootCmd.AddCommand(addUKeyCmd)

	addUKeyCmd.Flags().StringVarP(&userPubkey, "user", "u", "", "user public key")
	addUKeyCmd.Flags().StringVarP(&output, "output", "o", file.AuthKeyFilePath, "output")

	addUKeyCmd.MarkFlagRequired("user")
}
