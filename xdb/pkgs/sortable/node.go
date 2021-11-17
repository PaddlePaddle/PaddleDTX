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
	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
)

// descending order
type Nodes blockchain.Nodes

func (n Nodes) Less(i, j int) bool {
	return n[i].RegTime > n[j].RegTime
}
func (n Nodes) Len() int {
	return len(n)
}
func (n Nodes) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

type NodeHs blockchain.NodeHs

func (n NodeHs) Less(i, j int) bool {
	return n[i].Node.RegTime > n[j].Node.RegTime
}
func (n NodeHs) Len() int {
	return len(n)
}
func (n NodeHs) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}
