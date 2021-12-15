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

package types

import (
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// WriteOptions options for writing file to system
type WriteOptions struct {
	User        string `json:"user"`
	Token       string `json:"token"`
	Namespace   string `json:"namespace"`
	FileName    string `json:"file_name"`
	ExpireTime  int64  `json:"expire_time"`
	Description string `json:"description"`
	Extra       string `json:"extra"`
}

// Valid checks if WriteOptions is valid
func (o *WriteOptions) Valid() error {
	if len(o.User) == 0 {
		return errorx.New(errorx.ErrCodeParam, "empty user")
	}

	if len(o.Token) == 0 {
		return errorx.New(errorx.ErrCodeParam, "empty token")
	}

	if len(o.Namespace) == 0 {
		return errorx.New(errorx.ErrCodeParam, "empty namespace")
	}

	if len(o.FileName) == 0 {
		return errorx.New(errorx.ErrCodeParam, "empty file name")
	}

	if o.ExpireTime <= time.Now().UnixNano() {
		return errorx.New(errorx.ErrCodeParam, "invalid file expire time")
	}

	return nil
}

// ReadOptions read file from engine
// use user+namespace+filename or fileID to locate a file
// will use fileID first if not empty
type ReadOptions struct {
	User      string `json:"user"`
	Timestamp uint64 `json:"timestamp"`
	Token     string `json:"token"`
	Namespace string `json:"namespace"`
	FileName  string `json:"file_name"`
	FileID    string `json:"file_id"`
}

// Valid check if ReadOptions is valid
func (r *ReadOptions) Valid() error {
	if len(r.User) == 0 {
		return errorx.New(errorx.ErrCodeParam, "empty user")
	}
	if r.Timestamp == 0 {
		return errorx.New(errorx.ErrCodeParam, "empty timestamp")
	}

	var idEmpty, nameEmpty bool

	if len(r.FileID) == 0 {
		idEmpty = true
	}

	if len(r.Token) == 0 || len(r.Namespace) == 0 || len(r.FileName) == 0 {
		nameEmpty = true
	}

	if idEmpty && nameEmpty {
		return errorx.New(errorx.ErrCodeParam, "use id or user+namespace+filename")
	}

	return nil
}

// PushOptions options for pushing slice to storage node
type PushOptions struct {
	SliceID  string `json:"slice_id"`
	SourceId string `json:"source_id"` // dataOwner node id
}

// PullOptions options for pulling slice from storage node
type PullOptions struct {
	SliceID   string
	FileID    string
	Timestamp uint64
	Signature string
}

// AddNodeOptions options for adding storage node to blockchain
type AddNodeOptions struct {
	NodeID  string
	Name    string
	Address string
	Online  bool
	Token   string
}

// NodeOfflineOptions options for setting storage node with offline status on blockchain
type NodeOfflineOptions struct {
	NodeID string
	Nonce  int64
	Token  string
}

// NodeOnlineOptions options for setting storage node with online status on blockchain
type NodeOnlineOptions struct {
	NodeID string
	Nonce  int64
	Token  string
}

// ListFileOptions options for listing files from blockchain
type ListFileOptions struct {
	Owner     []byte // file owner
	Namespace string // file namespace

	TimeStart   int64 // time period
	TimeEnd     int64
	CurrentTime int64  // current time
	Limit       uint64 // file limit
}

// UpdateFileEtimeOptions options for updating file expire time
type UpdateFileEtimeOptions struct {
	Owner       string
	FileID      string
	ExpireTime  int64
	CurrentTime int64
	Token       string
}

// AddNsOptions options for adding namespace on blockchain
type AddNsOptions struct {
	Owner       string
	Namespace   string
	Description string
	Replica     int
	CreateTime  int64
	Token       string
}

// UpdateNsOptions options for updating namespace replica
type UpdateNsOptions struct {
	Owner       string
	Namespace   string
	Replica     int
	CurrentTime int64
	Token       string
}

type ListNsOptions ListFileOptions
