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

package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// PublishFile publishes file on xchain
func (x *Xdata) PublishFile(ctx code.Context) code.Response {
	// get PublishFileOptions
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:file"))
	}
	// unmarshal opt
	var opt blockchain.PublishFileOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal PublishFileOptions"))
	}
	// get file
	f := opt.File

	// marshal file
	s, err := json.Marshal(f)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal File"))
	}

	// verify sig
	err = x.checkSign(opt.Signature, f.Owner, s)
	if err != nil {
		return code.Error(err)
	}
	// judge if slices is empty
	if len(f.Slices) == 0 {
		return code.Error(errorx.New(errorx.ErrCodeParam,
			"slices is empty when publishing file"))
	}
	// judge if id exists
	if _, err := ctx.GetObject([]byte(opt.File.ID)); err == nil {
		return code.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated fileID"))
	}

	// judge if fileNsIndex exists
	fileNsIndex := packFileNsIndex(f.Owner, f.Namespace)
	nsr, err := ctx.GetObject([]byte(fileNsIndex))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "file namespace not found"))
	}
	var ns blockchain.Namespace
	if err = json.Unmarshal(nsr, &ns); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal namespace"))
	}

	// judge if filenameIndex exists
	filenameIndex := packFileNameIndex(f.Owner, f.Namespace, f.Name)
	if id, err := ctx.GetObject([]byte(filenameIndex)); err == nil {
		fc, err := x.getFileByID(ctx, id)
		if err != nil {
			return code.Error(err)
		}
		if fc.ExpireTime+blockchain.FileRetainPeriod.Nanoseconds() > opt.File.PublishTime {
			return code.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated file name"))
		}
	}

	// if there's already a file with the same name in user's storage, overwrite it with the new one directly
	// set id-file on chain
	if err := ctx.PutObject([]byte(f.ID), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set id-file on chain"))
	}
	// set filenameIndex-id on chain
	if err := ctx.PutObject([]byte(filenameIndex), []byte(f.ID)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set index-id on chain"))
	}
	// set filenameListIndex-id on chain
	filenameListIndex := packFileNameListIndex(f.Owner, f.Namespace, f.Name, f.PublishTime)
	if err := ctx.PutObject([]byte(filenameListIndex), []byte(f.ID)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set file listIndex-id on chain"))
	}

	// update file num of fileNsIndex
	ns.FileTotalNum += 1
	ns.UpdateTime = f.PublishTime
	nsf, err := json.Marshal(ns)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal File namespace"))
	}
	if err := ctx.PutObject([]byte(fileNsIndex), nsf); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to update index-ns on chain"))
	}
	nsListIndex := packFileNsListIndex(ns.Owner, ns.Name, ns.CreateTime)
	if err := ctx.PutObject([]byte(nsListIndex), nsf); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to update listIndex-ns on chain"))
	}

	// set node-sliceID-expireTime on chain
	nodeSice := make(map[string][]string)
	for _, slice := range f.Slices {
		nodeSice[string(slice.NodeID)] = append(nodeSice[string(slice.NodeID)], slice.ID)
	}
	for nodeID, sliceL := range nodeSice {
		prefixNodeFileSlice := packNodeSliceIndex(nodeID, f)
		if err := ctx.PutObject([]byte(prefixNodeFileSlice), []byte(strings.Join(sliceL, ","))); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set index-id on chain"))
		}
	}

	return code.OK([]byte("Published"))
}

// AddFileNs adds file namespace
func (x *Xdata) AddFileNs(ctx code.Context) code.Response {
	// get AddFileNs
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.AddNsOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal AddNsOptions"))
	}

	ns := opt.Namespace
	s, err := json.Marshal(ns)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal namespace"))
	}
	// verify sig
	err = x.checkSign(opt.Signature, ns.Owner, s)
	if err != nil {
		return code.Error(err)
	}

	// judge if fileNsIndex exists
	fileNsIndex := packFileNsIndex(ns.Owner, ns.Name)
	// get file ns
	if _, err := ctx.GetObject([]byte(fileNsIndex)); err == nil {
		return code.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated file namespace"))
	}

	// put index-ns on chain
	if err := ctx.PutObject([]byte(fileNsIndex), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set index-ns on chain"))
	}
	// put listIndex-ns on chain
	nsListIndex := packFileNsListIndex(ns.Owner, ns.Name, ns.CreateTime)
	if err := ctx.PutObject([]byte(nsListIndex), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set listIndex-ns on chain"))
	}

	return code.OK([]byte("OK"))
}

