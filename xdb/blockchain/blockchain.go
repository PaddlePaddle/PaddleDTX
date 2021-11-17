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

package blockchain

import (
	"encoding/json"
	"time"
)

const (
	ChallengeToProve = "ToProve"
	ChallengeProved  = "Proved"
	ChallengeFailed  = "Failed"

	NodeHealthGood   = "Green"
	NodeHealthMedium = "Yellow"
	NodeHealthBad    = "Red"

	NodeHealthTimeDur = 7 // days
	HeartBeatFreq     = time.Minute
	HeartBeatPerDay   = 1440

	DefaultChallProvedRate = 0.85
	DefaultHearBeatRate    = 0.85

	NodeHealthChallProp     = 0.7
	NodeHealthHeartBeatProp = 0.3

	NodeHealthBoundGood   = 0.85
	NodeHealthBoundMedium = 0.6

	FileRetainPeriod = 7 * 24 * time.Hour

	ContractMessageMaxSize = 4 * 1024 * 1024
)

// PublicSliceMeta public, description of a slice stored on a specific node
type PublicSliceMeta struct {
	ID         string // slice ID
	CipherHash []byte // hash of cipher text
	Length     uint64 // length of cipher text
	NodeID     []byte // where slice is stored
	// for pdp
	SliceIdx int    // slice index stored on this node, like 0,1,2...
	SigmaI   []byte // computed by index and ciphertext
}

// PrivateSliceMeta private, description of the order of original slices
type PrivateSliceMeta struct {
	SliceID   string // slice ID
	PlainHash []byte // hash of plain text
}

type FileStructure []PrivateSliceMeta

func (f *FileStructure) Marshal() ([]byte, error) {
	return json.Marshal(f)
}

func (f *FileStructure) Parse(bs []byte) error {
	return json.Unmarshal(bs, f)
}

// File public information stored on chain
type File struct {
	ID          string            // file ID, generate by engine
	Name        string            // file name, user input
	Description string            // file description, user input (optional)
	Namespace   string            // file namespace, user input
	Owner       []byte            // owner, user input
	Length      uint64            // plain text length
	MerkleRoot  []byte            // merkle root of slices (plain text)
	Slices      []PublicSliceMeta // unordered slices
	Structure   []byte            // encrypted FileStructure
	PublishTime int64             // publish time on blockchain
	ExpireTime  int64             // file expire time

	// for pdp
	PdpPubkey []byte
	RandU     []byte
	RandV     []byte

	// extension
	Ext []byte
}

type FileH struct {
	File   File
	Health string
}

type PublishFileOptions struct {
	File      File
	Signature []byte
}

// Challenge public information stored on chain
type Challenge struct {
	ID         string // challenge ID
	FileOwner  []byte // file owner
	TargetNode []byte // storage node
	FileID     string // file ID

	SliceIDs []string // slice IDs to challenge
	Indices  [][]byte // indices of slice IDs
	Vs       [][]byte // random params
	Sigmas   [][]byte // sigma for each index

	SliceID     string
	Ranges      []Range
	HashOfProof []byte

	ChallengAlgorithm string // challenge algorithm

	Status        string // challenge status
	ChallengeTime int64  // challenge publish time
	AnswerTime    int64  // challenge answer time
}

type ListFileOptions struct {
	Owner     []byte // file owner
	Namespace string // file namespace

	TimeStart   int64
	TimeEnd     int64
	CurrentTime int64
	Limit       uint64 // file number limit
}

type ListChallengeOptions struct {
	FileOwner  []byte // file owner
	TargetNode []byte // storage node
	FileID     string // file ID
	Status     string // challenge status

	TimeStart int64 // challenge time period
	TimeEnd   int64
	Limit     uint64 // challenge limit
}

type ChallengeRequestOptions struct {
	ChallengeID   string
	FileOwner     []byte
	TargetNode    []byte
	FileID        string
	SliceIDs      []string
	ChallengeTime int64

	Indices [][]byte
	Vs      [][]byte
	Sigmas  [][]byte
	Sig     []byte

	SliceID string
	Ranges  []Range

	ChallengAlgorithm string

	HashOfProof []byte
}

type Range struct {
	Start uint64
	End   uint64
}

type ChallengeAnswerOptions struct {
	ChallengeID string
	Sigma       []byte
	Mu          []byte
	Sig         []byte
	AnswerTime  int64

	Proof []byte
}

type Nodes []Node

type Node struct {
	ID       []byte
	Name     string
	Address  string
	Online   bool  // whether node is online or offline
	RegTime  int64 // node register time
	UpdateAt int64 // node recent update time
}

type NodeH struct {
	Node   Node
	Health string
}

type NodeHs []NodeH

type AddNodeOptions struct {
	Node      Node
	Signature []byte
}

type NodeOperateOptions struct {
	NodeID []byte
	Sig    []byte
	Nonce  int64
}

type UpdatExptimeOptions struct {
	FileId        string
	NewExpireTime int64
	CurrentTime   int64
	Signature     []byte
}

type UpdateFilePSMOptions struct {
	FileID    string
	Owner     []byte
	Slices    []PublicSliceMeta
	Signature []byte
}

type UpdateNsReplicaOptions struct {
	Owner       []byte
	Name        string
	Replica     int
	CurrentTime int64
	Signature   []byte
}

type ListNodeSliceOptions struct {
	Target []byte

	StartTime int64
	EndTime   int64
	Limit     uint64
}

type NodeSliceMigrateOptions ListNodeSliceOptions

type AddNsOptions struct {
	Namespace Namespace
	Signature []byte
}

type Namespace struct {
	Name          string
	Description   string
	Owner         []byte
	Replica       int
	FilesStruSize int
	FileTotalNum  int64
	CreateTime    int64
	UpdateTime    int64
}

type NamespaceH struct {
	Namespace      Namespace
	FileNormalNum  int
	FileExpiredNum int
	GreenFileNum   int
	YellowFileNum  int
	RedFileNum     int
}

// FileSysHealth describes system health status for a file owner
type FileSysHealth struct {
	FileNum         int     // total files number, includes expired files
	FileExpiredNum  int     // expired files number
	NsNum           int     // namespace number
	GreenFileNum    int     // green files number
	YellowFileNum   int     // yellow files number
	RedFileNum      int     // red files number
	SysHealth       string  // system health status, green/yellow/red
	FilesHealthRate float64 // file health rate

	NodeNum        int     // total nodes number
	GreenNodeNum   int     // green node number
	YellowNodeNum  int     // yellow node number
	RedNodeNum     int     // red node number
	NodeHealthRate float64 // node health rate
}

type ListNsOptions ListFileOptions

type GetChallengeNumOptions struct {
	TargetNode []byte // storage node
	Status     string // challenge status, optional

	TimeStart int64 // challenge time period
	TimeEnd   int64
}

type UpdateNsFilesCapOptions struct {
	Owner       []byte
	Name        string
	CurrentTime int64
	Signature   []byte
}
