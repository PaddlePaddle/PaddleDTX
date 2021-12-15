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

package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// ListFiles lists files from blockchain
func (e *Engine) ListFiles(opt types.ListFileOptions) (
	[]blockchain.File, error) {
	if len(opt.Namespace) != 0 {
		if _, err := e.chain.GetNsByName(opt.Owner, opt.Namespace); err != nil {
			if errorx.Is(err, errorx.ErrCodeNotFound) {
				return nil, errorx.New(errorx.ErrCodeNotFound, "ns not found")
			} else {
				return nil, errorx.Wrap(err, "failed to get ns from blockchain")
			}
		}
	}
	bcopt := blockchain.ListFileOptions{
		Owner:       opt.Owner,
		Namespace:   opt.Namespace,
		TimeStart:   opt.TimeStart,
		TimeEnd:     opt.TimeEnd,
		Limit:       opt.Limit,
		CurrentTime: opt.CurrentTime,
	}
	files, err := e.chain.ListFiles(&bcopt)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to read blockchain")
	}
	return files, nil
}

// ListExpiredFiles list expired but still valid files
func (e *Engine) ListExpiredFiles(opt types.ListFileOptions) (
	[]blockchain.File, error) {
	if len(opt.Namespace) != 0 {
		_, err := e.chain.GetNsByName(opt.Owner, opt.Namespace)
		if err != nil {
			if errorx.Is(err, errorx.ErrCodeNotFound) {
				return nil, errorx.New(errorx.ErrCodeNotFound, "ns not found")
			} else {
				return nil, errorx.Wrap(err, "failed to get ns from blockchain")
			}
		}
	}
	bcopt := blockchain.ListFileOptions{
		Owner:       opt.Owner,
		Namespace:   opt.Namespace,
		TimeStart:   opt.TimeStart,
		TimeEnd:     opt.TimeEnd,
		Limit:       opt.Limit,
		CurrentTime: opt.CurrentTime,
	}
	files, err := e.chain.ListExpiredFiles(&bcopt)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to read blockchain")
	}
	return files, nil
}

// GetFileByID gets file by id from blockchain
func (e *Engine) GetFileByID(ctx context.Context, id string) (hfile blockchain.FileH, err error) {
	var file blockchain.File
	file, err = e.chain.GetFileByID(id)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return hfile, err
		} else {
			return hfile, errorx.Wrap(err, "failed to read blockchain")
		}
	}

	// get replica from chain
	bns, err := e.chain.GetNsByName(file.Owner, file.Namespace)
	if err != nil {
		return hfile, errorx.Wrap(err, "failed to get ns from blockchain")
	}
	health, err := common.GetFileHealth(ctx, e.chain, file, bns.Replica)
	if err != nil {
		return hfile, err
	}
	hfile = blockchain.FileH{
		File:   file,
		Health: health,
	}
	return hfile, nil
}

// GetFileByName gets file by name from blockchain
func (e *Engine) GetFileByName(ctx context.Context, owner []byte, ns, name string) (
	hfile blockchain.FileH, err error) {
	var file blockchain.File
	file, err = e.chain.GetFileByName(owner, ns, name)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return hfile, err
		} else {
			return hfile, errorx.Wrap(err, "failed to read blockchain")
		}
	}

	// get replica from chain
	bns, err := e.chain.GetNsByName(file.Owner, file.Namespace)
	if err != nil {
		return hfile, errorx.Wrap(err, "failed to get ns from blockchain")
	}
	health, err := common.GetFileHealth(ctx, e.chain, file, bns.Replica)
	if err != nil {
		return hfile, err
	}
	hfile = blockchain.FileH{
		File:   file,
		Health: health,
	}
	return hfile, nil
}