// UpdateNsReplica updates file namespace replica
func (x *Xdata) UpdateNsReplica(ctx code.Context) code.Response {
	// get opt
	o, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.UpdateNsReplicaOptions
	if err := json.Unmarshal(o, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal UpdateNsReplicaOptions"))
	}
	// verify sig
	m := fmt.Sprintf("%s,%d,%d", opt.Name, opt.Replica, opt.CurrentTime)
	if err := x.checkSign(opt.Signature, opt.Owner, []byte(m)); err != nil {
		return code.Error(err)
	}

	//get file ns
	fileNsIndex := packFileNsIndex(opt.Owner, opt.Name)
	nsr, err := ctx.GetObject([]byte(fileNsIndex))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "file namespace not found"))
	}
	var n blockchain.Namespace
	if err = json.Unmarshal(nsr, &n); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal namespace"))
	}
	if n.Replica >= opt.Replica {
		return code.Error(errorx.New(errorx.ErrCodeParam, "bad param:replica"))
	}

	n.Replica = opt.Replica
	n.UpdateTime = opt.CurrentTime
	s, err := json.Marshal(n)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal Namespaces"))
	}
	// put index-ns on chain
	if err := ctx.PutObject([]byte(fileNsIndex), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to update nsindex-replica on chain"))
	}
	// put listIndex-ns on chain
	nsListIndex := packFileNsListIndex(n.Owner, n.Name, n.CreateTime)
	if err := ctx.PutObject([]byte(nsListIndex), s); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to update nsListIndex-replica on chain"))
	}
	return code.OK([]byte("OK"))
}

// UpdateFilePublicSliceMeta is used to update file public slice metas
func (x *Xdata) UpdateFilePublicSliceMeta(ctx code.Context) code.Response {
	// get opt
	o, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.UpdateFilePSMOptions
	if err := json.Unmarshal(o, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal UpdateNsReplicaOptions"))
	}
	// verify sig
	ns := opt.Slices
	s, err := json.Marshal(ns)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal slices"))
	}
	if err := x.checkSign(opt.Signature, opt.Owner, s); err != nil {
		return code.Error(err)
	}

	// get file from id
	fs, err := ctx.GetObject([]byte(opt.FileID))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "file not found"))
	}
	var f blockchain.File
	if err = json.Unmarshal(fs, &f); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File"))
	}
	if string(f.Owner) != string(opt.Owner) {
		return code.Error(errorx.New(errorx.ErrCodeNotAuthorized, "bad param, file owner is wrong"))
	}
	// update slices
	f.Slices = opt.Slices
	nfs, err := json.Marshal(f)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal Namespaces"))
	}

	if err := ctx.PutObject([]byte(f.ID), nfs); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set id-file on chain"))
	}
	return code.OK([]byte("OK"))
}

// GetFileByName gets file by name from xchain
func (x *Xdata) GetFileByName(ctx code.Context) code.Response {
	// get owner
	owner, ok := ctx.Args()["owner"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:owner"))
	}
	// get ns
	ns, ok := ctx.Args()["ns"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:ns"))
	}
	// get name
	name, ok := ctx.Args()["name"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:name"))
	}
	ctime, err := x.getCtxTime(ctx, "currentTime")
	if err != nil {
		return code.Error(err)
	}
	// judge if fileNsIndex exists
	fileNsIndex := packFileNsIndex(owner, string(ns))
	if _, err := ctx.GetObject([]byte(fileNsIndex)); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeParam, "bad param, ns not found"))
	}
	// pack filenameindex
	index := packFileNameIndex(owner, string(ns), string(name))
	// get id from index
	id, err := ctx.GetObject([]byte(index))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "fileID not found with name+ns"))
	}
	// get file from id
	s, err := ctx.GetObject(id)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "file not found"))
	}

	var f blockchain.File
	if err = json.Unmarshal(s, &f); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File"))
	}
	if f.ExpireTime < ctime {
		return code.Error(errorx.New(errorx.ErrCodeNotFound, "file already expire"))
	}
	return code.OK(s)
}

