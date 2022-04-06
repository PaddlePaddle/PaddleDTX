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
	"context"
	"strings"

	"github.com/xuperchain/xuper-sdk-go/account"
	"github.com/xuperchain/xuper-sdk-go/contract"
	"github.com/xuperchain/xuper-sdk-go/pb"
	"github.com/xuperchain/xuper-sdk-go/xchain"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

type XChain struct {
	ContractName    string              // ContractName is name of contract
	ContractAccount string              // ContractAccount is a contract account
	ChainName       string              // ChainName is name of blockchain
	Account         *account.Account    // Account is the local account, and is also the client account to request blockchain
	XchainClient    *xchain.XuperClient // XchainClient is the util used to connect and request blockchain
}

// New creates a XChain client which is used for connecting and requesting blockchain
func New(conf *config.XchainConf) (*XChain, error) {
	if len(conf.Mnemonic) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing mnemonic")
	}
	if len(conf.ContractName) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing contract-name")
	}
	if len(conf.ContractAccount) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing contract-account")
	}
	if len(conf.ChainAddress) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing chain-node")
	}
	if len(conf.ChainName) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing chain-name")
	}
	// get xchain account using mnemonic
	acc, err := account.RetrieveAccount(conf.Mnemonic, 1)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"RetrieveAccount failed, err: %v", err)
	}
	// create xchain client
	xuperClient, err := xchain.NewXuperClient(conf.ChainAddress)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"NewXuperClient failed, err: %v", err)
	}
	return &XChain{
		ContractName:    conf.ContractName,
		ContractAccount: conf.ContractAccount,
		ChainName:       conf.ChainName,
		Account:         acc,
		XchainClient:    xuperClient,
	}, nil
}

// InvokeContract invokes the contract
func (x *XChain) InvokeContract(args map[string]string, mName string) ([]byte, error) {
	// initiate client for native contract
	nativeContract := contract.InitNativeContractWithClient(
		x.Account, x.ChainName, x.ContractName, x.ContractAccount, x.XchainClient)
	// pre-invoke contract
	preSelectUTXOResponse, err := nativeContract.PreInvokeNativeContract(mName, args)
	if err != nil {
		// handle error
		indexStart := strings.Index(err.Error(), "{")
		indexStop := strings.Index(err.Error(), "}")
		if indexStart > 0 && indexStop > 0 && indexStop >= indexStart {
			message := err.Error()[indexStart : indexStop+1]
			if len(message) > 0 {
				if c, m, ok := errorx.TryParseFromString(message); ok {
					return nil, errorx.Wrap(errorx.New(c, m), "failed to PreInvokeNativeContract")
				}
			}
		}
		return nil, errorx.ParseAndWrap(err, "failed to PreInvokeNativeContract")
	}
	// invoke contract
	if _, err := nativeContract.PostNativeContract(preSelectUTXOResponse); err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal,
			"PostNativeContract failed, err: %v", err)
	}
	return preSelectUTXOResponse.GetResponse().GetResponses()[0].GetBody(), nil
}

// QueryContract queries the contract
func (x *XChain) QueryContract(args map[string]string, mName string) ([]byte, error) {
	// initiate client for native contract
	nativeContract := contract.InitNativeContractWithClient(
		x.Account, x.ChainName, x.ContractName, x.ContractAccount, x.XchainClient)
	// send request
	resp, err := nativeContract.QueryNativeContract(mName, args)
	if err != nil {
		// handle error
		indexStart := strings.Index(err.Error(), "{")
		indexStop := strings.Index(err.Error(), "}")
		if indexStart > 0 && indexStop > 0 && indexStop >= indexStart {
			message := err.Error()[indexStart : indexStop+1]
			if len(message) > 0 {
				if c, m, ok := errorx.TryParseFromString(message); ok {
					return nil, errorx.Wrap(errorx.New(c, m), "failed to QueryNativeContract")
				}
			}
		}
		return nil, errorx.ParseAndWrap(err, "failed to QueryNativeContract")
	}
	return resp.GetResponse().GetResponses()[0].GetBody(), nil
}

// GetRootAndLatestBlockIdInChain gets latest block height
func (x *XChain) GetRootAndLatestBlockIdInChain() ([]byte, int64, error) {
	systemStatus, err := x.XchainClient.XchainClient.GetSystemStatus(context.Background(), &pb.CommonIn{})
	if err != nil {
		return nil, 0, err
	}
	for _, value := range systemStatus.SystemsStatus.BcsStatus {
		if value.Bcname == x.ChainName {
			return value.Meta.RootBlockid, value.Meta.TrunkHeight, nil
		}
	}
	return nil, 0, errorx.New(errorx.ErrCodeInternal, "can't find blockid")
}
