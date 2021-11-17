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

package xchain

import (
	"github.com/sirupsen/logrus"

	xchainblockchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain/xchain"
	xdataconfig "github.com/PaddlePaddle/PaddleDTX/xdb/config"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
)

var logger = logrus.WithField("module", "xchain")

type XChain struct {
	xchainblockchain.XChain
}

// New creates a XChain client used for connecting and requesting blockchain
func New(conf *config.XchainConf) (*XChain, error) {
	config := &xdataconfig.XchainConf{
		Mnemonic:        conf.Mnemonic,
		ContractName:    conf.ContractName,
		ContractAccount: conf.ContractAccount,
		ChainAddress:    conf.ChainAddress,
		ChainName:       conf.ChainName,
	}
	xc, err := xchainblockchain.New(config)
	if err != nil {
		return nil, err
	}
	return &XChain{*xc}, nil
}

// Close closes client
func (x *XChain) Close() {
	if err := x.XChain.XchainClient.XchainConn.Close(); err != nil {
		logger.WithError(err).Error("failed to close xchain client")
	}
	if err := x.XChain.XchainClient.XendorserConn.Close(); err != nil {
		logger.WithError(err).Error("failed to close endorser client")
	}
	logger.Info("close xchain client")
}