// UpdateFileExpireTime updates file expire time
func (e *Engine) UpdateFileExpireTime(ctx context.Context, opt types.UpdateFileEtimeOptions) (err error) {
	if err := e.verifyUserID(opt.Owner); err != nil {
		return err
	}
	sig, err := ecdsa.DecodeSignatureFromString(opt.Token)
	if err != nil {
		return errorx.Wrap(err, "failed to decode signature")
	}

	if opt.CurrentTime+5*time.Second.Nanoseconds() < time.Now().UnixNano() {
		return errorx.New(errorx.ErrCodeExpired, "request expired")
	}

	file, err := e.chain.GetFileByID(opt.FileID)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return err
		} else if !errorx.Is(err, errorx.ErrCodeExpired) {
			return errorx.Wrap(err, "failed to read blockchain")
		}
	}

	if opt.ExpireTime <= opt.CurrentTime || opt.ExpireTime <= file.ExpireTime {
		return errorx.New(errorx.ErrCodeParam, "invalid param expireTime")
	}

	uopt := &blockchain.UpdateExptimeOptions{
		FileId:        opt.FileID,
		NewExpireTime: opt.ExpireTime,
		CurrentTime:   opt.CurrentTime,
		Signature:     sig[:],
	}
	newFile, err := e.chain.UpdateFileExpireTime(uopt)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return err
		} else {
			return errorx.Wrap(err, "failed to update file on blockchain")
		}
	} else {
		logger.WithFields(logrus.Fields{
			"file_id":     opt.FileID,
			"expire_time": time.Unix(0, opt.ExpireTime).Format("2006-01-02 15:04:05"),
		}).Info("updated file expire time")
	}

	// if challenge type is "merkle", add new challenge material
	startTime := file.ExpireTime
	if startTime == 0 {
		startTime = opt.CurrentTime
	}
	challengeAlgorithm, _ := e.challenger.GetChallengeConf()
	if challengeAlgorithm == types.MerkleChallengeAlgorithm {
		ti := e.monitor.challengingMonitor.RequestInterval.Nanoseconds()
		if err := common.AddFileNewMerkleChallenge(ctx, e.challenger, e.chain, e.copier, e.encryptor, newFile, startTime, ti, logger); err != nil {
			return errorx.Wrap(err, "failed to add merkle challenges")
		}
	}
	return nil
}

// AddFileNs adds file namespace
func (e *Engine) AddFileNs(opt types.AddNsOptions) (err error) {
	if err := e.verifyUserID(opt.Owner); err != nil {
		return err
	}
	sig, err := ecdsa.DecodeSignatureFromString(opt.Token)
	if err != nil {
		return errorx.Wrap(err, "failed to decode signature")
	}
	pubkey := ecdsa.PublicKeyFromPrivateKey(e.monitor.challengingMonitor.PrivateKey)

	ns := blockchain.Namespace{
		Owner:        pubkey[:],
		Name:         opt.Namespace,
		Description:  opt.Description,
		CreateTime:   opt.CreateTime,
		UpdateTime:   opt.CreateTime,
		Replica:      opt.Replica,
		FileTotalNum: 0,
	}
	ans := &blockchain.AddNsOptions{
		Namespace: ns,
		Signature: sig[:],
	}

	if err := e.chain.AddFileNs(ans); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return err
		} else {
			return errorx.Wrap(err, "failed to add file ns on blockchain")
		}
	}
	return nil
}

// UpdateNsReplica updates file namespace replica
func (e *Engine) UpdateNsReplica(ctx context.Context, opt types.UpdateNsOptions) error {
	localPrv := e.monitor.challengingMonitor.PrivateKey
	localPub := ecdsa.PublicKeyFromPrivateKey(localPrv)
	m := fmt.Sprintf("%s,%d,%d", opt.Namespace, opt.Replica, opt.CurrentTime)
	if err := verifyUserToken(localPub.String(), opt.Token, hash.HashUsingSha256([]byte(m))); err != nil {
		return err
	}
	sig, err := ecdsa.DecodeSignatureFromString(opt.Token)
	if err != nil {
		return errorx.Wrap(err, "failed to decode signature")
	}

	// get replica from chain
	ns, err := e.chain.GetNsByName(localPub[:], opt.Namespace)
	if err != nil {
		return errorx.Wrap(err, "failed to get ns from blockchain")
	}
	if ns.Replica >= opt.Replica {
		return errorx.New(errorx.ErrCodeParam, "bad param: replica")
	}
	// get healthy node to expand slice
	healthNodes, err := common.GetHealthNodes(e.chain)
	if err != nil {
		return errorx.Wrap(err, "failed to get nodes from blockchain")
	}
	// new replica must less than nodes
	if opt.Replica > len(healthNodes) {
		return errorx.New(errorx.ErrCodeInternal, "no optional healthy node to expand replica")
	}

	// query file by ns
	bcopt := blockchain.ListFileOptions{
		Owner:       ns.Owner,
		Namespace:   ns.Name,
		TimeStart:   ns.CreateTime,
		TimeEnd:     time.Now().UnixNano(),
		CurrentTime: time.Now().UnixNano(),
	}
	files, err := e.chain.ListFiles(&bcopt)
	if err != nil {
		return errorx.Wrap(err, "failed to get file list on  blockchain")
	}

	nsMaxFilesStructSize := 0
	for _, v := range files {
		nsMaxFilesStructSize += calculateFileMaxStructSize(len(v.Slices)/ns.Replica, opt.Replica)
	}
	if nsMaxFilesStructSize >= blockchain.ContractMessageMaxSize {
		return errorx.Wrap(err, "files total struct size of ns more than maximum, expand slices failed")
	}

	sopt := &blockchain.UpdateNsReplicaOptions{
		Owner:       localPub[:],
		Name:        opt.Namespace,
		Replica:     opt.Replica,
		CurrentTime: opt.CurrentTime,
		Signature:   sig[:],
	}
	err = e.chain.UpdateNsReplica(sopt)
	if err != nil {
		return errorx.Wrap(err, "failed to update file ns replica on blockchain")
	}
	if err := e.nsReplicaExpansion(ctx, files, healthNodes, opt.Replica, localPrv); err != nil {
		return errorx.Wrap(err, "expand failed, files total struct size of ns more than maximum")
	}
	return nil
}

