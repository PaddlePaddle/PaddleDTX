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
	"fmt"
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

type WriteOptions struct {
	PrivateKey string

	Namespace   string
	FileName    string
	ExpireTime  int64
	Description string
	Extra       string
}

// Write upload a file
func (c *Client) Write(ctx context.Context, r io.Reader, opt WriteOptions) (
	servertypes.WriteResponse, error) {

	privkey, err := ecdsa.DecodePrivateKeyFromString(opt.PrivateKey)
	if err != nil {
		return servertypes.WriteResponse{}, err
	}
	pubkey := ecdsa.PublicKeyFromPrivateKey(privkey)
	owner := pubkey.String()

	msg := fmt.Sprintf("%s,%s,%s", owner, opt.Namespace, opt.FileName)
	h := hash.HashUsingSha256([]byte(msg))

	sig, err := ecdsa.Sign(privkey, h)
	if err != nil {
		return servertypes.WriteResponse{}, errorx.Wrap(err, "failed to sign")
	}

	url := c.baseAddr
	joinPath(&url, "file", "write")

	q := url.Query()
	q.Add("user", owner)
	q.Add("token", sig.String())
	q.Add("ns", opt.Namespace)
	q.Add("name", opt.FileName)
	q.Add("desc", opt.Description)
	q.Add("ext", opt.Extra)
	q.Add("expireTime", strconv.FormatInt(opt.ExpireTime, 10))
	url.RawQuery = q.Encode()

	var resp servertypes.WriteResponse
	if err := httpkg.PostResponse(ctx, url.String(), r, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// ReadOptions use FileID or Namespace+FileName
type ReadOptions struct {
	PrivateKey string

	Namespace string
	FileName  string

	FileID string
}

// Read download a file
func (c *Client) Read(ctx context.Context, opt ReadOptions) (io.ReadCloser, error) {
	privkey, err := ecdsa.DecodePrivateKeyFromString(opt.PrivateKey)
	if err != nil {
		return nil, err
	}
	tm := strconv.FormatInt(time.Now().UnixNano(), 10)
	pubkey := ecdsa.PublicKeyFromPrivateKey(privkey)

	var msg string
	if len(opt.FileID) > 0 {
		msg = fmt.Sprintf("%s,%s", opt.FileID, tm)
	} else {
		msg = fmt.Sprintf("%s,%s,%s,%s", pubkey.String(), opt.Namespace, opt.FileName, tm)
	}
	h := hash.HashUsingSha256([]byte(msg))

	sig, err := ecdsa.Sign(privkey, h)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to sign")
	}

	url := c.baseAddr
	joinPath(&url, "file", "read")

	q := url.Query()
	q.Add("user", pubkey.String())
	q.Add("token", sig.String())
	q.Add("ns", opt.Namespace)
	q.Add("name", opt.FileName)
	q.Add("file_id", opt.FileID)
	q.Add("timestamp", tm)
	url.RawQuery = q.Encode()

	reader, err := httpkg.Get(ctx, url.String())
	if err != nil {
		return nil, err
	}

	return reader, nil
}

// ListNodes list all storage nodes in system
func (c *Client) ListNodes(ctx context.Context) (blockchain.Nodes, error) {
	url := c.baseAddr
	joinPath(&url, "node", "list")
	var nodes blockchain.Nodes
	if err := httpkg.GetResponse(ctx, url.String(), &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNode get storage node by node id
func (c *Client) GetNode(ctx context.Context, id string) (blockchain.Node, error) {
	url := c.baseAddr
	joinPath(&url, "node", "get")
	q := url.Query()
	q.Add("id", id)
	url.RawQuery = q.Encode()
	var node blockchain.Node
	if err := httpkg.GetResponse(ctx, url.String(), &node); err != nil {
		return node, err
	}
	return node, nil
}

// GetNodeHeartbeat get storage node heart beat number
func (c *Client) GetNodeHeartbeat(ctx context.Context, id string, ctime int64) (map[string]int, error) {
	url := c.baseAddr
	joinPath(&url, "node", "gethbnum")
	q := url.Query()
	q.Add("id", id)
	q.Add("ctime", strconv.FormatInt(ctime, 10))
	url.RawQuery = q.Encode()
	var res map[string]int
	if err := httpkg.GetResponse(ctx, url.String(), &res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetMigrateRecords get storage node migrate records
func (c *Client) GetMigrateRecords(ctx context.Context, id string, start, end int64, limit int64) ([]map[string]interface{}, error) {
	url := c.baseAddr
	joinPath(&url, "node", "getmrecord")
	q := url.Query()
	q.Add("id", id)
	q.Add("start", strconv.FormatInt(start, 10))
	q.Add("end", strconv.FormatInt(end, 10))
	q.Add("limit", strconv.FormatInt(limit, 10))
	url.RawQuery = q.Encode()
	var ms []map[string]interface{}
	if err := httpkg.GetResponse(ctx, url.String(), &ms); err != nil {
		return nil, err
	}
	return ms, nil
}

// GetNodeHealth get storage node health status by node id
func (c *Client) GetNodeHealth(ctx context.Context, id string) (string, error) {
	url := c.baseAddr
	joinPath(&url, "node", "health")
	q := url.Query()
	q.Add("id", id)
	url.RawQuery = q.Encode()
	var status string
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
	pubkey := ecdsa.PublicKeyFromPrivateKey(private)
	nodeID := pubkey.String()

	nonce := time.Now().UnixNano()
	m := fmt.Sprintf("%s,%d", nodeID, nonce)
	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign")
	}

	url := c.baseAddr
	if online {
		joinPath(&url, "node", "online")
	} else {
		joinPath(&url, "node", "offline")
	}
	q := url.Query()
	q.Add("node", nodeID)
	q.Add("nonce", strconv.FormatInt(nonce, 10))
	q.Add("token", sig.String())
	url.RawQuery = q.Encode()
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

type ListFileOptions struct {
	Owner     string
	Namespace string

	TimeStart int64
	TimeEnd   int64
	Limit     int64
}

type ListNsOptions struct {
	Owner string

	TimeStart int64
	TimeEnd   int64
	Limit     int64
}

// ListFileAuthOptions define parameters for authorizers or appliers to query the list of file authorization application
type ListFileAuthOptions struct {
	Owner     string
	Applier   string
	FileID    string
	Status    string
	TimeStart int64
	TimeEnd   int64
	Limit     int64
}

// ConfirmAuthOptions define parameters for authorizers to confirm the file authorization application
type ConfirmAuthOptions struct {
	PrivateKey   string
	AuthID       string
	ExpireTime   int64
	RejectReason string
	Status       bool
}

// ListFiles list unexpired files
func (c *Client) ListFiles(ctx context.Context, opt ListFileOptions) ([]blockchain.File, error) {
	url := c.baseAddr
	joinPath(&url, "file", "list")
	q := url.Query()
	q.Add("owner", opt.Owner)
	q.Add("ns", opt.Namespace)
	q.Add("start", strconv.FormatInt(opt.TimeStart, 10))
	q.Add("end", strconv.FormatInt(opt.TimeEnd, 10))
	q.Add("limit", strconv.FormatInt(opt.Limit, 10))
	url.RawQuery = q.Encode()
	var files []blockchain.File
	if err := httpkg.GetResponse(ctx, url.String(), &files); err != nil {
		return nil, err
	}
	return files, nil
}

// ListExpiredFiles list expired but valid files
func (c *Client) ListExpiredFiles(ctx context.Context, opt ListFileOptions) ([]blockchain.File, error) {
	url := c.baseAddr
	joinPath(&url, "file", "listexp")
	q := url.Query()
	q.Add("owner", opt.Owner)
	q.Add("ns", opt.Namespace)
	q.Add("start", strconv.FormatInt(opt.TimeStart, 10))
	q.Add("end", strconv.FormatInt(opt.TimeEnd, 10))
	q.Add("limit", strconv.FormatInt(opt.Limit, 10))
	url.RawQuery = q.Encode()
	var files []blockchain.File
	if err := httpkg.GetResponse(ctx, url.String(), &files); err != nil {
		return nil, err
	}
	return files, nil
}

// GetFileByID get file info by file id
func (c *Client) GetFileByID(ctx context.Context, id string) (blockchain.FileH, error) {
	url := c.baseAddr
	joinPath(&url, "file", "getbyid")
	q := url.Query()
	q.Add("id", id)
	url.RawQuery = q.Encode()
	var hfile blockchain.FileH
	if err := httpkg.GetResponse(ctx, url.String(), &hfile); err != nil {
		return hfile, err
	}
	return hfile, nil
}

// GetFileByName get file info by file name, owner and namespace
func (c *Client) GetFileByName(ctx context.Context, owner, ns, name string) (blockchain.FileH, error) {
	url := c.baseAddr
	joinPath(&url, "file", "getbyname")
	q := url.Query()
	q.Add("owner", owner)
	q.Add("ns", ns)
	q.Add("name", name)
	url.RawQuery = q.Encode()
	var hfile blockchain.FileH
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
	pubkey := ecdsa.PublicKeyFromPrivateKey(private)

	currentTime := time.Now().UnixNano()
	m := fmt.Sprintf("%s,%d,%d", id, expireTime, currentTime)
	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign file expire time")
	}

	url := c.baseAddr
	joinPath(&url, "file", "updatexptime")
	q := url.Query()
	q.Add("id", id)
	q.Add("expireTime", strconv.FormatInt(expireTime, 10))
	q.Add("ctime", strconv.FormatInt(currentTime, 10))
	q.Add("user", pubkey.String())
	q.Add("token", sig.String())
	url.RawQuery = q.Encode()
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
	pubkey := ecdsa.PublicKeyFromPrivateKey(private)

	ctime := time.Now().UnixNano()
	m := fmt.Sprintf("%s,%s,%d,%d", ns, des, ctime, replica)

	// sign ns info
	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign file namespace")
	}
	url := c.baseAddr
	joinPath(&url, "file", "addns")
	q := url.Query()
	q.Add("ns", ns)
	q.Add("replica", strconv.Itoa(replica))
	q.Add("user", pubkey.String())
	q.Add("token", sig.String())
	q.Add("ctime", strconv.FormatInt(ctime, 10))
	q.Add("desc", des)
	url.RawQuery = q.Encode()
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
	pubkey := ecdsa.PublicKeyFromPrivateKey(private)

	currentTime := time.Now().UnixNano()
	m := fmt.Sprintf("%s,%d,%d", ns, replica, currentTime)
	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign update ns replica param")
	}

	url := c.baseAddr
	joinPath(&url, "file", "ureplica")
	q := url.Query()
	q.Add("ns", ns)
	q.Add("replica", strconv.Itoa(replica))
	q.Add("ctime", strconv.FormatInt(currentTime, 10))
	q.Add("user", pubkey.String())
	q.Add("token", sig.String())
	url.RawQuery = q.Encode()
	if _, err := httpkg.Post(ctx, url.String(), nil); err != nil {
		return err
	}
	return nil
}

// ListFileNs list file namespaces
func (c *Client) ListFileNs(ctx context.Context, opt ListNsOptions) ([]blockchain.Namespace, error) {
	url := c.baseAddr
	joinPath(&url, "file", "listns")
	q := url.Query()
	q.Add("owner", opt.Owner)
	q.Add("start", strconv.FormatInt(opt.TimeStart, 10))
	q.Add("end", strconv.FormatInt(opt.TimeEnd, 10))
	q.Add("limit", strconv.FormatInt(opt.Limit, 10))
	url.RawQuery = q.Encode()
	var nss []blockchain.Namespace
	if err := httpkg.GetResponse(ctx, url.String(), &nss); err != nil {
		return nil, err
	}
	return nss, nil
}

// GetNsByName get namespace by name and owner
func (c *Client) GetNsByName(ctx context.Context, owner, ns string) (blockchain.NamespaceH, error) {
	url := c.baseAddr
	joinPath(&url, "file", "getns")
	q := url.Query()
	q.Add("owner", owner)
	q.Add("name", ns)
	url.RawQuery = q.Encode()
	var nsh blockchain.NamespaceH
	if err := httpkg.GetResponse(ctx, url.String(), &nsh); err != nil {
		return nsh, err
	}
	return nsh, nil
}

// GetFileSysHealth get system health status of an owner
func (c *Client) GetFileSysHealth(ctx context.Context, owner string) (blockchain.FileSysHealth, error) {
	url := c.baseAddr
	joinPath(&url, "file", "getsyshealth")
	q := url.Query()
	q.Add("owner", owner)
	url.RawQuery = q.Encode()
	var fh blockchain.FileSysHealth
	if err := httpkg.GetResponse(ctx, url.String(), &fh); err != nil {
		return fh, err
	}
	return fh, nil
}

// GetAuth get file authorization application detail by authID
func (c *Client) GetAuth(ctx context.Context, authID string) (blockchain.FileAuthApplication, error) {
	url := c.baseAddr
	joinPath(&url, "file", "getauthbyid")
	q := url.Query()
	q.Add("authID", authID)
	url.RawQuery = q.Encode()
	var fa blockchain.FileAuthApplication
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
	pubkey := ecdsa.PublicKeyFromPrivateKey(private)
	m := fmt.Sprintf("%s,%d,%s", opt.AuthID, opt.ExpireTime, opt.RejectReason)
	sig, err := ecdsa.Sign(private, hash.HashUsingSha256([]byte(m)))
	if err != nil {
		return errorx.Wrap(err, "failed to sign confirm file authorization application")
	}

	url := c.baseAddr
	joinPath(&url, "file", "confirmauth")
	q := url.Query()
	q.Add("authID", opt.AuthID)
	q.Add("status", strconv.FormatBool(opt.Status))
	q.Add("user", pubkey.String())
	q.Add("expireTime", strconv.FormatInt(opt.ExpireTime, 10))
	q.Add("rejectReason", opt.RejectReason)
	q.Add("token", sig.String())

	url.RawQuery = q.Encode()
	if _, err := httpkg.Post(ctx, url.String(), nil); err != nil {
		return err
	}
	return nil
}

// ListFileAuths get the list of file authorization applications
func (c *Client) ListFileAuths(ctx context.Context, opt ListFileAuthOptions) (blockchain.FileAuthApplications, error) {
	url := c.baseAddr
	joinPath(&url, "file", "listauth")
	q := url.Query()
	q.Add("applierPubkey", opt.Applier)
	q.Add("authorizerPubkey", opt.Owner)
	q.Add("fileID", opt.FileID)
	q.Add("status", opt.Status)
	q.Add("start", strconv.FormatInt(opt.TimeStart, 10))
	q.Add("end", strconv.FormatInt(opt.TimeEnd, 10))
	q.Add("limit", strconv.FormatInt(opt.Limit, 10))
	url.RawQuery = q.Encode()
	var fileAuths blockchain.FileAuthApplications
	if err := httpkg.GetResponse(ctx, url.String(), &fileAuths); err != nil {
		return nil, err
	}
	return fileAuths, nil
}

// GetChallengeByID get challenge info by challenge id
func (c *Client) GetChallengeByID(ctx context.Context, id string) (blockchain.Challenge, error) {
	url := c.baseAddr
	joinPath(&url, "challenge", "getbyid")
	q := url.Query()
	q.Add("id", id)
	url.RawQuery = q.Encode()
	var challenge blockchain.Challenge
	if err := httpkg.GetResponse(ctx, url.String(), &challenge); err != nil {
		return challenge, err
	}
	return challenge, nil
}

type GetChallengesOptions struct {
	Owner      string
	TargetNode string
	FileID     string // optional, filter

	TimeStart int64
	TimeEnd   int64
	Limit     int64
}

// GetToProveChallenges get challenges with status "ToProve"
func (c *Client) GetToProveChallenges(ctx context.Context, opt GetChallengesOptions) ([]blockchain.Challenge, error) {
	url := c.baseAddr
	joinPath(&url, "challenge", "toprove")
	q := url.Query()
	q.Add("owner", opt.Owner)
	q.Add("node", opt.TargetNode)
	q.Add("file", opt.FileID)
	q.Add("start", strconv.FormatInt(opt.TimeStart, 10))
	q.Add("end", strconv.FormatInt(opt.TimeEnd, 10))
	q.Add("limit", strconv.FormatInt(int64(opt.Limit), 10))
	url.RawQuery = q.Encode()
	var challenges []blockchain.Challenge
	if err := httpkg.GetResponse(ctx, url.String(), &challenges); err != nil {
		return challenges, err
	}
	return challenges, nil
}

// GetProvedChallenges get challenges with status "Proved"
func (c *Client) GetProvedChallenges(ctx context.Context, opt GetChallengesOptions) ([]blockchain.Challenge, error) {
	url := c.baseAddr
	joinPath(&url, "challenge", "proved")
	q := url.Query()
	q.Add("owner", opt.Owner)
	q.Add("node", opt.TargetNode)
	q.Add("file", opt.FileID)
	q.Add("start", strconv.FormatInt(opt.TimeStart, 10))
	q.Add("end", strconv.FormatInt(opt.TimeEnd, 10))
	q.Add("limit", strconv.FormatInt(int64(opt.Limit), 10))
	url.RawQuery = q.Encode()
	var challenges []blockchain.Challenge
	if err := httpkg.GetResponse(ctx, url.String(), &challenges); err != nil {
		return challenges, err
	}
	return challenges, nil
}

// GetFailedChallenges get challenges with status "Failed"
func (c *Client) GetFailedChallenges(ctx context.Context, opt GetChallengesOptions) ([]blockchain.Challenge, error) {
	url := c.baseAddr
	joinPath(&url, "challenge", "failed")
	q := url.Query()
	q.Add("owner", opt.Owner)
	q.Add("node", opt.TargetNode)
	q.Add("file", opt.FileID)
	q.Add("start", strconv.FormatInt(opt.TimeStart, 10))
	q.Add("end", strconv.FormatInt(opt.TimeEnd, 10))
	q.Add("limit", strconv.FormatInt(int64(opt.Limit), 10))
	url.RawQuery = q.Encode()
	var challenges []blockchain.Challenge
	if err := httpkg.GetResponse(ctx, url.String(), &challenges); err != nil {
		return challenges, err
	}
	return challenges, nil
}