// GetFileByID gets file by id from xchain
func (x *Xdata) GetFileByID(ctx code.Context) code.Response {
	// get id
	id, ok := ctx.Args()["id"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}
	ctime, err := x.getCtxTime(ctx, "currentTime")
	if err != nil {
		return code.Error(err)
	}
	// get file from id
	fs, err := ctx.GetObject(id)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "file not found"))
	}
	var f blockchain.File
	if err = json.Unmarshal(fs, &f); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File"))
	}
	if f.ExpireTime < ctime {
		return code.Error(errorx.New(errorx.ErrCodeExpired, "file already expire"))
	}
	return code.OK(fs)
}

// UpdateFileExpireTime updates file expiration time
func (x *Xdata) UpdateFileExpireTime(ctx code.Context) code.Response {
	// get UpdateExptimeOptions
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.UpdateExptimeOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal UpdateExptimeOptions"))
	}
	// get file from id
	f, err := x.getFileByID(ctx, []byte(opt.FileID))
	if err != nil {
		return code.Error(err)
	}

	if f.ExpireTime+blockchain.FileRetainPeriod.Nanoseconds() <= opt.CurrentTime {
		return code.Error(errorx.New(errorx.ErrCodeParam, "file already expired over 7 days"))
	}
	if opt.NewExpireTime <= f.ExpireTime || opt.NewExpireTime <= opt.CurrentTime {
		return code.Error(errorx.New(errorx.ErrCodeParam, "invalid new expire time"))
	}

	// verify sig
	m := fmt.Sprintf("%s,%d,%d", opt.FileID, opt.NewExpireTime, opt.CurrentTime)
	err = x.checkSign(opt.Signature, f.Owner, []byte(m))
	if err != nil {
		return code.Error(err)
	}

	// marshal file
	f.ExpireTime = opt.NewExpireTime
	nf, err := json.Marshal(f)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal File"))
	}
	// set id-file on chain
	if err := ctx.PutObject([]byte(f.ID), nf); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to set id-file on chain"))
	}
	return code.OK(nf)
}

// SliceMigrateRecord is used by node to slice migration record
func (x *Xdata) SliceMigrateRecord(ctx code.Context) code.Response {
	// get id
	nodeID, ok := ctx.Args()["nodeID"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}
	// get fileID
	fid, ok := ctx.Args()["fileID"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}
	// get sliceID
	sid, ok := ctx.Args()["sliceID"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:id"))
	}
	ctime, err := x.getCtxTime(ctx, "currentTime")
	if err != nil {
		return code.Error(err)
	}
	// get file from id
	file, err := x.getFileByID(ctx, fid)
	if err != nil {
		return code.Error(err)
	}
	// get signature
	signature, ok := ctx.Args()["signature"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:signature"))
	}
	msg := fmt.Sprintf("%s%s%s%s", string(fid), string(sid), string(nodeID), fmt.Sprintf("%d", ctime))
	// verify sig
	err = x.checkSign(signature, file.Owner, []byte(msg))
	if err != nil {
		return code.Error(err)
	}

	hindex := packNodeSliceMigrateIndex(string(nodeID), ctime)
	fs := map[string]interface{}{
		"fileID":  string(fid),
		"sliceID": string(sid),
	}
	b, err := json.Marshal(fs)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal fid-sid"))
	}
	// put index-migrate on xchain
	if err := ctx.PutObject([]byte(hindex), b); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeWriteBlockchain, "failed to put index-mirgate on xchain"))
	}
	return code.OK([]byte("ok"))
}

// ListFiles lists files from xchain
func (x *Xdata) ListFiles(ctx code.Context) code.Response {
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.ListFileOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListFileOptions"))
	}
	// pack prefix
	prefix := packFileNameFilter(opt.Owner, opt.Namespace)

	// get iter by prefix
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	// iterate iter
	var fs []blockchain.File
	for iter.Next() {
		if opt.Limit > 0 && int64(len(fs)) >= opt.Limit {
			break
		}
		f, err := x.getFileByID(ctx, iter.Value())
		if err != nil {
			return code.Error(err)
		}
		if f.PublishTime < opt.TimeStart || (opt.TimeEnd > 0 && f.PublishTime > opt.TimeEnd) || f.ExpireTime <= opt.CurrentTime {
			continue
		}
		fs = append(fs, f)
	}

	s, err := json.Marshal(fs)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Files"))
	}
	return code.OK(s)
}

