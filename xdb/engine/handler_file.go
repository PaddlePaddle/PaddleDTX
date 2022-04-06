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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecies"
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
	owner, err := e.getPubKey(opt.Owner)
	if err != nil {
		return nil, err
	}
	// check file's namespace is exist
	if _, err := e.chain.GetNsByName(owner[:], opt.Namespace); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "ns not found")
		}
		return nil, errorx.Wrap(err, "failed to get ns from blockchain")
	}
	bcopt := blockchain.ListFileOptions{
		Owner:       owner,
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
	owner, err := e.getPubKey(opt.Owner)
	if err != nil {
		return nil, err
	}
	if _, err := e.chain.GetNsByName(owner[:], opt.Namespace); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "ns not found")
		}
		return nil, errorx.Wrap(err, "failed to get ns from blockchain")
	}

	bcopt := blockchain.ListFileOptions{
		Owner:       owner,
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
		}
		return hfile, errorx.Wrap(err, "failed to read blockchain")
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
func (e *Engine) GetFileByName(ctx context.Context, pubkey, ns, name string) (
	hfile blockchain.FileH, err error) {
	var file blockchain.File
	owner, err := e.getPubKey(pubkey)
	if err != nil {
		return hfile, err
	}
	file, err = e.chain.GetFileByName(owner, ns, name)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return hfile, err
		}
		return hfile, errorx.Wrap(err, "failed to read blockchain")
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
	if err := e.verifyUserID(opt.User); err != nil {
		return err
	}
	m := fmt.Sprintf("%s,%d,%d", opt.FileID, opt.ExpireTime, opt.CurrentTime)
	if err := verifyUserToken(opt.User, opt.Token, hash.HashUsingSha256([]byte(m))); err != nil {
		return err
	}

	localPrv := e.monitor.challengingMonitor.PrivateKey
	sig, err := ecdsa.Sign(localPrv, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign")
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
		FileID:        opt.FileID,
		NewExpireTime: opt.ExpireTime,
		CurrentTime:   opt.CurrentTime,
		Signature:     sig[:],
	}
	newFile, err := e.chain.UpdateFileExpireTime(uopt)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return err
		}
		return errorx.Wrap(err, "failed to update file on blockchain")
	}
	logger.WithFields(logrus.Fields{
		"file_id":     opt.FileID,
		"expire_time": time.Unix(0, opt.ExpireTime).Format("2006-01-02 15:04:05"),
	}).Info("updated file expire time")

	// add new challenge material
	startTime := file.ExpireTime
	if startTime <= opt.CurrentTime {
		startTime = opt.CurrentTime
	}
	challengeAlgorithm, pairingConf := e.challenger.GetChallengeConf()
	interval := e.monitor.challengingMonitor.RequestInterval.Nanoseconds()
	if challengeAlgorithm == types.MerkleChallengeAlgorithm {
		if err := common.AddFileNewMerkleChallenge(ctx, e.challenger, e.chain, e.copier, e.encryptor, newFile, startTime, interval, logger); err != nil {
			return errorx.Wrap(err, "failed to add merkle challenge material")
		}
	}
	if challengeAlgorithm == types.PairingChallengeAlgorithm {
		if err := common.AddFilePairingChallenges(ctx, pairingConf, e.chain, e.copier, newFile, opt.User, startTime, interval, logger); err != nil {
			return errorx.Wrap(err, "failed to add pairing based challenge material")
		}
	}
	return nil
}

// AddFileNs adds file namespace, opt.User is dataOwner node client's public key
func (e *Engine) AddFileNs(opt types.AddNsOptions) (err error) {
	if err := e.verifyUserID(opt.User); err != nil {
		return err
	}

	m := fmt.Sprintf("%s,%s,%d,%d", opt.Namespace, opt.Description, opt.CreateTime, opt.Replica)
	if err := verifyUserToken(opt.User, opt.Token, hash.HashUsingSha256([]byte(m))); err != nil {
		return err
	}

	// sign with the private key of the dataOwner node
	pubkey := ecdsa.PublicKeyFromPrivateKey(e.monitor.challengingMonitor.PrivateKey)
	namespace := blockchain.Namespace{
		Name:         opt.Namespace,
		Description:  opt.Description,
		Owner:        pubkey[:],
		CreateTime:   opt.CreateTime,
		UpdateTime:   opt.CreateTime,
		Replica:      opt.Replica,
		FileTotalNum: 0,
	}

	s, err := json.Marshal(namespace)
	if err != nil {
		return errorx.Wrap(err, "failed to marshal namespace")
	}
	sig, err := ecdsa.Sign(e.monitor.challengingMonitor.PrivateKey, hash.HashUsingSha256(s))
	if err != nil {
		return errorx.Wrap(err, "failed to sign file expire time")
	}
	ans := &blockchain.AddNsOptions{
		Namespace: namespace,
		Signature: sig[:],
	}

	if err := e.chain.AddFileNs(ans); err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return err
		}
		return errorx.Wrap(err, "failed to add file ns on blockchain")
	}
	return nil
}

