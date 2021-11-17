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
	"sort"
	"strings"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

func (mc *MockChain) PublishFile(ctx context.Context, opt *blockchain.PublishFileOptions) error {
	defer mc.persistent()

	mc.Files[opt.File.ID] = opt.File

	filenameIndex := packFilenameIndex(opt.File.Owner, opt.File.Namespace, opt.File.Name)
	mc.FilenameIndexs[filenameIndex] = opt.File.ID

	return nil
}

func (mc *MockChain) GetFileByName(ctx context.Context,
	owner []byte, ns, name string) (blockchain.File, error) {

	index := packFilenameIndex(owner, ns, name)
	id, exist := mc.FilenameIndexs[index]
	if !exist {
		return blockchain.File{}, errorx.ErrNotFound
	}

	f, found := mc.Files[id]
	if !found {
		panic("bug!!!")
	}

	return f, nil
}

func (mc *MockChain) GetFileByID(ctx context.Context, id string) (blockchain.File, error) {
	f, exist := mc.Files[id]
	if !exist {
		return f, errorx.ErrNotFound
	}

	return f, nil
}

//UpdateExpTime update file expiretime
func (mc *MockChain) UpdateFileExpireTime(ctx context.Context, opt *blockchain.UpdatExptimeOptions) (blockchain.File, error) {
	f, exist := mc.Files[opt.FileId]
	f.ExpireTime = opt.NewExpireTime
	mc.Files[opt.FileId] = f
	if !exist {
		return f, errorx.ErrNotFound
	}
	return f, nil
}

func (mc *MockChain) UpdateNsFilesCap(ctx context.Context, opt *blockchain.UpdateNsFilesCapOptions) (ns blockchain.Namespace, err error) {
	mc.NsList[opt.Name] = blockchain.Namespace{}
	return mc.NsList[opt.Name], nil
}

func (mc *MockChain) AddFileNs(ctx context.Context, opt *blockchain.AddNsOptions) error {
	mc.NsList[opt.Namespace.Name] = blockchain.Namespace{}
	return nil
}

func (mc *MockChain) UpdateNsReplica(ctx context.Context, opt *blockchain.UpdateNsReplicaOptions) error {
	mc.NsList[opt.Name] = blockchain.Namespace{}
	return nil
}

func (mc *MockChain) UpdateFilePublicSliceMeta(ctx context.Context, opt *blockchain.UpdateFilePSMOptions) error {
	return nil
}

func (mc *MockChain) SliceMigrateRecord(ctx context.Context, id, sig []byte, fid, sid string, ctime int64) error {
	return nil
}

func (mc *MockChain) ListFileNs(ctx context.Context, opt *blockchain.ListNsOptions) ([]blockchain.Namespace, error) {
	_, exist := mc.NsList[opt.Namespace]
	if !exist {
		return nil, errorx.ErrNotFound
	}
	return nil, nil
}

func (mc *MockChain) GetNsByName(ctx context.Context, owner []byte, name string) (blockchain.Namespace, error) {
	ns, exist := mc.NsList[name]
	if !exist {
		return ns, errorx.ErrNotFound
	}
	return ns, nil
}

func (mc *MockChain) ListFiles(ctx context.Context, opt *blockchain.ListFileOptions) (
	[]blockchain.File, error) {

	prefix := packFilenameFilter(opt.Owner, opt.Namespace)

	// sort
	indexArray := make([]indexTuple, 0, len(mc.FilenameIndexs))
	for idx, fileID := range mc.FilenameIndexs {
		if strings.HasPrefix(idx, prefix) {
			indexArray = append(indexArray, indexTuple{Index: idx, ID: fileID})
		}
	}
	sort.Sort(sortableIndexTuple(indexArray))

	// iterate
	var fs []blockchain.File
	for _, tuple := range indexArray {
		if opt.Limit > 0 && uint64(len(fs)) >= opt.Limit {
			break
		}
		f, ok := mc.Files[tuple.ID]
		if !ok {
			panic("should never happen")
		}
		fs = append(fs, f)

	}

	return fs, nil
}

func (mc *MockChain) ListExpiredFiles(ctx context.Context, opt *blockchain.ListFileOptions) (
	[]blockchain.File, error) {

	return nil, nil
}