// ListFileNs lists file namespaces by owner
func (e *Engine) ListFileNs(opt types.ListNsOptions) (nss []blockchain.Namespace, err error) {
	nsopt := blockchain.ListNsOptions{
		Owner:       opt.Owner,
		TimeStart:   opt.TimeStart,
		TimeEnd:     opt.TimeEnd,
		Limit:       opt.Limit,
		CurrentTime: time.Now().UnixNano(),
	}
	nss, err = e.chain.ListFileNs(&nsopt)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to read blockchain")
	}
	return nss, nil
}

// GetNsByName gets namespace by name from blockchain
func (e *Engine) GetNsByName(ctx context.Context, owner []byte, name string) (nsh blockchain.NamespaceH, err error) {
	// get replica from chain
	ns, err := e.chain.GetNsByName(owner, name)
	if err != nil {
		return nsh, errorx.Wrap(err, "failed to get ns from blockchain")
	}
	nsh, err = common.GetNsFilesHealth(ctx, ns, e.chain)
	if err != nil {
		return nsh, errorx.Wrap(err, "failed to get ns health")
	}

	return nsh, nil
}

// GetFileSysHealth get file owner system health status
func (e *Engine) GetFileSysHealth(ctx context.Context, owner []byte) (fh blockchain.FileSysHealth, err error) {
	nsopt := blockchain.ListNsOptions{
		Owner:     owner,
		TimeStart: 0,
		TimeEnd:   time.Now().UnixNano(),
	}
	nss, err := e.chain.ListFileNs(&nsopt)
	if err != nil {
		return fh, errorx.Wrap(err, "failed to read blockchain")
	}
	// get nodes list
	nodes, err := e.chain.ListNodes()
	if err != nil {
		return fh, errorx.Wrap(err, "failed to read blockchain")
	}
	fh, err = common.GetFileSysHealth(ctx, nss, nodes, e.chain)
	if err != nil {
		return fh, errorx.Wrap(err, "failed to read blockchain")
	}
	return fh, nil
}

// GetChallengeById gets a challenge by ID
func (e *Engine) GetChallengeById(id string) (challenge blockchain.Challenge, err error) {
	challenge, err = e.chain.GetChallengeById(id)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return challenge, errorx.New(errorx.ErrCodeNotFound, "challenge not found")
		} else {
			return challenge, errorx.Wrap(err, "failed to read blockchain")
		}
	}
	return challenge, nil
}

// GetChallenges lists all challenges with given status from blockchain
func (e *Engine) GetChallenges(opt blockchain.ListChallengeOptions) (challenges []blockchain.Challenge, err error) {
	_, err = e.chain.GetNode(opt.TargetNode)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return challenges, errorx.New(errorx.ErrCodeNotFound, "node not found")
		} else {
			return challenges, errorx.Wrap(err, "failed to read blockchain")
		}
	}
	challenges, err = e.chain.ListChallengeRequests(&opt)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return challenges, errorx.New(errorx.ErrCodeNotFound, "challenge not found")
		} else {
			return challenges, errorx.Wrap(err, "failed to read blockchain")
		}
	}
	return challenges, nil
}

func (e *Engine) nsReplicaExpansion(ctx context.Context, files []blockchain.File, healthNodes blockchain.NodeHs,
	replica int, pri ecdsa.PrivateKey) error {

	nodesMap := common.ToNodeHsMap(healthNodes)
	interval := e.monitor.challengingMonitor.RequestInterval.Nanoseconds()

	wg := sync.WaitGroup{}
	wg.Add(len(files))
	for _, f := range files {
		go func(f blockchain.File) {
			defer wg.Done()
			err := common.ExpandFileSlices(ctx, pri, e.copier, e.encryptor, e.chain, e.challenger,
				f, nodesMap, replica, healthNodes, interval, logger)
			if err != nil {
				logger.WithField("file_id", f.ID).WithError(err).Error("failed to expand file")
			} else {
				logger.WithField("file_id", f.ID).Info("success file expanded")
			}
		}(f)
	}
	wg.Wait()
	return nil
}
