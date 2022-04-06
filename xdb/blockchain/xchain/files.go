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
	"encoding/json"
	"strconv"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// PublishFile publishes file on xchain
func (x *XChain) PublishFile(opt *blockchain.PublishFileOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal PublishFileOptions")
	}
	args := map[string]string{
		"opt": string(s),
	}
	mName := "PublishFile"
	if _, err = x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// GetFileByName gets file by name from xchain
func (x *XChain) GetFileByName(owner []byte, ns, name string) (blockchain.File, error) {
	var f blockchain.File
	args := map[string]string{
		"owner":       string(owner),
		"ns":          ns,
		"name":        name,
		"currentTime": strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	mName := "GetFileByName"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return f, err
	}
	if err = json.Unmarshal([]byte(s), &f); err != nil {
		return f, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File")
	}

	return f, nil
}

// GetFileByID gets file by id from xchain
func (x *XChain) GetFileByID(id string) (blockchain.File, error) {
	var f blockchain.File
	args := map[string]string{
		"id":          id,
		"currentTime": strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	mName := "GetFileByID"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return f, err
	}

	if err = json.Unmarshal([]byte(s), &f); err != nil {
		return f, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File")
	}

	return f, nil
}

// UpdateFileExpireTime updates file expiration time
func (x *XChain) UpdateFileExpireTime(opt *blockchain.UpdateExptimeOptions) (blockchain.File, error) {
	var file blockchain.File
	s, err := json.Marshal(*opt)
	if err != nil {
		return file, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal UpdateFileExpireTime")
	}
	args := map[string]string{
		"opt": string(s),
	}
	mName := "UpdateFileExpireTime"
	resp, err := x.InvokeContract(args, mName)
	if err != nil {
		return file, err
	}

	if err = json.Unmarshal(resp, &file); err != nil {
		return file, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File")
	}
	return file, nil
}

// AddFileNs adds file namespace
func (x *XChain) AddFileNs(opt *blockchain.AddNsOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal AddNsOptions")
	}
	args := map[string]string{
		"opt": string(s),
	}
	mName := "AddFileNs"
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// UpdateNsReplica updates file namespace replica
func (x *XChain) UpdateNsReplica(opt *blockchain.UpdateNsReplicaOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal UpdateNsReplicaOptions")
	}
	args := map[string]string{
		"opt": string(s),
	}
	mName := "UpdateNsReplica"
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// UpdateFilePublicSliceMeta is used to update file public slice metas
func (x *XChain) UpdateFilePublicSliceMeta(opt *blockchain.UpdateFilePSMOptions) error {
	s, err := json.Marshal(*opt)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal UpdateFilePSMOptions")
	}
	args := map[string]string{
		"opt": string(s),
	}
	mName := "UpdateFilePublicSliceMeta"
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// SliceMigrateRecord is used by node to slice migration record
func (x *XChain) SliceMigrateRecord(id, sig []byte, fid, sid string, ctime int64) error {
	args := map[string]string{
		"nodeID":      string(id),
		"fileID":      fid,
		"sliceID":     sid,
		"signature":   string(sig),
		"currentTime": strconv.FormatInt(ctime, 10),
	}
	mName := "SliceMigrateRecord"
	if _, err := x.InvokeContract(args, mName); err != nil {
		return err
	}
	return nil
}

// ListFileNs lists file namespaces by owner
func (x *XChain) ListFileNs(opt *blockchain.ListNsOptions) ([]blockchain.Namespace, error) {
	var ns []blockchain.Namespace
	opts, err := json.Marshal(*opt)
	if err != nil {
		return ns, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal ListNsOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ListFileNs"
	resp, err := x.QueryContract(args, mName)
	if err != nil {
		return ns, err
	}
	if err = json.Unmarshal([]byte(resp), &ns); err != nil {
		return ns, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File")
	}
	return ns, nil
}

// GetNsByName gets namespace by nsName from xchain
func (x *XChain) GetNsByName(owner []byte, name string) (blockchain.Namespace, error) {
	var ns blockchain.Namespace
	args := map[string]string{
		"owner": string(owner),
		"name":  name,
	}
	mName := "GetNsByName"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return ns, err
	}

	if err = json.Unmarshal([]byte(s), &ns); err != nil {
		return ns, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File")
	}
	return ns, nil
}

// ListFiles lists files from xchain
func (x *XChain) ListFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error) {
	var fs []blockchain.File

	opts, err := json.Marshal(*opt)
	if err != nil {
		return fs, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal ListFileOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ListFiles"
	s, err := x.QueryContract(args, mName)
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
func (x *XChain) ListExpiredFiles(opt *blockchain.ListFileOptions) ([]blockchain.File, error) {
	var fs []blockchain.File

	opts, err := json.Marshal(*opt)
	if err != nil {
		return fs, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal ListFileOptions")
	}
	args := map[string]string{
		"opt": string(opts),
	}
	mName := "ListExpiredFiles"
	s, err := x.QueryContract(args, mName)
	if err != nil {
		return fs, err
	}
	if err = json.Unmarshal(s, &fs); err != nil {
		return fs, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal Files")
	}

	return fs, nil
}
