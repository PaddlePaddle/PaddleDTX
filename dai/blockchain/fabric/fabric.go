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
	fabricchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain/fabric"
	xdbconfig "github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
)

var logger = logrus.WithField("module", "xchain")

type Config struct {
	fabricchain.Config
}

type Fabric struct {
	fabricchain.Fabric
}

// New creates a XChain client used for connecting and requesting blockchain
func New(conf *config.FabricConf) (*Fabric, error) {
	config := &xdbconfig.FabricConf{
		ConfigFile: conf.ConfigFile,
		ChannelID:  conf.ChannelID,
		Chaincode:  conf.Chaincode,
		UserName:   conf.UserName,
		OrgName:    conf.OrgName,
	}

	fa, err := fabricchain.New(config)

	if err != nil {
		return nil, err
	}
	return &Fabric{*fa}, nil
}

func (f *Fabric) Close() {
}