// ListExpiredFiles lists expired but valid files
func (x *Xdata) ListExpiredFiles(ctx code.Context) code.Response {
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.ListFileOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListFileOptions"))
	}
	// pack prefix
	prefix := packFileNameFilter(opt.Owner, opt.Namespace)

	// get iter by prefix
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	// iterate iter
	var fs []blockchain.File
	for iter.Next() {
		if opt.Limit > 0 && int64(len(fs)) >= opt.Limit {
			break
		}
		f, err := x.getFileByID(ctx, iter.Value())
		if err != nil {
			return code.Error(err)
		}
		if f.PublishTime < opt.TimeStart || (opt.TimeEnd > 0 && f.PublishTime > opt.TimeEnd) {
			continue
		}
		if f.ExpireTime < opt.CurrentTime-blockchain.FileRetainPeriod.Nanoseconds() || f.ExpireTime > opt.CurrentTime {
			continue
		}
		fs = append(fs, f)
	}

	s, err := json.Marshal(fs)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Files"))
	}
	return code.OK(s)
}

// ListFileNs lists file namespaces by owner
func (x *Xdata) ListFileNs(ctx code.Context) code.Response {
	// get opt
	s, ok := ctx.Args()["opt"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:opt"))
	}
	// unmarshal opt
	var opt blockchain.ListNsOptions
	if err := json.Unmarshal(s, &opt); err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListNsOptions"))
	}
	// pack prefix
	prefix := packFileNsListFilter(opt.Owner)

	// get iter by prefix
	iter := ctx.NewIterator(code.PrefixRange([]byte(prefix)))
	defer iter.Close()

	// iterate iter
	var nss []blockchain.Namespace
	for iter.Next() {
		if opt.Limit > 0 && int64(len(nss)) >= opt.Limit {
			break
		}
		var ns blockchain.Namespace
		if err := json.Unmarshal(iter.Value(), &ns); err != nil {
			return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal namespaces"))
		}
		if ns.CreateTime < opt.TimeStart || (opt.TimeEnd > 0 && ns.CreateTime > opt.TimeEnd) {
			continue
		}
		nss = append(nss, ns)
	}

	s, err := json.Marshal(nss)
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal namespace list"))
	}
	return code.OK(s)
}

func (x *Xdata) GetNsByName(ctx code.Context) code.Response {
	// get owner
	owner, ok := ctx.Args()["owner"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:owner"))
	}
	// get ns
	ns, ok := ctx.Args()["name"]
	if !ok {
		return code.Error(errorx.New(errorx.ErrCodeParam, "missing param:ns"))
	}
	// get file ns
	filensIndex := packFileNsIndex(owner, string(ns))
	nsr, err := ctx.GetObject([]byte(filensIndex))
	if err != nil {
		return code.Error(errorx.NewCode(err, errorx.ErrCodeNotFound, "file namespace not found"))
	}

	return code.OK(nsr)
}

func (x *Xdata) getCtxTime(ctx code.Context, timeName string) (int64, error) {
	pTime, cok := ctx.Args()[timeName]
	if !cok {
		return 0, errorx.New(errorx.ErrCodeParam, "missing param:%s", timeName)
	}
	ptime, err := strconv.ParseInt(string(pTime), 10, 64)
	if err != nil {
		return 0, errorx.NewCode(err, errorx.ErrCodeParam, "bad param:%d", ptime)
	}
	return ptime, nil
}

func (x *Xdata) getFileByID(ctx code.Context, fileID []byte) (f blockchain.File, err error) {
	s, err := ctx.GetObject(fileID)
	if err != nil {
		return f, errorx.NewCode(err, errorx.ErrCodeNotFound,
			"the file[%s] not found", string(fileID))
	}
	if err = json.Unmarshal(s, &f); err != nil {
		return f, errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File")
	}
	return f, nil
}

func (x *Xdata) checkSign(sign, owner, mes []byte) (err error) {
	// verify sig
	if len(sign) != ecdsa.SignatureLength {
		return errorx.New(errorx.ErrCodeParam, "bad param:signature")
	}
	var pubkey [ecdsa.PublicKeyLength]byte
	var sig [ecdsa.SignatureLength]byte
	copy(pubkey[:], owner)
	copy(sig[:], sign)
	if err := ecdsa.Verify(pubkey, hash.HashUsingSha256(mes), sig); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeBadSignature, "failed to verify signature")
	}
	return nil
}
