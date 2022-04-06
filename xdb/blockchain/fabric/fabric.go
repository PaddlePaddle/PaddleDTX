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
	"os"
	"strings"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"

	conf "github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// Config configures fabric driver
type Config struct {
	ConfigFile  string // ConfigFile is path of sdk configuration file
	ChannelID   string // ChannelID indicates Channel id
	ChaincodeID string // ChaincodeID indicates name of ChainCode installed on Channel
	UserName    string // UserName is name of user used to request fabric network
	OrgName     string // OrgName indicates the organization User belongs to, with which to request fabric network
}

type Fabric struct {
	Config        *Config
	Sdk           *fabsdk.FabricSDK // Sdk is Fabric SDK instance
	ChannelClient *channel.Client   // ChannelClient is client to request channel
	ledgerClient  *ledger.Client    // ledgerClient is client to request ledger
}

// New new Fabric
func New(conf *conf.FabricConf) (*Fabric, error) {
	if len(conf.ConfigFile) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing fabric configuration file")
	}
	if len(conf.ChannelID) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing channel id")
	}
	if len(conf.Chaincode) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing chaincode")
	}
	if len(conf.UserName) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing user name")
	}
	if len(conf.OrgName) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing org name")
	}

	// check config file
	_, err := os.Stat(conf.ConfigFile)
	if !(err == nil || os.IsExist(err)) {
		return nil, errorx.New(errorx.ErrCodeConfig, "fabric sdk configuration file not exist")
	}

	fabricConfig := &Config{
		ConfigFile:  conf.ConfigFile,
		ChannelID:   conf.ChannelID,
		ChaincodeID: conf.Chaincode,
		UserName:    conf.UserName,
		OrgName:     conf.OrgName,
	}
	fabricDriver := &Fabric{Config: fabricConfig}

	// initiate Sdk
	sdk, err := fabsdk.New(config.FromFile(conf.ConfigFile))
	if err != nil {
		return nil, errorx.Wrap(err, "failed to create fabric sdk")
	}
	fabricDriver.Sdk = sdk

	// initiate channelClient
	clientContext := sdk.ChannelContext(
		conf.ChannelID,
		fabsdk.WithUser(conf.UserName),
		fabsdk.WithOrg(conf.OrgName))
	channelClient, err := channel.New(clientContext)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to create channel client")
	}
	fabricDriver.ChannelClient = channelClient

	// initiate ledgerClient
	fabricDriver.ledgerClient, err = ledger.New(clientContext)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to create new ledger client")
	}

	return fabricDriver, err
}

// InvokeContract invokes the contract
func (f *Fabric) InvokeContract(args [][]byte, mName string) ([]byte, error) {
	channelReq := channel.Request{
		ChaincodeID: f.Config.ChaincodeID,
		Fcn:         mName,
		Args:        args,
	}

	response, err := f.ChannelClient.Execute(channelReq, channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		indexStart := strings.Index(err.Error(), "{")
		indexStop := strings.Index(err.Error(), "}")
		if indexStart > 0 && indexStop > 0 && indexStop >= indexStart {
			message := err.Error()[indexStart : indexStop+1]
			if len(message) > 0 {
				if c, m, ok := errorx.TryParseFromString(message); ok {
					return nil, errorx.Wrap(errorx.New(c, m), "failed to execute chaincode")
				}
			}
		}
		return nil, errorx.ParseAndWrap(err, "failed to execute chaincode")
	}
	return response.Payload, nil
}

// QueryContract queries the contract
func (f *Fabric) QueryContract(args [][]byte, mName string) ([]byte, error) {
	channelReq := channel.Request{
		ChaincodeID: f.Config.ChaincodeID,
		Fcn:         mName,
		Args:        args,
	}

	response, err := f.ChannelClient.Query(channelReq, channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		indexStart := strings.Index(err.Error(), "{")
		indexStop := strings.Index(err.Error(), "}")
		if indexStart > 0 && indexStop > 0 && indexStop >= indexStart {
			message := err.Error()[indexStart : indexStop+1]
			if len(message) > 0 {
				if c, m, ok := errorx.TryParseFromString(message); ok {
					return nil, errorx.Wrap(errorx.New(c, m), "failed to Query Contract")
				}
			}
		}
		return nil, errorx.ParseAndWrap(err, "failed to Query Contract")
	}
	return response.Payload, nil
}