// UpdateNsReplica updates file namespace replica
func (e *Engine) UpdateNsReplica(ctx context.Context, opt types.UpdateNsOptions) error {
	if err := e.verifyUserID(opt.User); err != nil {
		return err
	}
	m := fmt.Sprintf("%s,%d,%d", opt.Namespace, opt.Replica, opt.CurrentTime)
	if err := verifyUserToken(opt.User, opt.Token, hash.HashUsingSha256([]byte(m))); err != nil {
		return err
	}

	// sign using local key
	localPrv := e.monitor.challengingMonitor.PrivateKey
	localPub := ecdsa.PublicKeyFromPrivateKey(localPrv)
	sig, err := ecdsa.Sign(localPrv, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign")
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

	// update ns replica on blockchain
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

	// expand slices and challenge material
	if err := e.nsReplicaExpansion(ctx, files, healthNodes, opt.Replica, localPrv); err != nil {
		return errorx.Wrap(err, "expand file slices failed")
	}
	return nil
}

// ListFileNs lists file namespaces by owner
func (e *Engine) ListFileNs(opt types.ListNsOptions) (nss []blockchain.Namespace, err error) {
	owner, err := e.getPubKey(opt.Owner)
	if err != nil {
		return nss, err
	}
	nsopt := blockchain.ListNsOptions{
		Owner:       owner,
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
func (e *Engine) GetNsByName(ctx context.Context, pubkey, name string) (nsh blockchain.NamespaceH, err error) {
	owner, err := e.getPubKey(pubkey)
	if err != nil {
		return nsh, err
	}
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
func (e *Engine) GetFileSysHealth(ctx context.Context, pubkey string) (fh blockchain.FileSysHealth, err error) {
	owner, err := e.getPubKey(pubkey)
	if err != nil {
		return fh, err
	}
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

// ListFileAuths query the list of authorization applications
func (e *Engine) ListFileAuths(opt types.ListFileAuthOptions) (fileAuths blockchain.FileAuthApplications, err error) {
	authorizer, err := e.getPubKey(opt.Authorizer)
	if err != nil {
		return fileAuths, err
	}
	// if FileID not empty, judge whether the file on chain exists
	if opt.FileID != "" {
		_, err := e.chain.GetFileByID(opt.FileID)
		if err != nil {
			if errorx.Is(err, errorx.ErrCodeNotFound) {
				return fileAuths, err
			}
			return fileAuths, errorx.Wrap(err, "failed to read blockchain")
		}
	}
	bcopt := blockchain.ListFileAuthOptions{
		Authorizer: authorizer,
		FileID:     opt.FileID,
		TimeStart:  opt.TimeStart,
		TimeEnd:    opt.TimeEnd,
		Limit:      opt.Limit,
		Status:     opt.Status,
	}
	// if opt.Applier not empty, check applier's public key
	if opt.Applier != "" {
		applier, err := ecdsa.DecodePublicKeyFromString(opt.Applier)
		if err != nil {
			return fileAuths, err
		}
		bcopt.Applier = applier[:]
	}
	fileAuths, err = e.chain.ListFileAuthApplications(&bcopt)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to read blockchain")
	}
	return fileAuths, nil
}

// ConfirmAuth the dataOwner node confirms or rejects the applier's file authorization application
func (e *Engine) ConfirmAuth(opt types.ConfirmAuthOptions) error {
	// check whether opt.User is equal to the dataOwner node public key or authorized client's public key
	if err := e.verifyUserID(opt.User); err != nil {
		return err
	}
	m := fmt.Sprintf("%s,%d,%s", opt.AuthID, opt.ExpireTime, opt.RejectReason)
	if err := verifyUserToken(opt.User, opt.Token, hash.HashUsingSha256([]byte(m))); err != nil {
		return errorx.Wrap(err, "failed to verify user token")
	}

	fileAuth, err := e.GetAuthByID(opt.AuthID)
	if err != nil {
		return errorx.Wrap(err, "failed to get file authorization application by authID")
	}

	copt := &blockchain.ConfirmFileAuthOptions{
		ID:           opt.AuthID,
		RejectReason: opt.RejectReason,
		CurrentTime:  time.Now().UnixNano(),
		ExpireTime:   opt.ExpireTime,
	}
	sigMes := fmt.Sprintf("%s,%d,", copt.ID, copt.CurrentTime)
	if opt.Status {
		// Obtain an encryption key once and twice
		authKey, err := e.getAuthKey(fileAuth.FileID, fileAuth.Applier, opt.ExpireTime)
		if err != nil {
			return errorx.Wrap(err, "failed to get file authorization encryption key")
		}
		copt.AuthKey = authKey
		sigMes += fmt.Sprintf("%x,%d", authKey, copt.ExpireTime)
	} else {
		sigMes += copt.RejectReason
	}
	// Sign confirm info
	sig, err := ecdsa.Sign(e.monitor.challengingMonitor.PrivateKey, hash.HashUsingSha256([]byte(sigMes)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign file authorization application")
	}
	copt.Signature = sig[:]

	if opt.Status {
		err = e.chain.ConfirmFileAuthApplication(copt)
	} else {
		err = e.chain.RejectFileAuthApplication(copt)
	}
	if err != nil {
		return errorx.Wrap(err, "failed to confirm the applier's authorization on blockchain")
	}
	return nil
}

// GetAuthKey get the authorization key for file decryption
func (e *Engine) getAuthKey(fileID string, applier []byte, expireTime int64) ([]byte, error) {
	// Query file details
	file, err := e.chain.GetFileByID(fileID)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to get file from blockchain")
	}
	// The authorization expiration time must be less than the file expiration time
	if file.ExpireTime <= expireTime {
		return nil, errorx.New(errorx.ErrCodeParam, "file expireTime less than authorization expireTime")
	}
	authKey := make(map[string]interface{})
	// Get the first-level derived key
	firstEncSecret := e.encryptor.GetKey(fileID, "", []byte{})
	authKey["firstEncSecret"] = firstEncSecret

	// Get the second-level derived key
	secondEncSecret := make(map[string]map[string]interface{})
	slicesPool := makeSlicesPool4Read(file.Slices)
	for sliceID, targetPools := range slicesPool {
		secondEncSecret[sliceID] = make(map[string]interface{})
		for _, slice := range targetPools {
			secondEncSecret[sliceID][string(slice.NodeID)] = e.encryptor.GetKey(fileID, sliceID, slice.NodeID)
		}
	}
	authKey["secondEncSecret"] = secondEncSecret

	authKeyBytes, err := json.Marshal(authKey)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "fail to marshal authKey")
	}
	// parse ecdsa.PublicKey to EC public key
	var pubkey [ecdsa.PublicKeyLength]byte
	copy(pubkey[:], applier)
	applierPublicKey, err := ecdsa.ParsePublicKey(pubkey)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "fail to parse applier's publicKey")
	}
	// applier's EC public key encrypt the authKey
	cypherText, err := ecies.Encrypt(&applierPublicKey, authKeyBytes)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "fail to encrypt the authKey")
	}
	return cypherText, nil
}

