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
	ID         string `json:"id"`         // slice ID
	CipherHash []byte `json:"cipherHash"` // hash of cipher text
	Length     uint64 `json:"length"`     // length of cipher text
	NodeID     []byte `json:"nodeID"`     // where slice is stored
	StorIndex  string `json:"storIndex"`  // storage index of slice, is used to query a slice from Storage, created by StorageNode

	// for pairing based challenge
	SliceIdx int `json:"sliceIdx"` // slice index stored on this node, like 1,2,3...
}

// PrivateSliceMeta private, description of the order of original slices
type PrivateSliceMeta struct {
	SliceID   string `json:"sliceID"`   // slice ID
	PlainHash []byte `json:"plainHash"` // hash of plain text
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
	ID          string            `json:"id"`          // file ID, generate by engine
	Name        string            `json:"name"`        // file name, user input
	Description string            `json:"description"` // file description, user input (optional)
	Namespace   string            `json:"namespace"`   // file namespace, user input
	Owner       []byte            `json:"owner"`       // owner, user input
	Length      uint64            `json:"length"`      // plain text length
	MerkleRoot  []byte            `json:"merkleRoot"`  // merkle root of slices (plain text)
	Slices      []PublicSliceMeta `json:"slices"`      // unordered slices
	Structure   []byte            `json:"structure"`   // encrypted FileStructure
	PublishTime int64             `json:"publishTime"` // publish time on blockchain
	ExpireTime  int64             `json:"expireTime"`  // file expire time

	// for pairing based challenge
	PdpPubkey []byte `json:"pdpPubkey"`
	RandU     []byte `json:"randU"`
	RandV     []byte `json:"randV"`

	// extension
	Ext []byte `json:"ext"`
}

type FileH struct {
	File   File   `json:"file"`
	Health string `json:"health"`
}

type PublishFileOptions struct {
	File      File   `json:"file"`
	Signature []byte `json:"signature"`
}

// Challenge public information stored on chain
type Challenge struct {
	ID         string `json:"id"`         // challenge ID
	FileOwner  []byte `json:"fileOwner"`  // file owner
	TargetNode []byte `json:"targetNode"` // storage node
	FileID     string `json:"fileID"`     // file ID

	ChallengeAlgorithm string   `json:"challengeAlgorithm"` // challenge algorithm
	SliceIDs           []string `json:"sliceIDs"`           // slice IDs to challenge
	SliceStorIndexes   []string `json:"sliceStorIndexes"`   // storage index of slice, is used to query a slice from Storage
	Indices            [][]byte `json:"indices"`            // indices of slice IDs
	Vs                 [][]byte `json:"vs"`                 // random params
	Round              int64    `json:"round"`              // challenge found
	RandThisRound      []byte   `json:"randThisRound"`      // random number for the challenge

	SliceID        string  `json:"sliceID"`
	SliceStorIndex string  `json:"sliceStorIndex"` // storage index of slice, is used to query a slice from Storage
	Ranges         []Range `json:"ranges"`
	HashOfProof    []byte  `json:"hashOfProof"`

	Status        string `json:"status"`        // challenge status
	ChallengeTime int64  `json:"challengeTime"` // challenge publish time
	AnswerTime    int64  `json:"answerTime"`    // challenge answer time
}

type ListFileOptions struct {
	Owner     []byte `json:"owner"`     // file owner
	Namespace string `json:"namespace"` // file namespace

	TimeStart   int64 `json:"timeStart"`
	TimeEnd     int64 `json:"timeEnd"`
	CurrentTime int64 `json:"currentTime"`
	Limit       int64 `json:"limit"` // file number limit
}

type ListChallengeOptions struct {
	FileOwner  []byte `json:"fileOwner"`  // file owner
	TargetNode []byte `json:"targetNode"` // storage node
	FileID     string `json:"fileID"`     // file ID
	Status     string `json:"status"`     // challenge status

	TimeStart int64 `json:"timeStart"` // challenge time period
	TimeEnd   int64 `json:"timeEnd"`
	Limit     int64 `json:"limit"` // challenge limit
}

