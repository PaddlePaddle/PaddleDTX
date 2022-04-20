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
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// PublishFile publishes file onto fabric
func (x *Xdata) PublishFile(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting PublishFileOptions")
	}

	// unmarshal opt
	var opt blockchain.PublishFileOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal PublishFileOptions").Error())
	}

	// get file
	f := opt.File
	// marshal file
	s, err := json.Marshal(f)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal File").Error())
	}

	// verify sig
	err = x.checkSign(opt.Signature, f.Owner, s)
	if err != nil {
		return shim.Error(err.Error())
	}

	// judge if slices is empty
	if len(f.Slices) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeParam,
			"slices is empty when publishing file").Error())
	}
	// judge if id exists
	if resp := x.getValue(stub, []string{opt.File.ID}); len(resp.Payload) != 0 {
		return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated fileID").Error())
	}

	// judge if fileNsIndex exists
	fileNsIndex := packFileNsIndex(f.Owner, f.Namespace)
	resp := x.getValue(stub, []string{fileNsIndex})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound,
			"file namespace not found: %s", resp.Message).Error())
	}
	var ns blockchain.Namespace
	if err = json.Unmarshal(resp.Payload, &ns); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal namespace").Error())
	}

	// if there's a expired file with the same name in user's storage, overwrite it with the new one
	filenameIndex := packFileNameIndex(f.Owner, f.Namespace, f.Name)

	if resp := x.getValue(stub, []string{filenameIndex}); len(resp.Payload) != 0 {
		fc, err := x.getFileByID(stub, string(resp.Payload))
		if err != nil {
			return shim.Error(err.Error())
		}
		if fc.ExpireTime+blockchain.FileRetainPeriod.Nanoseconds() > opt.File.PublishTime {
			return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated file name").Error())
		}
	}

	// set id-file on chain
	if resp := x.setValue(stub, []string{f.ID, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set id-file on chain: %s", resp.Message).Error())
	}
	// set filenameIndex-id on chain
	if resp := x.setValue(stub, []string{filenameIndex, f.ID}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index-id on chain: %s", resp.Message).Error())
	}
	// set filenameListIndex-id on chain
	filenameListIndex := packFileNameListIndex(f.Owner, f.Namespace, f.Name, f.PublishTime)
	if resp := x.setValue(stub, []string{filenameListIndex, f.ID}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set listIndex-id on chain: %s", resp.Message).Error())
	}

	// update file num of fileNsIndex
	ns.FileTotalNum += 1
	ns.UpdateTime = f.PublishTime
	nsf, err := json.Marshal(ns)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal File namespace").Error())
	}
	if resp := x.setValue(stub, []string{fileNsIndex, string(nsf)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to update index-ns on chain: %s", resp.Message).Error())
	}
	nsListIndex := packFileNsListIndex(ns.Owner, ns.Name, ns.CreateTime)
	if resp := x.setValue(stub, []string{nsListIndex, string(nsf)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to update listIndex-ns on chain: %s", resp.Message).Error())
	}

	// set node-sliceID-expireTime on chain
	nodeSlice := make(map[string][]string)
	for _, slice := range f.Slices {
		nodeSlice[string(slice.NodeID)] = append(nodeSlice[string(slice.NodeID)], slice.ID)
	}
	for nodeID, sliceL := range nodeSlice {
		prefixNodeFileSlice := packNodeSliceIndex(nodeID, f)
		if resp := x.setValue(stub, []string{prefixNodeFileSlice, strings.Join(sliceL, ",")}); resp.Status == shim.ERROR {
			return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
				"failed to set index-id on chain: %s", resp.Message).Error())
		}
	}

	return shim.Success([]byte("Published"))
}