// GetAuthByID get file authorization application detail by authID
func (e *Engine) GetAuthByID(id string) (fileAuth blockchain.FileAuthApplication, err error) {
	fileAuth, err = e.chain.GetAuthApplicationByID(id)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return fileAuth, errorx.New(errorx.ErrCodeNotFound, "authorization application not found")
		}
		return fileAuth, errorx.Wrap(err, "failed to read blockchain")
	}
	return fileAuth, nil
}

// GetChallengeByID gets a challenge by id
func (e *Engine) GetChallengeByID(id string) (challenge blockchain.Challenge, err error) {
	challenge, err = e.chain.GetChallengeByID(id)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return challenge, errorx.New(errorx.ErrCodeNotFound, "challenge not found")
		}
		return challenge, errorx.Wrap(err, "failed to read blockchain")
	}
	return challenge, nil
}

// GetChallenges lists all challenges with given status from blockchain
func (e *Engine) GetChallenges(opt blockchain.ListChallengeOptions) (challenges []blockchain.Challenge, err error) {
	_, err = e.chain.GetNode(opt.TargetNode)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return challenges, errorx.New(errorx.ErrCodeNotFound, "node not found")
		}
		return challenges, errorx.Wrap(err, "failed to read blockchain")
	}
	challenges, err = e.chain.ListChallengeRequests(&opt)
	if err != nil {
		if errorx.Is(err, errorx.ErrCodeNotFound) {
			return challenges, errorx.New(errorx.ErrCodeNotFound, "challenge not found")
		}
		return challenges, errorx.Wrap(err, "failed to read blockchain")
	}
	return challenges, nil
}

// nsReplicaExpansion expand file replica under the namespace
// After each slice under the file is restored,
// the replica is copied and pushed to the new storage node
// challenges will be generated for new storage node's slices
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
				logger.WithField("file_id", f.ID).Info("successfully expanded file")
			}
		}(f)
	}
	wg.Wait()
	return nil
}
