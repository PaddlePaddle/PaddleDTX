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

package fabric

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// PublishFile publishes file onto fabric
func (f *Fabric) PublishFile(opt *blockchain.PublishFileOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal PublishFileOptions")
	}

	if _, err = f.InvokeContract([][]byte{s}, "PublishFile"); err != nil {
		return err
	}
	return nil
}

// GetFileByName gets file by name from fabric
func (f *Fabric) GetFileByName(owner []byte, ns, name string) (blockchain.File, error) {
	var file blockchain.File

	args := [][]byte{owner, []byte(ns), []byte(name), []byte(strconv.FormatInt(time.Now().UnixNano(), 10))}
	s, err := f.QueryContract(args, "GetFileByName")
	if err != nil {
		return file, err
	}
	if err = json.Unmarshal(s, &file); err != nil {
		return file, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File")
	}

	return file, nil
}

// GetFileByID gets file by id from fabric
func (f *Fabric) GetFileByID(id string) (blockchain.File, error) {
	var file blockchain.File
	args := [][]byte{[]byte(id), []byte(strconv.FormatInt(time.Now().UnixNano(), 10))}
	s, err := f.QueryContract(args, "GetFileByID")
	if err != nil {
		return file, err
	}

	if err = json.Unmarshal(s, &file); err != nil {
		return file, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File")
	}

	return file, nil
}

// UpdateFileExpireTime updates file expiration time
func (f *Fabric) UpdateFileExpireTime(opt *blockchain.UpdateExptimeOptions) (blockchain.File, error) {
	var file blockchain.File
	s, err := json.Marshal(*opt)
	if err != nil {
		return file, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal UpdateFileExpireTime")
	}

	resp, err := f.InvokeContract([][]byte{s}, "UpdateFileExpireTime")
	if err != nil {
		return file, err
	}

	if err = json.Unmarshal(resp, &file); err != nil {
		return file, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File")
	}
	return file, nil
}

// AddFileNs adds file namespace
func (f *Fabric) AddFileNs(opt *blockchain.AddNsOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal AddNsOptions")
	}

	if _, err := f.InvokeContract([][]byte{s}, "AddFileNs"); err != nil {
		return err
	}
	return nil
}

// UpdateNsReplica updates file namespace replica
func (f *Fabric) UpdateNsReplica(opt *blockchain.UpdateNsReplicaOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal UpdateNsReplicaOptions")
	}

	if _, err := f.InvokeContract([][]byte{s}, "UpdateNsReplica"); err != nil {
		return err
	}
	return nil
}

// UpdateFilePublicSliceMeta is used to update file public slice metas
func (f *Fabric) UpdateFilePublicSliceMeta(opt *blockchain.UpdateFilePSMOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal UpdateFilePSMOptions")
	}

	if _, err := f.InvokeContract([][]byte{s}, "UpdateFilePublicSliceMeta"); err != nil {
		return err
	}
	return nil
}

// SliceMigrateRecord is used by node to slice migration record
func (f *Fabric) SliceMigrateRecord(id, sig []byte, fid, sid string, ctime int64) error {
	args := [][]byte{id, []byte(fid), []byte(sid), sig, []byte(strconv.FormatInt(ctime, 10))}
	if _, err := f.InvokeContract(args, "SliceMigrateRecord"); err != nil {
		return err
	}
	return nil
}

// ListFileNs lists file namespaces by owner
func (f *Fabric) ListFileNs(opt *blockchain.ListNsOptions) ([]blockchain.Namespace, error) {
	var ns []blockchain.Namespace
	opts, err := json.Marshal(*opt)
	if err != nil {
		return ns, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal ListNsOptions")
	}

	resp, err := f.QueryContract([][]byte{opts}, "ListFileNs")
	if err != nil {
		return ns, err
	}
	if err = json.Unmarshal(resp, &ns); err != nil {
		return ns, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File")
	}
	return ns, nil
}

// GetNsByName gets namespace by nsName from fabric
func (f *Fabric) GetNsByName(owner []byte, name string) (blockchain.Namespace, error) {
	var ns blockchain.Namespace
	args := [][]byte{owner, []byte(name)}
	s, err := f.QueryContract(args, "GetNsByName")
	if err != nil {
		return ns, err
	}

	if err = json.Unmarshal(s, &ns); err != nil {
		return ns, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File")
	}
	return ns, nil
}

// ListFiles lists files from fabric
func (f *Fabric) ListFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error) {
	var fs []blockchain.File

	opts, err := json.Marshal(*opt)
	if err != nil {
		return fs, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal ListFileOptions")
	}

	s, err := f.QueryContract([][]byte{opts}, "ListFiles")
	if err != nil {
		return fs, err
	}
	if err = json.Unmarshal(s, &fs); err != nil {
		return fs, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal Files")
	}

	return fs, nil
}

// ListExpiredFiles lists expired but valid files
func (f *Fabric) ListExpiredFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error) {
	var fs []blockchain.File

	opts, err := json.Marshal(*opt)
	if err != nil {
		return fs, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal ListFileOptions")
	}

	s, err := f.QueryContract([][]byte{opts}, "ListExpiredFiles")
	if err != nil {
		return fs, err
	}
	if err = json.Unmarshal(s, &fs); err != nil {
		return fs, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal Files")
	}

	return fs, nil
}
