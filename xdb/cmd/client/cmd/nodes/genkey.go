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
	"encoding/base64"
	"fmt"

	fl_crypto "github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/spf13/cobra"
)

var xchainClient = new(fl_crypto.XchainCryptoClient)

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
	},
}

// genPDPKeyCmd randomly generates pdp keys
var genPDPKeyCmd = &cobra.Command{
	Use:   "genpdpkeys",
	Short: "generate pdp keys",
	Run: func(cmd *cobra.Command, args []string) {
		pdpPriv, pdpPub, err := xchainClient.GenPDPRandomKeyPair()
		if err != nil {
			fmt.Printf("failed to generate PDP key pair, err: %v\n", err)
			return
		}
		randu, err := xchainClient.RandomPDPWithinOrder()
		if err != nil {
			fmt.Printf("failed to generate randomU, err: %v\n", err)
			return
		}
		randv, err := xchainClient.RandomPDPWithinOrder()
		if err != nil {
			fmt.Printf("failed to generate randomV, err: %v\n", err)
			return
		}

		fmt.Println("pdp-sk:", base64.StdEncoding.EncodeToString(pdpPriv))
		fmt.Println("pdp-pk:", base64.StdEncoding.EncodeToString(pdpPub))
		fmt.Println("pdp-randu:", base64.StdEncoding.EncodeToString(randu))
		fmt.Println("pdp-randv:", base64.StdEncoding.EncodeToString(randv))
	},
}

func init() {
	rootCmd.AddCommand(genKeyCmd)
	rootCmd.AddCommand(genPDPKeyCmd)
}