// ChallengeRequestOptions used for dataOwner nodes to add challenge request on chain
type ChallengeRequestOptions struct {
	ChallengeID      string   `json:"challengeID"`
	FileOwner        []byte   `json:"fileOwner"`
	TargetNode       []byte   `json:"targetNode"`
	FileID           string   `json:"fileID"`
	SliceIDs         []string `json:"sliceIDs"`
	SliceStorIndexes []string `json:"sliceStorIndexes"` // storage index of slice, is used to query a slice from Storage
	ChallengeTime    int64    `json:"challengeTime"`

	ChallengeAlgorithm string `json:"challengeAlgorithm"`

	Indices       [][]byte `json:"indices"`
	Vs            [][]byte `json:"vs"`
	Round         int64    `json:"round"`
	RandThisRound []byte   `json:"randThisRound"`

	SliceID        string  `json:"sliceID"`
	SliceStorIndex string  `json:"sliceStorIndex"` // storage index of slice, is used to query a slice from Storage
	Ranges         []Range `json:"ranges"`
	HashOfProof    []byte  `json:"hashOfProof"`

	Signature []byte `json:"signature"`
}

type Range struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

// ChallengeAnswerOptions used for storage nodes to answer challenge request on chain
type ChallengeAnswerOptions struct {
	ChallengeID string `json:"challengeID"`
	Sigma       []byte `json:"sigma"`
	Mu          []byte `json:"mu"`
	AnswerTime  int64  `json:"answerTime"`

	Proof     []byte `json:"proof"`
	Signature []byte `json:"signature"`
}

type Nodes []Node

// Node define storage node info stored on chain
type Node struct {
	ID       []byte `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Online   bool   `json:"online"`   // whether node is online or offline
	RegTime  int64  `json:"regTime"`  // node register time
	UpdateAt int64  `json:"updateAt"` // node recent update time
}

type NodeH struct {
	Node   Node   `json:"node"`
	Health string `json:"health"`
}

type NodeHs []NodeH

// AddNodeOptions used to add storgae node into blockchain
type AddNodeOptions struct {
	Node      Node   `json:"node"`
	Signature []byte `json:"signature"`
}

// NodeOperateOptions used to online or offline storage node
type NodeOperateOptions struct {
	NodeID    []byte `json:"nodeID"`
	Nonce     int64  `json:"nonce"`
	Signature []byte `json:"signature"`
}

// NodeHeartBeatOptions define parameters for heartbeat detection of storage nodes
type NodeHeartBeatOptions struct {
	NodeID        []byte `json:"nodeID"`
	CurrentTime   int64  `json:"currentTime"`
	BeginningTime int64  `json:"beginningTime"`
	Signature     []byte `json:"signature"`
}

// UpdateExptimeOptions used to update file's expireTime
type UpdateExptimeOptions struct {
	FileID        string `json:"fileID"`
	NewExpireTime int64  `json:"newExpireTime"`
	CurrentTime   int64  `json:"currentTime"`
	Signature     []byte `json:"signature"`
}

// UpdateFilePSMOptions used to update the slice public info on chain
// when the dataOwner migrates slice from bad storage node to good storage node
type UpdateFilePSMOptions struct {
	FileID    string            `json:"fileID"`
	Owner     []byte            `json:"owner"`
	Slices    []PublicSliceMeta `json:"slices"`
	Signature []byte            `json:"signature"`
}

// UpdateNsReplicaOptions used to update the replica on the blockchain
type UpdateNsReplicaOptions struct {
	Owner       []byte `json:"owner"`
	Name        string `json:"name"`
	Replica     int    `json:"replica"`
	CurrentTime int64  `json:"currentTime"`
	Signature   []byte `json:"signature"`
}

// SliceMigrateOptions used to record migrate info for storage node
type SliceMigrateOptions struct {
	NodeID      []byte `json:"nodeID"`
	FileID      string `json:"fileID"`
	SliceID     string `json:"sliceID"`
	CurrentTime int64  `json:"currentTime"`
	Signature   []byte `json:"signature"`
}

type ListNodeSliceOptions struct {
	Target []byte `json:"target"`

	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`
	Limit     int64 `json:"limit"`
}

type NodeSliceMigrateOptions ListNodeSliceOptions

type AddNsOptions struct {
	Namespace Namespace `json:"namespace"`
	Signature []byte    `json:"signature"`
}

