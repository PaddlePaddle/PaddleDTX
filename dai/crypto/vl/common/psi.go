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

package common

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"sort"

	"github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	linear_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/linear_regression/gradient_descent/mpc_vertical"
)

var (
	xchainCryptoClient = new(xchain.XchainCryptoClient)
	defaultCurve       = elliptic.P256()
)

// GeneratePSIKeyPair generate ecc private and public key pair for PSI using default elliptic curve
// key pair is used for ID list encryption and intersection
func GeneratePSIKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(defaultCurve, rand.Reader)
}

// EncryptSampleIDSet encrypt local sample ID set by own public key
// IDSet is retrieved from local sample file, encrypted by local public key
func EncryptSampleIDSet(IDSet []string, publicKey *ecdsa.PublicKey) ([]byte, error) {
	encIDs := xchainCryptoClient.PSIEncryptSampleIDSet(IDSet, publicKey)
	return PSIEncSetToBytes(encIDs)
}

// ReEncryptIDSet re-encrypt others ID set by own private key
// encSet is the encryption of ID list, received from other party, already encrypted once
// encrypt encSet once more using local private key
func ReEncryptIDSet(encSet []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	set, err := PSIEncSetFromBytes(encSet)
	if err != nil {
		return nil, err
	}
	IDs := xchainCryptoClient.PSIReEncryptIDSet(set, privateKey)
	return PSIEncSetToBytes(IDs)
}

// IntersectTwoParts get intersection of two parts' ID set
// sampleID is ID list retrieved from local sample file
// reEncSetLocal is local ID list that was already encrypted twice
// reEncSetOthers is other party's ID list that was already encrypted twice
func IntersectTwoParts(sampleID []string, reEncSetLocal []byte, reEncSetOthers []byte) ([]string, error) {
	localSet, err := PSIEncSetFromBytes(reEncSetLocal)
	if err != nil {
		return nil, err
	}
	otherSet, err := PSIEncSetFromBytes(reEncSetOthers)
	if err != nil {
		return nil, err
	}
	otherSetList := []*linear_vertical.EncSet{otherSet}
	return xchainCryptoClient.PSIntersect(sampleID, localSet, otherSetList), nil
}

// RetrieveIDsFromFile retrieve ID set from file rows by id name
// fileRows is original sample rows, including feature list and sample values
// idName is the name of ID feature, like "id", "card_number"...
func RetrieveIDsFromFile(fileRows [][]string, idName string) ([]string, error) {
	if fileRows == nil {
		return nil, fmt.Errorf("empty file content")
	}

	// read the first row to get feature nums and ID feature index
	featureNum := len(fileRows[0])
	featureIndex := -1
	for i := 0; i < featureNum; i++ {
		if fileRows[0][i] == idName {
			featureIndex = i
		}
	}
	if featureIndex == -1 {
		return nil, fmt.Errorf("file does not contain sample id: %s", idName)
	}

	// read from all rows to get ID set
	var set []string
	for row := 1; row < len(fileRows); row++ {
		set = append(set, fileRows[row][featureIndex])
	}

	return set, nil
}

// RearrangeFileWithIntersectIDs re-arrange file by hash(ID) ascending order
// fileRows is original sample rows, including feature list and sample values
// idName is the name of ID feature
// IDs is the ID list after PSI
func RearrangeFileWithIntersectIDs(fileRows [][]string, idName string, IDs []string) ([][]string, error) {
	if fileRows == nil {
		return nil, fmt.Errorf("empty file content")
	}
	if len(IDs) == 0 {
		return nil, fmt.Errorf("empty ID list")
	}

	reOrderedIDs := reOrderIDSet(IDs)
	// first row is feature list, others are samples
	intersectRows := make([][]string, len(IDs)+1)

	featureNum := len(fileRows[0])
	featureIndex := -1
	for i := 0; i < featureNum; i++ {
		if fileRows[0][i] == idName {
			featureIndex = i
		}
	}
	if featureIndex == -1 {
		return nil, fmt.Errorf("file does not contain sample id: %s", idName)
	}
	var firstRow []string
	firstRow = append(firstRow, fileRows[0][0:featureIndex]...)
	firstRow = append(firstRow, fileRows[0][featureIndex+1:]...)
	intersectRows[0] = firstRow

	// read from all rows to remove ID and re-order rows
	for row := 1; row < len(fileRows); row++ {
		var newRow []string
		id := fileRows[row][featureIndex]

		if idx, exist := reOrderedIDs[id]; exist {
			newRow = append(newRow, fileRows[row][0:featureIndex]...)
			newRow = append(newRow, fileRows[row][featureIndex+1:]...)
			intersectRows[idx+1] = newRow
		}
	}

	return intersectRows, nil
}

// reOrderIDSet by ID string ascending order
func reOrderIDSet(IDs []string) map[string]int {
	idxMap := make(map[string]int)
	sort.Strings(IDs)
	for i := 0; i < len(IDs); i++ {
		idxMap[IDs[i]] = i
	}
	return idxMap
}
