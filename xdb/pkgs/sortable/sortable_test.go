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

package sortable

import (
	"sort"
	"testing"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
)

func TestSortFiles(t *testing.T) {
	file1 := blockchain.File{
		ID:          "file1",
		PublishTime: 111,
	}
	file2 := blockchain.File{
		ID:          "file2",
		PublishTime: 222,
	}
	file3 := blockchain.File{
		ID:          "file3",
		PublishTime: 333,
	}
	files := Files{file2, file3, file1}
	sort.Sort(files)
	if files[0].ID != file3.ID || files[1].ID != file2.ID {
		t.Errorf("sort files failed")
	}
}

func TestSortNamespaces(t *testing.T) {
	ns1 := blockchain.NamespaceH{
		Namespace: blockchain.Namespace{
			Name:       "ns1",
			CreateTime: 111,
		},
	}
	ns2 := blockchain.NamespaceH{
		Namespace: blockchain.Namespace{
			Name:       "ns2",
			CreateTime: 222,
		},
	}
	ns3 := blockchain.NamespaceH{
		Namespace: blockchain.Namespace{
			Name:       "ns3",
			CreateTime: 333,
		},
	}
	ns := NamespaceHs{ns2, ns3, ns1}
	sort.Sort(ns)
	if ns[0].Namespace.Name != ns3.Namespace.Name || ns[1].Namespace.Name != ns2.Namespace.Name {
		t.Errorf("sort namespaces failed")
	}
}

func TestSortNodes(t *testing.T) {
	node1 := blockchain.Node{
		ID:      []byte("node1"),
		RegTime: 111,
	}
	node2 := blockchain.Node{
		ID:      []byte("node2"),
		RegTime: 222,
	}
	node3 := blockchain.Node{
		ID:      []byte("node3"),
		RegTime: 333,
	}
	nodes := Nodes{node1, node2, node3}
	sort.Sort(nodes)
	if string(nodes[0].ID) != string(node3.ID) || string(nodes[1].ID) != string(node2.ID) {
		t.Errorf("sort nodes failed")
	}
}

func TestSortChallenges(t *testing.T) {
	challenge1 := blockchain.Challenge{
		ID:            "challenge1",
		ChallengeTime: 111,
	}
	challenge2 := blockchain.Challenge{
		ID:            "challenge2",
		ChallengeTime: 222,
	}
	challenge3 := blockchain.Challenge{
		ID:            "challenge3",
		ChallengeTime: 333,
	}
	challenges := Challenges{challenge1, challenge3, challenge2}
	sort.Sort(challenges)
	if challenges[0].ID != challenge3.ID || challenges[1].ID != challenge2.ID {
		t.Errorf("sort challenges failed")
	}
}
