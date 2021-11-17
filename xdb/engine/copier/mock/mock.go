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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

const (
	replica = 3
)

type MockCopier struct {
}

func New() *MockCopier {
	return &MockCopier{}
}

func (m *MockCopier) Select(slice slicer.Slice, healthNodes blockchain.NodeHs, opt *copier.SelectOptions) (
	copier.LocatedSlice, error) {
	if len(healthNodes) == 0 {
		return copier.LocatedSlice{}, errorx.New(errorx.ErrCodeInternal, "empty nodes array")
	}

	var nodes blockchain.Nodes
	for _, n := range healthNodes {
		nodes = append(nodes, n.Node)
	}
	if len(nodes) <= replica {
		ls := copier.LocatedSlice{
			Slice: slice,
			Nodes: nodes,
		}
		return ls, nil
	}

	rand.Seed(123456789)

	used := make(map[int]int)
	var selected blockchain.Nodes
	for {
		if len(selected) == replica {
			break
		}
		if len(selected) == len(nodes) {
			break
		}
		index := rand.Int() % len(nodes)
		if _, existed := used[index]; !existed {
			used[index] = 1
			selected = append(selected, nodes[index])
		}
	}

	ls := copier.LocatedSlice{
		Slice: slice,
		Nodes: selected,
	}
	return ls, nil
}

func (m *MockCopier) Push(ctx context.Context, id, sourceId string, r io.Reader, node *blockchain.Node) error {
	url := fmt.Sprintf("%s/v1/slice/push?slice_id=%s&source_id=%s", node.Address, id, sourceId)
	resp, err := http.Post(url, "application/octet-stream", r)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to push")
	}
	if resp.StatusCode != http.StatusOK {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "unexpected http status %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	// drop body
	// io.Copy(io.Discard, resp.Body)
	io.Copy(ioutil.Discard, resp.Body)
	return nil
}

func (m *MockCopier) Pull(ctx context.Context, id, fileId string, node *blockchain.Node) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/v1/slice/pull?slice_id=%s&file_id=%s", node.Address, id, fileId)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to pull")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "unexpected http status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (m *MockCopier) ReplicaExpansion(ctx context.Context, opt *copier.ReplicaExpOptions, enc common.MigrateEncryptor,
	ca, sourceId, fileId string) ([]blockchain.PublicSliceMeta, []encryptor.EncryptedSlice, error) {
	return nil, nil, nil
}
