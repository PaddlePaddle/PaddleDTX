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

package fabric

import (
	"github.com/sirupsen/logrus"

	fabricblockchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain/fabric"
	xdataconfig "github.com/PaddlePaddle/PaddleDTX/xdb/config"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
)

var logger = logrus.WithField("module", "xchain")

type Config struct {
	fabricblockchain.Config
}

type Fabric struct {
	fabricblockchain.Fabric
}

// New creates a XChain client used for connecting and requesting blockchain
func New(conf *config.FabricConf) (*Fabric, error) {
	config := &xdataconfig.FabricConf{
		ConfigFile:  conf.ConfigFile,
		ChannelID:   conf.ChannelID,
		ChaincodeID: conf.Chaincode,
		UserName:    conf.UserName,
		OrgName:     conf.OrgName,
	}

	fa, err := fabricblockchain.New(config)

	if err != nil {
		return nil, err
	}
	return &Fabric{*fa}, nil
}

func (f *Fabric) Close() {
	if err := f.Fabric.FabricClient.FabricConn.Close(); err != nil {
		logger.WithError(err).Error("failed to close fabric client")
	}
	if err := f.Fabric.FabricClient.XendorserConn.Close(); err != nil {
		logger.WithError(err).Error("failed to close endorser client")
	}
	logger.Info("close fabric client")
}
