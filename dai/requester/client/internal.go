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

package client

import (
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
)

// checkConfig checks configuration file
func checkConfig(configPath string) error {
	err := config.InitCliConfig(configPath)
	if err != nil {
		return errorx.New(errorx.ErrCodeConfig, "load config file failed: %s", err)
	}
	cliConf := config.GetCliConf()
	// only support xchain
	blockchainType := cliConf.Type
	if blockchainType != "xchain" {
		return errorx.New(errorx.ErrCodeConfig, "invalid blockchain type: %s", blockchainType)
	}
	return nil
}

// checkUserPrivateKey checks whether user public key is a valid ecc public key
func checkUserPrivateKey(privateKey string) (pubkey ecdsa.PublicKey, privkey ecdsa.PrivateKey, err error) {
	privkey, err = ecdsa.DecodePrivateKeyFromString(privateKey)
	if err != nil {
		return pubkey, privkey, err
	}
	pubkey = ecdsa.PublicKeyFromPrivateKey(privkey)
	return pubkey, privkey, nil
}