// AddFileNs adds file namespace
func (x *Xdata) AddFileNs(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting AddNsOptions")
	}

	// unmarshal opt
	var opt blockchain.AddNsOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal AddNsOptions").Error())
	}

	ns := opt.Namespace
	s, err := json.Marshal(ns)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal namespace").Error())
	}
	// verify sig
	err = x.checkSign(opt.Signature, ns.Owner, s)
	if err != nil {
		return shim.Error(err.Error())
	}

	// judge if fileNsIndex exists
	fileNsIndex := packFileNsIndex(ns.Owner, ns.Name)
	if resp := x.getValue(stub, []string{fileNsIndex}); len(resp.Payload) != 0 {
		return shim.Error(errorx.New(errorx.ErrCodeAlreadyExists, "duplicated file namespace").Error())
	}

	// put index-ns on chain
	if resp := x.setValue(stub, []string{fileNsIndex, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set index-ns on chain: %s", resp.Message).Error())
	}
	// put listIndex-ns on chain
	nsListIndex := packFileNsListIndex(ns.Owner, ns.Name, ns.CreateTime)
	if resp := x.setValue(stub, []string{nsListIndex, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set listIndex-ns on chain: %s", resp.Message).Error())
	}

	return shim.Success([]byte("OK"))
}

// UpdateNsReplica updates file namespace replica
func (x *Xdata) UpdateNsReplica(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting UpdateNsReplicaOptions")
	}

	// unmarshal opt
	var opt blockchain.UpdateNsReplicaOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal UpdateNsReplicaOptions").Error())
	}
	// verify sig
	m := fmt.Sprintf("%s,%d,%d", opt.Name, opt.Replica, opt.CurrentTime)
	if err := x.checkSign(opt.Signature, opt.Owner, []byte(m)); err != nil {
		return shim.Error(err.Error())
	}

	// get file ns
	fileNsIndex := packFileNsIndex(opt.Owner, opt.Name)
	resp := x.getValue(stub, []string{fileNsIndex})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound,
			"file namespace not found: %s", resp.Message).Error())
	}
	var n blockchain.Namespace
	if err := json.Unmarshal(resp.Payload, &n); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal namespace").Error())
	}
	if n.Replica >= opt.Replica {
		return shim.Error(errorx.New(errorx.ErrCodeParam, "bad param:replica").Error())
	}

	n.Replica = opt.Replica
	n.UpdateTime = opt.CurrentTime
	s, err := json.Marshal(n)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal Namespaces").Error())
	}
	// put index-ns on chain
	if resp := x.setValue(stub, []string{fileNsIndex, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to update nsindex-replica on chain: %s", resp.Message).Error())
	}
	// put listIndex-ns on chain
	nsListIndex := packFileNsListIndex(n.Owner, n.Name, n.CreateTime)
	if resp := x.setValue(stub, []string{nsListIndex, string(s)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to update nsListIndex-replica on chain: %s", resp.Message).Error())
	}
	return shim.Success([]byte("OK"))
}

// UpdateFilePublicSliceMeta is used to update file public slice metas
func (x *Xdata) UpdateFilePublicSliceMeta(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting UpdateFilePSMOptions")
	}

	// unmarshal opt
	var opt blockchain.UpdateFilePSMOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal UpdateNsReplicaOptions").Error())
	}
	// verify sig
	ns := opt.Slices
	s, err := json.Marshal(ns)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal slices").Error())
	}
	if err := x.checkSign(opt.Signature, opt.Owner, s); err != nil {
		return shim.Error(err.Error())
	}

	// get file from id
	resp := x.getValue(stub, []string{opt.FileID})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound,
			"file not found: %s", resp.Message).Error())
	}
	var f blockchain.File
	if err = json.Unmarshal(resp.Payload, &f); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File").Error())
	}
	if string(f.Owner) != string(opt.Owner) {
		return shim.Error(errorx.New(errorx.ErrCodeNotAuthorized, "bad param, file owner is wrong").Error())
	}

	// update slices
	f.Slices = opt.Slices
	nfs, err := json.Marshal(f)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal Namespaces").Error())
	}

	if resp := x.setValue(stub, []string{f.ID, string(nfs)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set id-file on chain: %s", resp.Message).Error())
	}
	return shim.Success([]byte("OK"))
}

// GetFileByName gets file by name from fabric
func (x *Xdata) GetFileByName(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 4 {
		return shim.Error("invalid arguments. expecting owner, ns, name and currentTime")
	}

	// get owner
	owner := []byte(args[0])
	// get ns
	ns := args[1]
	// get name
	name := args[2]
	// get current timestamp
	ctime, err := strconv.ParseInt(args[3], 10, 64)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parse timestamp").Error())
	}

	// judge if fileNsIndex exists
	fileNsIndex := packFileNsIndex(owner, ns)
	if resp := x.getValue(stub, []string{fileNsIndex}); len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeParam, "bad param, ns not found: %s", resp.Message).Error())
	}

	// pack fileNameIndex
	index := packFileNameIndex(owner, ns, name)
	//get id from index
	resp := x.getValue(stub, []string{index})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "fileID not found with name+ns: %s", resp.Message).Error())
	}
	// get file from id
	resp = x.getValue(stub, []string{string(resp.Payload)})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "file not found: %s", resp.Message).Error())
	}

	var f blockchain.File
	if err = json.Unmarshal(resp.Payload, &f); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal File").Error())
	}
	if f.ExpireTime < ctime {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "file already expire").Error())
	}
	return shim.Success(resp.Payload)
}

