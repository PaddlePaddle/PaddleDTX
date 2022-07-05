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

package http

import (
	"context"
	"io"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	httpkg "github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/http"
	util "github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/strings"
	servertypes "github.com/PaddlePaddle/PaddleDTX/xdb/server/types"
)

const (
	apiVersion = "v1"
)

type Client struct {
	baseAddr url.URL
}

// New new a client by server address
func New(addr string) (Client, error) {
	base, err := url.Parse(addr)
	if err != nil {
		return Client{}, errorx.NewCode(err, errorx.ErrCodeParam, "invalid addr")
	}
	base.Path = path.Join(base.Path, apiVersion)
	c := Client{
		baseAddr: *base,
	}
	return c, nil
}

// getRequestsUrl used to generate http api request url
func (c *Client) getRequestsUrl(path []string, params map[string]string) url.URL {
	url := c.baseAddr
	joinPath(&url, path...)
	q := url.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	url.RawQuery = q.Encode()
	return url
}

// Write upload a file
func (c *Client) Write(ctx context.Context, r io.Reader, opt WriteOptions) (
	servertypes.WriteResponse, error) {

	privkey, err := ecdsa.DecodePrivateKeyFromString(opt.PrivateKey)
	if err != nil {
		return servertypes.WriteResponse{}, err
	}
	// defines parameters for uploading files
	reqParams := map[string]string{
		"user":       ecdsa.PublicKeyFromPrivateKey(privkey).String(),
		"ns":         opt.Namespace,
		"name":       opt.FileName,
		"desc":       opt.Description,
		"ext":        opt.Extra,
		"expireTime": strconv.FormatInt(opt.ExpireTime, 10),
	}
	msg, err := util.GetSigMessage(reqParams)
	if err != nil {
		return servertypes.WriteResponse{}, errorx.Internal(err, "failed to get the message to sign")
	}

	sig, err := ecdsa.Sign(privkey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return servertypes.WriteResponse{}, errorx.Wrap(err, "failed to sign")
	}
	reqParams["token"] = sig.String()

	url := c.getRequestsUrl([]string{"file", "write"}, reqParams)
	var resp servertypes.WriteResponse
	if err := httpkg.PostResponse(ctx, url.String(), r, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// Read download a file
func (c *Client) Read(ctx context.Context, opt ReadOptions) (io.ReadCloser, error) {
	privkey, err := ecdsa.DecodePrivateKeyFromString(opt.PrivateKey)
	if err != nil {
		return nil, err
	}
	reqParams := map[string]string{
		"user":      ecdsa.PublicKeyFromPrivateKey(privkey).String(),
		"ns":        opt.Namespace,
		"name":      opt.FileName,
		"file_id":   opt.FileID,
		"timestamp": strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	msg, err := util.GetSigMessage(reqParams)
	if err != nil {
		return nil, errorx.Internal(err, "failed to get the message to sign")
	}
	sig, err := ecdsa.Sign(privkey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return nil, errorx.Wrap(err, "failed to sign")
	}
	reqParams["token"] = sig.String()

	url := c.getRequestsUrl([]string{"file", "read"}, reqParams)
	reader, err := httpkg.Get(ctx, url.String())
	if err != nil {
		return nil, err
	}

	return reader, nil
}

// ListNodes list all storage nodes in system
func (c *Client) ListNodes(ctx context.Context) (blockchain.Nodes, error) {
	var nodes blockchain.Nodes
	url := c.getRequestsUrl([]string{"node", "list"}, nil)
	if err := httpkg.GetResponse(ctx, url.String(), &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNode get storage node by node id
func (c *Client) GetNode(ctx context.Context, id string) (blockchain.Node, error) {
	var node blockchain.Node
	url := c.getRequestsUrl([]string{"node", "get"}, map[string]string{"id": id})
	if err := httpkg.GetResponse(ctx, url.String(), &node); err != nil {
		return node, err
	}
	return node, nil
}

// GetNodeHeartbeat get storage node heart beat number
func (c *Client) GetNodeHeartbeat(ctx context.Context, id string, ctime int64) (map[string]int, error) {
	var res map[string]int
	url := c.getRequestsUrl([]string{"node", "gethbnum"}, map[string]string{
		"id":    id,
		"ctime": strconv.FormatInt(ctime, 10),
	})
	if err := httpkg.GetResponse(ctx, url.String(), &res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetMigrateRecords get storage node migrate records
func (c *Client) GetMigrateRecords(ctx context.Context, id string, start, end int64, limit int64) ([]map[string]interface{}, error) {
	reqParams := map[string]string{
		"id":    id,
		"start": strconv.FormatInt(start, 10),
		"end":   strconv.FormatInt(end, 10),
		"limit": strconv.FormatInt(limit, 10),
	}
	var ms []map[string]interface{}
	url := c.getRequestsUrl([]string{"node", "getmrecord"}, reqParams)
	if err := httpkg.GetResponse(ctx, url.String(), &ms); err != nil {
		return nil, err
	}
	return ms, nil
}

// GetNodeHealth get storage node health status by node id
func (c *Client) GetNodeHealth(ctx context.Context, id string) (string, error) {
	var status string
	url := c.getRequestsUrl([]string{"node", "health"}, map[string]string{"id": id})
	if err := httpkg.GetResponse(ctx, url.String(), &status); err != nil {
		return "", err
	}
	return status, nil
}

// setNodeOnlineStatus set storage node status online/offline
func (c *Client) setNodeOnlineStatus(ctx context.Context, privateKey string, online bool) error {
	private, err := ecdsa.DecodePrivateKeyFromString(privateKey)
	if err != nil {
		return err
	}
	reqParams := map[string]string{
		"node":  ecdsa.PublicKeyFromPrivateKey(private).String(),
		"nonce": strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	msg, err := util.GetSigMessage(reqParams)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign")
	}

	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign")
	}
	reqParams["token"] = sig.String()

	var url url.URL
	if online {
		url = c.getRequestsUrl([]string{"node", "online"}, reqParams)
	} else {
		url = c.getRequestsUrl([]string{"node", "offline"}, reqParams)
	}
	if _, err := httpkg.Post(ctx, url.String(), nil); err != nil {
		return err
	}
	return nil
}

// NodeOffline set storage node status offline
func (c *Client) NodeOffline(ctx context.Context, privkey string) error {
	return c.setNodeOnlineStatus(ctx, privkey, false)
}

// NodeOnline set storage node status online
func (c *Client) NodeOnline(ctx context.Context, privkey string) error {
	return c.setNodeOnlineStatus(ctx, privkey, true)
}

// ListFiles list unexpired files
func (c *Client) ListFiles(ctx context.Context, opt ListFileOptions, isExpired bool) ([]blockchain.File, error) {
	reqParams := map[string]string{
		"owner": opt.Owner,
		"ns":    opt.Namespace,
		"start": strconv.FormatInt(opt.TimeStart, 10),
		"end":   strconv.FormatInt(opt.TimeEnd, 10),
		"limit": strconv.FormatInt(opt.Limit, 10),
	}
	var url url.URL
	if isExpired {
		url = c.getRequestsUrl([]string{"file", "listexp"}, reqParams)
	} else {
		url = c.getRequestsUrl([]string{"file", "list"}, reqParams)
	}
	var files []blockchain.File
	if err := httpkg.GetResponse(ctx, url.String(), &files); err != nil {
		return nil, err
	}
	return files, nil
}

// GetFileByID get file info by file id
func (c *Client) GetFileByID(ctx context.Context, id string) (blockchain.FileH, error) {
	var hfile blockchain.FileH
	url := c.getRequestsUrl([]string{"file", "getbyid"}, map[string]string{"id": id})
	if err := httpkg.GetResponse(ctx, url.String(), &hfile); err != nil {
		return hfile, err
	}
	return hfile, nil
}

// GetFileByName get file info by file name, owner and namespace
func (c *Client) GetFileByName(ctx context.Context, owner, ns, name string) (blockchain.FileH, error) {
	var hfile blockchain.FileH
	url := c.getRequestsUrl([]string{"file", "getbyname"}, map[string]string{"owner": owner, "ns": ns, "name": name})
	if err := httpkg.GetResponse(ctx, url.String(), &hfile); err != nil {
		return hfile, err
	}
	return hfile, nil
}

// UpdateExpTimeByID update file expire time by file id
func (c *Client) UpdateExpTimeByID(ctx context.Context, id, privateKey string, expireTime int64) error {
	private, err := ecdsa.DecodePrivateKeyFromString(privateKey)
	if err != nil {
		return err
	}
	reqParams := map[string]string{
		"id":         id,
		"user":       ecdsa.PublicKeyFromPrivateKey(private).String(),
		"ctime":      strconv.FormatInt(time.Now().UnixNano(), 10),
		"expireTime": strconv.FormatInt(expireTime, 10),
	}
	msg, err := util.GetSigMessage(reqParams)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign")
	}

	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign file expire time")
	}
	reqParams["token"] = sig.String()

	url := c.getRequestsUrl([]string{"file", "updatexptime"}, reqParams)
	if _, err := httpkg.Post(ctx, url.String(), nil); err != nil {
		return err
	}
	return nil
}

// AddFileNs add a file namespace
func (c *Client) AddFileNs(ctx context.Context, owner, priKey, ns, des string, replica int) error {
	private, err := ecdsa.DecodePrivateKeyFromString(priKey)
	if err != nil {
		return err
	}

	reqParams := map[string]string{
		"ns":      ns,
		"user":    ecdsa.PublicKeyFromPrivateKey(private).String(),
		"replica": strconv.Itoa(replica),
		"ctime":   strconv.FormatInt(time.Now().UnixNano(), 10),
		"desc":    des,
	}
	msg, err := util.GetSigMessage(reqParams)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign")
	}
	// sign ns info
	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign file namespace")
	}
	reqParams["token"] = sig.String()

	url := c.getRequestsUrl([]string{"file", "addns"}, reqParams)
	if _, err := httpkg.Post(ctx, url.String(), nil); err != nil {
		return err
	}
	return nil
}

// UpdateFileNsReplica update namespace replica
func (c *Client) UpdateFileNsReplica(ctx context.Context, priKey, ns string, replica int) error {
	private, err := ecdsa.DecodePrivateKeyFromString(priKey)
	if err != nil {
		return err
	}
	reqParams := map[string]string{
		"ns":      ns,
		"user":    ecdsa.PublicKeyFromPrivateKey(private).String(),
		"replica": strconv.Itoa(replica),
		"ctime":   strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	msg, err := util.GetSigMessage(reqParams)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign")
	}
	// sign the message
	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign update ns replica param")
	}
	reqParams["token"] = sig.String()

	url := c.getRequestsUrl([]string{"file", "ureplica"}, reqParams)
	if _, err := httpkg.Post(ctx, url.String(), nil); err != nil {
		return err
	}
	return nil
}

// ListFileNs list file namespaces
func (c *Client) ListFileNs(ctx context.Context, opt ListNsOptions) ([]blockchain.Namespace, error) {
	reqParams := map[string]string{
		"owner": opt.Owner,
		"start": strconv.FormatInt(opt.TimeStart, 10),
		"end":   strconv.FormatInt(opt.TimeEnd, 10),
		"limit": strconv.FormatInt(opt.Limit, 10),
	}

	var nss []blockchain.Namespace
	url := c.getRequestsUrl([]string{"file", "listns"}, reqParams)
	if err := httpkg.GetResponse(ctx, url.String(), &nss); err != nil {
		return nil, err
	}
	return nss, nil
}

// GetNsByName get namespace by name and owner
func (c *Client) GetNsByName(ctx context.Context, owner, ns string) (blockchain.NamespaceH, error) {
	var nsh blockchain.NamespaceH
	url := c.getRequestsUrl([]string{"file", "getns"}, map[string]string{"owner": owner, "name": ns})
	if err := httpkg.GetResponse(ctx, url.String(), &nsh); err != nil {
		return nsh, err
	}
	return nsh, nil
}

// GetFileSysHealth get system health status of an owner
func (c *Client) GetFileSysHealth(ctx context.Context, owner string) (blockchain.FileSysHealth, error) {
	var fh blockchain.FileSysHealth
	url := c.getRequestsUrl([]string{"file", "getsyshealth"}, map[string]string{"owner": owner})
	if err := httpkg.GetResponse(ctx, url.String(), &fh); err != nil {
		return fh, err
	}
	return fh, nil
}

// GetAuth get file authorization application detail by authID
func (c *Client) GetAuth(ctx context.Context, authID string) (blockchain.FileAuthApplication, error) {
	var fa blockchain.FileAuthApplication
	url := c.getRequestsUrl([]string{"file", "getauthbyid"}, map[string]string{"authID": authID})
	if err := httpkg.GetResponse(ctx, url.String(), &fa); err != nil {
		return fa, err
	}
	return fa, nil
}

// ConfirmOrRejectAuth confirm or reject applier's file authorization application by opt.Status
func (c *Client) ConfirmOrRejectAuth(ctx context.Context, opt ConfirmAuthOptions) error {
	private, err := ecdsa.DecodePrivateKeyFromString(opt.PrivateKey)
	if err != nil {
		return err
	}
	reqParams := map[string]string{
		"authID":       opt.AuthID,
		"user":         ecdsa.PublicKeyFromPrivateKey(private).String(),
		"status":       strconv.FormatBool(opt.Status),
		"expireTime":   strconv.FormatInt(opt.ExpireTime, 10),
		"rejectReason": opt.RejectReason,
	}
	msg, err := util.GetSigMessage(reqParams)
	if err != nil {
		return errorx.Internal(err, "failed to get the message to sign")
	}
	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign confirm file authorization application")
	}
	reqParams["token"] = sig.String()

	url := c.getRequestsUrl([]string{"file", "confirmauth"}, reqParams)
	if _, err := httpkg.Post(ctx, url.String(), nil); err != nil {
		return err
	}
	return nil
}

