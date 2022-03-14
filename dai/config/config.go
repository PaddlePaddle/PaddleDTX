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
	"strings"

	"github.com/spf13/viper"

	"github.com/PaddlePaddle/PaddleDTX/dai/util/file"
)

var (
	logConf      *Log
	executorConf *ExecutorConf
	cliConf      *ExecutorBlockchainConf
)

type ExecutorConf struct {
	Name          string
	ListenAddress string
	PublicAddress string
	PrivateKey    string
	KeyPath       string
	Mode          *ExecutorModeConf
	Mpc           *ExecutorMpcConf
	Storage       *ExecutorStorageConf
	Blockchain    *ExecutorBlockchainConf
}

type ExecutorModeConf struct {
	Type string
	Self *XuperDBConf
}

type ExecutorMpcConf struct {
	TrainTaskLimit   int
	PredictTaskLimit int
	RpcTimeout       int
	TaskConcurrency  int
	TaskLimitTime    int
}

type ExecutorStorageConf struct {
	Type                       string
	LocalModelStoragePath      string
	LocalEvaluationStoragePath string
	XuperDB                    *XuperDBConf
	Local                      *PredictLocalConf
}

type XuperDBConf struct {
	PrivateKey string
	Host       string
	KeyPath    string
	NameSpace  string
	ExpireTime int64
}

type PredictLocalConf struct {
	LocalPredictStoragePath string
}

type ExecutorBlockchainConf struct {
	Type   string
	Xchain *XchainConf
}

type XchainConf struct {
	Mnemonic        string
	ContractName    string
	ContractAccount string
	ChainAddress    string
	ChainName       string
}

type Log struct {
	Level string
	Path  string
}

// InitConfig parses configuration file
func InitConfig(configPath string) error {
	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return err
	}
	logConf = new(Log)
	err := v.Sub("log").Unmarshal(logConf)
	if err != nil {
		return err
	}
	executorConf = new(ExecutorConf)
	err = v.Sub("executor").Unmarshal(executorConf)
	if err != nil {
		return err
	}
	if executorConf.PrivateKey == "" {
		privateKeyBytes, err := file.ReadFile(executorConf.KeyPath, file.PrivateKeyFileName)
		if err == nil && len(privateKeyBytes) != 0 {
			executorConf.PrivateKey = strings.TrimSpace(string(privateKeyBytes))
		} else {
			return err
		}
	}
	return nil
}

// InitCliConfig parses client configuration file. if cli's configuration file is not existed, use executor's configuration file.
func InitCliConfig(configPath string) error {
	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return err
	}
	innerV := v.Sub("blockchain")
	if innerV != nil {
		// If "blockchain" was existed, cli would use the configuration of cli.
		cliConf = new(ExecutorBlockchainConf)
		err := innerV.Unmarshal(cliConf)
		if err != nil {
			return err
		}
		return nil
	} else {
		// If "blockchain" wasn't existed, use the configuration of the executor.
		err := InitConfig(configPath)
		if err == nil {
			cliConf = executorConf.Blockchain
		}
		return err
	}
}

func GetExecutorConf() *ExecutorConf {
	return executorConf
}

func GetLogConf() *Log {
	return logConf
}

func GetCliConf() *ExecutorBlockchainConf {
	return cliConf
}
