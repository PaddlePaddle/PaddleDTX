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

package config

type DataOwnerConf struct {
	Name          string
	ListenAddress string
	PrivateKey    string
	KeyPath       string
	PublicAddress string

	Slicer     *DataOwnerSlicerConf
	Encryptor  *DataOwnerEncryptorConf
	Blockchain *BlockchainConf
	Copier     *DataOwnerCopierConf
	Monitor    *MonitorConf
	Challenger *DataOwnerChallenger
}

type DataOwnerSlicerConf struct {
	Type         string
	SimpleSlicer *SimpleSlicerConf
}

type SimpleSlicerConf struct {
	BlockSize int64
	QueueSize int64
}

type DataOwnerEncryptorConf struct {
	Type          string
	SoftEncryptor *SoftEncryptorConf
}

type SoftEncryptorConf struct {
	Password string
}

type DataOwnerChallenger struct {
	Type    string
	Pairing *ChallengerPairingConf
	Merkle  *ChallengerMerkleConf
}

type ChallengerPairingConf struct {
	MaxIndexNum int64
	Sk          string
	Pk          string
	Randu       string
	Randv       string
}

type ChallengerMerkleConf struct {
	LeveldbRoot string
	ShrinkSize  int64
	SegmentSize int64
}

type DataOwnerCopierConf struct {
	Type string
}