// ListFileAuths get the list of file authorization applications
func (c *Client) ListFileAuths(ctx context.Context, opt ListFileAuthOptions) (blockchain.FileAuthApplications, error) {
	reqParams := map[string]string{
		"applierPubkey":    opt.Applier,
		"authorizerPubkey": opt.Owner,
		"fileID":           opt.FileID,
		"status":           opt.Status,
		"start":            strconv.FormatInt(opt.TimeStart, 10),
		"end":              strconv.FormatInt(opt.TimeEnd, 10),
		"limit":            strconv.FormatInt(opt.Limit, 10),
	}

	var fileAuths blockchain.FileAuthApplications
	url := c.getRequestsUrl([]string{"file", "listauth"}, reqParams)
	if err := httpkg.GetResponse(ctx, url.String(), &fileAuths); err != nil {
		return nil, err
	}
	return fileAuths, nil
}

// GetChallengeByID get challenge info by challenge id
func (c *Client) GetChallengeByID(ctx context.Context, id string) (blockchain.Challenge, error) {
	var challenge blockchain.Challenge
	url := c.getRequestsUrl([]string{"challenge", "getbyid"}, map[string]string{"id": id})
	if err := httpkg.GetResponse(ctx, url.String(), &challenge); err != nil {
		return challenge, err
	}
	return challenge, nil
}

// GetChallenges get challenges with status "ToProve" or "Proved" or "Failed"
func (c *Client) GetChallenges(ctx context.Context, opt GetChallengesOptions, status string) ([]blockchain.Challenge, error) {
	reqParams := map[string]string{
		"owner": opt.Owner,
		"node":  opt.TargetNode,
		"file":  opt.FileID,
		"start": strconv.FormatInt(opt.TimeStart, 10),
		"end":   strconv.FormatInt(opt.TimeEnd, 10),
		"limit": strconv.FormatInt(int64(opt.Limit), 10),
	}
	var url url.URL
	switch status {
	case blockchain.ChallengeToProve:
		url = c.getRequestsUrl([]string{"challenge", "toprove"}, reqParams)
	case blockchain.ChallengeProved:
		url = c.getRequestsUrl([]string{"challenge", "proved"}, reqParams)
	case blockchain.ChallengeFailed:
		url = c.getRequestsUrl([]string{"challenge", "failed"}, reqParams)
	}

	var challenges []blockchain.Challenge
	if err := httpkg.GetResponse(ctx, url.String(), &challenges); err != nil {
		return challenges, err
	}
	return challenges, nil
}