// GetFileByID gets file by id from fabric
func (x *Xdata) GetFileByID(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("invalid arguments. expecting id and currentTime")
	}

	// get id
	id := args[0]

	// get current timestamp
	ctime, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parse timestamp").Error())
	}

	//get file from id
	resp := x.getValue(stub, []string{id})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound, "file not found: %s", resp.Message).Error())
	}
	var f blockchain.File
	if err = json.Unmarshal(resp.Payload, &f); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File").Error())
	}

	if f.ExpireTime < ctime {
		return shim.Error(errorx.New(errorx.ErrCodeExpired, "file already expire").Error())
	}
	return shim.Success(resp.Payload)
}

// UpdateFileExpireTime updates file expiration time
func (x *Xdata) UpdateFileExpireTime(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting UpdateExptimeOptions")
	}

	// unmarshal opt
	var opt blockchain.UpdateExptimeOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal UpdateExptimeOptions").Error())
	}
	//get file from id
	f, err := x.getFileByID(stub, opt.FileID)
	if err != nil {
		return shim.Error(err.Error())
	}

	if f.ExpireTime+blockchain.FileRetainPeriod.Nanoseconds() <= opt.CurrentTime {
		return shim.Error(errorx.New(errorx.ErrCodeParam, "file already expired over 7 days").Error())
	}
	if opt.NewExpireTime <= f.ExpireTime || opt.NewExpireTime <= opt.CurrentTime {
		return shim.Error(errorx.New(errorx.ErrCodeParam, "invalid new expire time").Error())
	}

	// verify sig
	m := fmt.Sprintf("%s,%d,%d", opt.FileID, opt.NewExpireTime, opt.CurrentTime)
	err = x.checkSign(opt.Signature, f.Owner, []byte(m))
	if err != nil {
		return shim.Error(err.Error())
	}

	// marshal file
	f.ExpireTime = opt.NewExpireTime
	nf, err := json.Marshal(f)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal File").Error())
	}
	// set id-file on chain
	if resp := x.setValue(stub, []string{f.ID, string(nf)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to set id-file on chain: %s", resp.Message).Error())
	}
	return shim.Success(nf)
}

// SliceMigrateRecord is used by node to slice migration record
func (x *Xdata) SliceMigrateRecord(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 5 {
		return shim.Error("invalid arguments. expecting nodeID, fileID, sliceID, signature and currentTime")
	}

	// get id
	nodeID := []byte(args[0])
	// get fileID
	fid := args[1]
	// get sliceID
	sid := args[2]
	// get timestamp
	ctime, err := strconv.ParseInt(args[4], 10, 64)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parse timestamp").Error())
	}

	//get file from id
	f, err := x.getFileByID(stub, fid)
	if err != nil {
		return shim.Error(err.Error())
	}
	// get signature
	mes := fmt.Sprintf("%s%s%s%s", fid, sid, string(nodeID), fmt.Sprintf("%d", ctime))
	// verify sig
	err = x.checkSign([]byte(args[3]), f.Owner, []byte(mes))
	if err != nil {
		return shim.Error(err.Error())
	}

	hindex := packNodeSliceMigrateIndex(string(nodeID), ctime)
	fs := map[string]interface{}{
		"fileID":  fid,
		"sliceID": sid,
	}
	b, err := json.Marshal(fs)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal fid-sid").Error())
	}
	// put index-migrate on chain
	if resp := x.setValue(stub, []string{hindex, string(b)}); resp.Status == shim.ERROR {
		return shim.Error(errorx.New(errorx.ErrCodeWriteBlockchain,
			"failed to put index-migrate on chain: %s", resp.Message).Error())
	}
	return shim.Success([]byte("ok"))
}