// Namespace define file namespace stored on chain
// Used by dataOwner to store files, like a folder
type Namespace struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Owner        []byte `json:"owner"`   // file namespace owner
	Replica      int    `json:"replica"` // The replicas of files under the namespace
	FileTotalNum int64  `json:"fileTotalNum"`
	CreateTime   int64  `json:"createTime"`
	UpdateTime   int64  `json:"updateTime"`
}

// NamespaceH used to list file's information under namespace
type NamespaceH struct {
	Namespace      Namespace `json:"namespace"`
	FileNormalNum  int       `json:"fileNormalNum"`
	FileExpiredNum int       `json:"fileExpiredNum"`
	GreenFileNum   int       `json:"greenFileNum"`
	YellowFileNum  int       `json:"yellowFileNum"`
	RedFileNum     int       `json:"redFileNum"`
}

// FileSysHealth describes system health status for a file owner
type FileSysHealth struct {
	FileNum         int     `json:"fileNum"`         // total files number, includes expired files
	FileExpiredNum  int     `json:"fileExpiredNum"`  // expired files number
	NsNum           int     `json:"nsNum"`           // namespace number
	GreenFileNum    int     `json:"greenFileNum"`    // green files number
	YellowFileNum   int     `json:"yellowFileNum"`   // yellow files number
	RedFileNum      int     `json:"redFileNum"`      // red files number
	SysHealth       string  `json:"sysHealth"`       // system health status, green/yellow/red
	FilesHealthRate float64 `json:"filesHealthRate"` // file health rate

	NodeNum        int     `json:"nodeNum"`        // total nodes number
	GreenNodeNum   int     `json:"greenNodeNum"`   // green node number
	YellowNodeNum  int     `json:"yellowNodeNum"`  // yellow node number
	RedNodeNum     int     `json:"redNodeNum"`     // red node number
	NodeHealthRate float64 `json:"nodeHealthRate"` // node health rate
}

type ListNsOptions ListFileOptions

type GetChallengeNumOptions struct {
	TargetNode []byte `json:"targetNode"` // storage node
	Status     string `json:"status"`     // challenge status, optional

	TimeStart int64 `json:"timeStart"` // challenge time period
	TimeEnd   int64 `json:"timeEnd"`
}

// FileAuthApplication define the file's authorization application stored on chain
type FileAuthApplication struct {
	ID           string `json:"id"`
	FileID       string `json:"fileID"` // file ID to authorize
	Name         string `json:"name"`
	Description  string `json:"description"`
	Applier      []byte `json:"applier"`    // applier's public key, who needs to use files
	Authorizer   []byte `json:"authorizer"` // file's owner
	AuthKey      []byte `json:"authKey"`    // authorization key, appliers used the key to decrypt the file
	Status       string `json:"status"`
	RejectReason string `json:"rejectReason"` // reason of the rejected authorization
	CreateTime   int64  `json:"createTime"`
	ApprovalTime int64  `json:"approvalTime"` // time when authorizer confirmed or rejected the authorization
	ExpireTime   int64  `json:"expireTime"`   // expiration time for file use

	// extension
	Ext []byte `json:"ext"`
}

type FileAuthApplications []*FileAuthApplication

// PublishFileAuthOptions parameters for appliers to publish file authorization application
type PublishFileAuthOptions struct {
	FileAuthApplication FileAuthApplication `json:"fileAuthApplication"`
	Signature           []byte              `json:"signature"`
}

// ConfirmFileAuthOptions parameters for authorizers to confirm or reject file authorization application
type ConfirmFileAuthOptions struct {
	ID           string `json:"id"`
	AuthKey      []byte `json:"authKey"`      // authorized file decryption key, if authorizer confirms authorization, it cannot be empty
	RejectReason string `json:"rejectReason"` // if authorizer rejects authorization, it cannot be empty
	CurrentTime  int64  `json:"currentTime"`
	ExpireTime   int64  `json:"expireTime"`

	Signature []byte `json:"signature"` // authorizer's signature
}

// ListFileAuthOptions parameters for authorizers or appliers to query the list of file authorization application
type ListFileAuthOptions struct {
	Applier    []byte `json:"applier"`    // applier's public key
	Authorizer []byte `json:"authorizer"` // authorizer's public key
	FileID     string `json:"fileID"`
	Status     string `json:"status"` // file authorization application status
	TimeStart  int64  `json:"timeStart"`
	TimeEnd    int64  `json:"timeEnd"`
	Limit      int64  `json:"limit"` // limit number of applications in list request
}
