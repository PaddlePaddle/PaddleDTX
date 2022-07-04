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
	"net/url"
	"path"
)

func joinPath(base *url.URL, paths ...string) {
	ps := append([]string{base.Path}, paths...)
	base.Path = path.Join(ps...)
}

// WriteOptions define the parameters required to upload the file
type WriteOptions struct {
	PrivateKey string

	Namespace   string
	FileName    string
	ExpireTime  int64
	Description string
	Extra       string
}

// ReadOptions download files using FileID or Namespace+FileName
type ReadOptions struct {
	PrivateKey string

	Namespace string
	FileName  string

	FileID string
}

// ListFileOptions support paging query
type ListFileOptions struct {
	Owner     string
	Namespace string

	TimeStart int64
	TimeEnd   int64
	Limit     int64
}

// ListNsOptions support paging query
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

// GetChallengesOptions support paging query
type GetChallengesOptions struct {
	Owner      string
	TargetNode string
	FileID     string // optional, filter

	TimeStart int64
	TimeEnd   int64
	Limit     int64
}