// ListFiles lists files from fabric
func (x *Xdata) ListFiles(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting ListFileOptions")
	}

	// unmarshal opt
	var opt blockchain.ListFileOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListFileOptions").Error())
	}

	// pack prefix
	prefix, attr := packFileNameFilter(opt.Owner, opt.Namespace)
	// get iter by prefix
	iterator, err := stub.GetStateByPartialCompositeKey(prefix, attr)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	// iterate iter
	var fs []blockchain.File
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		if opt.Limit > 0 && int64(len(fs)) >= opt.Limit {
			break
		}
		f, err := x.getFileByID(stub, string(queryResponse.Value))
		if err != nil {
			return shim.Error(err.Error())
		}
		if f.PublishTime < opt.TimeStart || (opt.TimeEnd > 0 && f.PublishTime > opt.TimeEnd) || f.ExpireTime <= opt.CurrentTime {
			continue
		}
		fs = append(fs, f)
	}

	s, err := json.Marshal(fs)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Files").Error())
	}
	return shim.Success(s)
}

// ListExpiredFiles lists expired but valid files
func (x *Xdata) ListExpiredFiles(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting ListFileOptions")
	}

	// unmarshal opt
	var opt blockchain.ListFileOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListFileOptions").Error())
	}

	// pack prefix
	prefix, attr := packFileNameFilter(opt.Owner, opt.Namespace)
	// get iter by prefix
	iterator, err := stub.GetStateByPartialCompositeKey(prefix, attr)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	// iterate iter
	var fs []blockchain.File
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		if opt.Limit > 0 && int64(len(fs)) >= opt.Limit {
			break
		}
		f, err := x.getFileByID(stub, string(queryResponse.Value))
		if err != nil {
			return shim.Error(err.Error())
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
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal Files").Error())
	}
	return shim.Success(s)
}

// ListFileNs lists file namespaces by owner
func (x *Xdata) ListFileNs(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("invalid arguments. expecting ListNsOptions")
	}

	// unmarshal opt
	var opt blockchain.ListNsOptions
	if err := json.Unmarshal([]byte(args[0]), &opt); err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to unmarshal ListNsOptions").Error())
	}

	// pack prefix
	prefix, attr := packFileNsListFilter(opt.Owner)
	// get iter by prefix
	iterator, err := stub.GetStateByPartialCompositeKey(prefix, attr)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	// iterate iter
	var nss []blockchain.Namespace
	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		if opt.Limit > 0 && int64(len(nss)) >= opt.Limit {
			break
		}
		var ns blockchain.Namespace
		if err := json.Unmarshal(queryResponse.Value, &ns); err != nil {
			return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
				"failed to unmarshal namespaces").Error())
		}
		if ns.CreateTime < opt.TimeStart || (opt.TimeEnd > 0 && ns.CreateTime > opt.TimeEnd) {
			continue
		}
		nss = append(nss, ns)
	}

	s, err := json.Marshal(nss)
	if err != nil {
		return shim.Error(errorx.NewCode(err, errorx.ErrCodeInternal,
			"failed to marshal namespace list").Error())
	}
	return shim.Success(s)
}

// GetNsByName gets namespace by nsName from fabric
func (x *Xdata) GetNsByName(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("invalid arguments. expecting owner and name")
	}

	// get owner
	owner := []byte(args[0])
	// get ns
	ns := args[1]

	// judge if fileNsIndex exists
	fileNsIndex := packFileNsIndex(owner, ns)
	// get file ns
	resp := x.getValue(stub, []string{fileNsIndex})
	if len(resp.Payload) == 0 {
		return shim.Error(errorx.New(errorx.ErrCodeNotFound,
			"file namespace not found: %s", resp.Message).Error())
	}
	return shim.Success(resp.Payload)
}

func (x *Xdata) getFileByID(stub shim.ChaincodeStubInterface, fileID string) (f blockchain.File, err error) {
	resp := x.getValue(stub, []string{fileID})
	if len(resp.Payload) == 0 {
		return f, errorx.NewCode(err, errorx.ErrCodeNotFound, "file[%s] not found", fileID)
	}
	if err = json.Unmarshal(resp.Payload, &f); err != nil {
		return f, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal File")
	}
	return f, nil
}

// check signature
func (x *Xdata) checkSign(sign, owner, mes []byte) (err error) {
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
