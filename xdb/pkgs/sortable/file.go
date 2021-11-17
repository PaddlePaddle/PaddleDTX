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
type Files []blockchain.File

func (f Files) Less(i, j int) bool {
	return f[i].PublishTime > f[j].PublishTime
}
func (f Files) Len() int {
	return len(f)
}
func (f Files) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type FileHs []blockchain.FileH

func (f FileHs) Less(i, j int) bool {
	return f[i].File.PublishTime > f[j].File.PublishTime
}
func (f FileHs) Len() int {
	return len(f)
}
func (f FileHs) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type Namespaces []blockchain.Namespace

func (f Namespaces) Less(i, j int) bool {
	return f[i].CreateTime > f[j].CreateTime
}
func (f Namespaces) Len() int {
	return len(f)
}
func (f Namespaces) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type NamespaceHs []blockchain.NamespaceH

func (f NamespaceHs) Less(i, j int) bool {
	return f[i].Namespace.CreateTime > f[j].Namespace.CreateTime
}
func (f NamespaceHs) Len() int {
	return len(f)
}
func (f NamespaceHs) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
