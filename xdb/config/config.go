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

package config

import (
	"errors"
	"strings"

	"github.com/spf13/viper"

	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/file"
)

// The application can be running as a dataOwner node or a storage node.
const NodeTypeDataOwner = "dataOwner"
const NodeTypeStorage = "storage"

var (
	// distinguish the mode of the application
	serverType string
	logConf    *Log
	// the configuration when running as a dataOwner node
	dataOwnerConf *DataOwnerConf
	// the configuration when running as a storage node
	storageConf *StorageConf
)

type BlockchainConf struct {
	Type   string
	Xchain *XchainConf
	Fabric *FabricConf
}

type XchainConf struct {
	Mnemonic        string
	ContractName    string
	ContractAccount string
	ChainAddress    string
	ChainName       string
}

type FabricConf struct {
	ConfigFile string
	ChannelID  string
	Chaincode  string
	UserName   string
	OrgName    string
}

type MonitorConf struct {
	ChallengingSwitch    string
	NodemaintainerSwitch string
	FileclearInterval    int
	FilemaintainerSwitch string
	FilemigrateInterval  int
}

type ServerConf struct {
	Name          string
	ListenAddress string
	PrivateKey    string
	PublicAddress string
}

type Log struct {
	Level string
	Path  string
}

// InitConfig, load and parses configuration file
func InitConfig(config string) error {
	v := viper.New()
	v.SetConfigFile(config)
	if err := v.ReadInConfig(); err != nil {
		return err
	}
	var err error
	logConf = new(Log)
	err = v.Sub("log").Unmarshal(logConf)
	if err != nil {
		return err
	}
	serverType = v.Get("type").(string)
	if serverType == NodeTypeDataOwner {
		dataOwnerConf = new(DataOwnerConf)
		err = v.Sub(NodeTypeDataOwner).Unmarshal(dataOwnerConf)
	} else if serverType == NodeTypeStorage {
		storageConf = new(StorageConf)
		err = v.Sub(NodeTypeStorage).Unmarshal(storageConf)
	} else {
		return errors.New("unSupported Node Type")
	}
	return err
}

// GetServerType
func GetServerType() string {
	return serverType
}

// GetStorageConf
func GetStorageConf() *StorageConf {
	return storageConf
}

// GetDataOwnerConf
func GetDataOwnerConf() *DataOwnerConf {
	return dataOwnerConf
}

// GetLogConf
func GetLogConf() *Log {
	return logConf
}

// GetServerConf
func GetServerConf() *ServerConf {
	var privateKey string

	if serverType == NodeTypeDataOwner {
		privateKey = dataOwnerConf.PrivateKey
		privateKeyBytes, err := file.ReadFile(dataOwnerConf.KeyPath, file.PrivateKeyFileName)
		if err == nil && len(privateKeyBytes) != 0 {
			privateKey = strings.TrimSpace(string(privateKeyBytes))
		}
		return &ServerConf{
			Name:          dataOwnerConf.Name,
			ListenAddress: dataOwnerConf.ListenAddress,
			PrivateKey:    privateKey,
			PublicAddress: dataOwnerConf.PublicAddress}
	} else if serverType == NodeTypeStorage {
		privateKey = storageConf.PrivateKey
		privateKeyBytes, err := file.ReadFile(storageConf.KeyPath, file.PrivateKeyFileName)
		if err == nil && len(privateKeyBytes) != 0 {
			privateKey = strings.TrimSpace(string(privateKeyBytes))
		}

		return &ServerConf{
			Name:          storageConf.Name,
			ListenAddress: storageConf.ListenAddress,
			PrivateKey:    privateKey,
			PublicAddress: storageConf.PublicAddress}
	} else {
		return nil
	}
}

// GetBlockchainConf
func GetBlockchainConf() *BlockchainConf {
	if serverType == NodeTypeDataOwner {
		return dataOwnerConf.Blockchain
	} else if serverType == NodeTypeStorage {
		return storageConf.Blockchain
	} else {
		return nil
	}
}
