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

package mock

import (
	"encoding/json"
	"io/ioutil"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/testings"
)

const (
	persistentFileName = "mockchain.json"
)

type MockChain struct {
	Nodes      map[string]blockchain.Node
	Files      map[string]blockchain.File // filekey = uuid
	NsList     map[string]blockchain.Namespace
	Challenges map[string]blockchain.Challenge // challengekey = uuid

	FilenameIndexs         map[string]string // index_fn/owner/namespace/filename -> filekey
	ChallengeIndexs4Owner  map[string]string // index_cho/owner/targetnode/sliceid/start/end -> challengekey
	ChallengeIndexs4Target map[string]string // index_cht/targetnode/sliceid/start/end -> challengekey

	Persistent bool
}

type NewMockchainOptions struct {
	Persistent bool
}

func New(opt *NewMockchainOptions) *MockChain {
	var chain *MockChain

	// try recover from file
	bs, err := ioutil.ReadFile(persistentFileName)
	if err != nil || len(bs) == 0 {
		// init a new one
		chain = initChain()
	} else {
		// recover
		chain = &MockChain{
			Nodes:                  make(map[string]blockchain.Node),
			Files:                  make(map[string]blockchain.File),
			NsList:                 make(map[string]blockchain.Namespace),
			FilenameIndexs:         make(map[string]string),
			Challenges:             make(map[string]blockchain.Challenge),
			ChallengeIndexs4Owner:  make(map[string]string),
			ChallengeIndexs4Target: make(map[string]string),
		}
		if err := json.Unmarshal(bs, chain); err != nil {
			panic(err)
		}
	}
	chain.Persistent = opt.Persistent

	chain.persistent()
	return chain
}

func initChain() *MockChain {
	nodes := map[string]blockchain.Node{
		testings.PK1: {
			ID:      mustToHex(testings.PK1),
			Name:    "node1",
			Address: "http://node1:80",
			Online:  true,
		},
		testings.PK2: {
			ID:      mustToHex(testings.PK2),
			Name:    "node2",
			Address: "http://node2:80",
			Online:  true,
		},
		testings.PK3: {
			ID:      mustToHex(testings.PK3),
			Name:    "node3",
			Address: "http://node3:80",
			Online:  true,
		},
		testings.PK4: {
			ID:      mustToHex(testings.PK4),
			Name:    "node4",
			Address: "http://node4:80",
			Online:  true,
		},
	}

	return &MockChain{
		Nodes:                  nodes,
		Files:                  make(map[string]blockchain.File),
		FilenameIndexs:         make(map[string]string),
		Challenges:             make(map[string]blockchain.Challenge),
		ChallengeIndexs4Owner:  make(map[string]string),
		ChallengeIndexs4Target: make(map[string]string),
	}
}

func (mc *MockChain) persistent() {
	if !mc.Persistent {
		return
	}

	bs, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(persistentFileName, bs, 0600); err != nil {
		panic(err)
	}
}
