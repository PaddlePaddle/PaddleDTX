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

package random

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
)

func TestBSearch(t *testing.T) {
	nodes := []nodeWeight{
		{nodeID: []byte{1}, start: 1, end: 1},
		{nodeID: []byte{2}, start: 2, end: 2},
		{nodeID: []byte{3}, start: 3, end: 3},
	}
	idFound := bSearch(nodes, 2)
	require.Equal(t, idFound, []byte{2})
	idFound = bSearch(nodes, 1)
	require.Equal(t, idFound, []byte{1})
	idFound = bSearch(nodes, 3)
	require.Equal(t, idFound, []byte{3})

	nodes = []nodeWeight{
		{nodeID: []byte{1}, start: 1, end: 1},
	}
	idFound = bSearch(nodes, 1)
	require.Equal(t, idFound, []byte{1})

	nodes = []nodeWeight{
		{nodeID: []byte{1}, start: 0, end: 0},
		{nodeID: []byte{2}, start: 1, end: 1},
		{nodeID: []byte{3}, start: 2, end: 2},
		{nodeID: []byte{4}, start: 3, end: 3},
	}
	idFound = bSearch(nodes, 1)
	require.Equal(t, idFound, []byte{2})
}

func TestBSearch2(t *testing.T) {
	nodes := []nodeWeight{
		{nodeID: []byte{1}, start: 0, end: 9},
		{nodeID: []byte{2}, start: 10, end: 19},
		{nodeID: []byte{3}, start: 20, end: 29},
		{nodeID: []byte{4}, start: 30, end: 39},
	}

	idFound := bSearch(nodes, 17)
	require.Equal(t, idFound, []byte{2})
	idFound = bSearch(nodes, 6)
	require.Equal(t, idFound, []byte{1})
	idFound = bSearch(nodes, 20)
	require.Equal(t, idFound, []byte{3})
	idFound = bSearch(nodes, 39)
	require.Equal(t, idFound, []byte{4})
}

func TestSelection(t *testing.T) {
	c := &RandomCopier{}

	slice := slicer.Slice{}
	slice.ID = "hello"
	slice.Data = []byte("0a0b")

	// select only one good node
	node1 := blockchain.NodeH{
		Node: blockchain.Node{
			ID: []byte{1},
		},
		Health: blockchain.NodeHealthGood,
	}
	node2 := blockchain.NodeH{
		Node: blockchain.Node{
			ID: []byte{2},
		},
		Health: blockchain.NodeHealthBad,
	}
	nodes := blockchain.NodeHs{node1, node2}
	ls, err := c.Select(slice, nodes, &copier.SelectOptions{Replica: 1})
	require.NoError(t, err)
	require.Equal(t, 1, len(ls.Nodes))
	require.Equal(t, ls.Slice, slice)

	// select two good nodes
	node3 := blockchain.NodeH{
		Node: blockchain.Node{
			ID: []byte{3},
		},
		Health: blockchain.NodeHealthGood,
	}
	nodes = blockchain.NodeHs{node1, node2, node3}
	ls, err = c.Select(slice, nodes, &copier.SelectOptions{Replica: 2})
	require.NoError(t, err)
	require.Equal(t, 2, len(ls.Nodes))

	// select three nodes, good and medium
	node3.Node.Online = true
	node4 := blockchain.NodeH{
		Node: blockchain.Node{
			ID:     []byte{4},
			Online: true,
		},
		Health: blockchain.NodeHealthMedium,
	}
	nodes = blockchain.NodeHs{node1, node2, node3, node4}
	ls, err = c.Select(slice, nodes, &copier.SelectOptions{Replica: 3})
	require.NoError(t, err)
	require.Equal(t, 3, len(ls.Nodes))
}
