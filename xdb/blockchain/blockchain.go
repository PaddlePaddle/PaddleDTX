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

// define variables about challenge/fileAuth status
const (
	// Define the status of the challenge stored in Contract
	ChallengeToProve = "ToProve"
	ChallengeProved  = "Proved"
	ChallengeFailed  = "Failed"

	// Define File authorization status stored in Contract
	FileAuthUnapproved = "Unapproved" // the applier published file's authorization application and the authorizer has not yet approved
	FileAuthApproved   = "Approved"   // the authorizer approved applier's authorization application
	FileAuthRejected   = "Rejected"   // the authorizer rejected applier's authorization application
)

// define variables about node health
const (
	// Define the status of the node health
	NodeHealthGood   = "Green"
	NodeHealthMedium = "Yellow"
	NodeHealthBad    = "Red"

	NodeHealthTimeDur = 7 // days
	HeartBeatFreq     = time.Minute
	HeartBeatPerDay   = 1440

	DefaultChallProvedRate = 0.85
	DefaultHearBeatRate    = 0.85
	// Define the challenge ratio and the health ratio
	// Storage node's health is determined by the approved challenge ratio and the heartbeat ratio
	NodeHealthChallProp     = 0.7
	NodeHealthHeartBeatProp = 0.3

	// Define the threshold for the storage node's health
	NodeHealthBoundGood   = 0.85 // health ratio greater than 0.85 means node's status is Green
	NodeHealthBoundMedium = 0.6  // health ratio between 0.6 and 0.85 means node's status is Yellow
)

// define variables about monitor module
const (
	FileRetainPeriod = 7 * 24 * time.Hour
)

// define variables about contract request
const (
	// Define the maximum number of list query
	ListMaxNumber = 100
)

// PublicSliceMeta public, description of a slice stored on a specific node
type PublicSliceMeta struct {
	ID         string // slice ID
	CipherHash []byte // hash of cipher text
	Length     uint64 // length of cipher text
	NodeID     []byte // where slice is stored
	// for pairing based challenge
	SliceIdx int // slice index stored on this node, like 1,2,3...
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

	// for pairing based challenge
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

	ChallengeAlgorithm string   // challenge algorithm
	SliceIDs           []string // slice IDs to challenge
	Indices            [][]byte // indices of slice IDs
	Vs                 [][]byte // random params
	Round              int64    // challenge found
	RandThisRound      []byte   // random number for the challenge

	SliceID     string
	Ranges      []Range
	HashOfProof []byte

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
	Limit       int64 // file number limit
}

type ListChallengeOptions struct {
	FileOwner  []byte // file owner
	TargetNode []byte // storage node
	FileID     string // file ID
	Status     string // challenge status

	TimeStart int64 // challenge time period
	TimeEnd   int64
	Limit     int64 // challenge limit
}

// ChallengeRequestOptions used for dataOwner nodes to add challenge request on chain
type ChallengeRequestOptions struct {
	ChallengeID   string
	FileOwner     []byte
	TargetNode    []byte
	FileID        string
	SliceIDs      []string
	ChallengeTime int64

	ChallengeAlgorithm string

	Indices       [][]byte
	Vs            [][]byte
	Round         int64
	RandThisRound []byte

	SliceID     string
	Ranges      []Range
	HashOfProof []byte

	Sig []byte
}

type Range struct {
	Start uint64
	End   uint64
}

// ChallengeAnswerOptions used for storage nodes to answer challenge request on chain
type ChallengeAnswerOptions struct {
	ChallengeID string
	Sigma       []byte
	Mu          []byte
	Sig         []byte
	AnswerTime  int64

	Proof []byte
}

type Nodes []Node

// Node define storage node info stored on chain
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

type UpdateExptimeOptions struct {
	FileID        string
	NewExpireTime int64
	CurrentTime   int64
	Signature     []byte
}

// UpdateFilePSMOptions used to update the slice public info on chain
// when the dataOwner migrates slice from bad storage node to good storage node
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
	Limit     int64
}

type NodeSliceMigrateOptions ListNodeSliceOptions

type AddNsOptions struct {
	Namespace Namespace
	Signature []byte
}

// Namespace define file namespace stored on chain
// Used by dataOwner to store files, like a folder
type Namespace struct {
	Name         string
	Description  string
	Owner        []byte // file namespace owner
	Replica      int    // The replicas of files under the namespace
	FileTotalNum int64
	CreateTime   int64
	UpdateTime   int64
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

// FileAuthApplication define the file's authorization application stored on chain
type FileAuthApplication struct {
	ID           string
	FileID       string // file ID to authorize
	Name         string
	Description  string
	Applier      []byte // applier's public key, who needs to use files
	Authorizer   []byte // file's owner
	AuthKey      []byte // authorization key, appliers used the key to decrypt the file
	Status       string
	RejectReason string // reason of the rejected authorization
	CreateTime   int64
	ApprovalTime int64 // time when authorizer confirmed or rejected the authorization
	ExpireTime   int64 // expiration time for file use

	// extension
	Ext []byte
}

type FileAuthApplications []*FileAuthApplication

// PublishFileAuthOptions parameters for appliers to publish file authorization application
type PublishFileAuthOptions struct {
	FileAuthApplication FileAuthApplication
	Signature           []byte
}

// ConfirmFileAuthOptions parameters for authorizers to confirm or reject file authorization application
type ConfirmFileAuthOptions struct {
	ID           string
	AuthKey      []byte // authorized file decryption key, if authorizer confirms authorization, it cannot be empty
	RejectReason string // if authorizer rejects authorization, it cannot be empty
	CurrentTime  int64
	ExpireTime   int64

	Signature []byte // authorizer's signature
}

// ListFileAuthOptions parameters for authorizers or appliers to query the list of file authorization application
type ListFileAuthOptions struct {
	Applier    []byte // applier's public key
	Authorizer []byte // authorizer's public key
	FileID     string
	Status     string // file authorization application status
	TimeStart  int64
	TimeEnd    int64
	Limit      int64 // limit number of applications in list request
}
