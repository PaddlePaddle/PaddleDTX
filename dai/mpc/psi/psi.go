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

package psi

import (
	"crypto/ecdsa"
	"sync"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	csv "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common/csv"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
)

// VLPSI psi for vertical learning
// initialized at the beginning of training by Learner as well as prediction by Model
// see vl_common.psi for more
type VLPSI interface {
	// EncryptSampleIDSet to encrypt local IDs
	EncryptSampleIDSet() ([]byte, error)

	// SetReEncryptIDSet sets re-encrypted IDs from other party,
	// and tries to calculate final re-encrypted IDs
	// returns True if calculation is Done, otherwise False if still waiting for others' parts
	// returns Error if any mistake happens
	SetReEncryptIDSet(party string, reEncIDs []byte) (bool, error)

	// ReEncryptIDSet to encrypt encrypted IDs for other party
	ReEncryptIDSet(party string, encIDs []byte) ([]byte, error)

	// SetOtherReEncryptIDSet sets final re-encrypted IDs of other party
	SetOtherFinalReEncryptIDSet(party string, reEncIDs []byte) error

	// IntersectParts tries to calculate intersection with all parties samples
	// returns True with final result if calculation is Done, otherwise False if still waiting for others' samples
	// returns Error if any mistake happens
	// You'd better call it when SetReEncryptIDSet returns Done or SetOtherFinalReEncryptIDSet finishes
	IntersectParts() (bool, [][]string, []string, error)
}

// vlTwoPartsPsi implements VLPSI
type vlTwoPartsPsi struct {
	name          string
	privkey       *ecdsa.PrivateKey // local ecc private key for ID encryption
	samplesFile   []byte            // csv file content subjected to specified form
	samplesIdName string            // feature name for samples ID, used to extract IDs
	parties       map[string]bool   // names of other parties who participate MPC

	// intermediate results
	// see vl_common.psi for more
	ids                          []string
	rows                         [][]string
	encIDs                       []byte
	finalReEncIDs                []byte
	reEncryptIDSetsFromOthers    sync.Map // stores re-encrypted ID-Sets returned by other parties
	finalReEncryptIDSetsOfOthers sync.Map // stores final re-encrypted ID-Sets for other parties

	// final results
	done      bool
	newRows   [][]string
	intersect []string
}

// EncryptSampleIDSet encrypt sample ID list using own public key
func (vp *vlTwoPartsPsi) EncryptSampleIDSet() ([]byte, error) {
	err := vp.readSamples()
	if err != nil {
		return []byte{}, errorx.New(errcodes.ErrCodePSISamplesFile, "mistake[%s] happened when PSI read IDs from file", err.Error())
	}

	encIDs, err := vl_common.EncryptSampleIDSet(vp.ids, &vp.privkey.PublicKey)
	if err != nil {
		return []byte{}, errorx.New(errcodes.ErrCodePSIEncryptSampleIDSet, "mistake[%s] happened when PSI encrypt SampleIDSet", err.Error())
	}

	vp.encIDs = encIDs

	return vp.encIDs, nil
}

func (vp *vlTwoPartsPsi) SetReEncryptIDSet(party string, reEncIDs []byte) (bool, error) {
	if _, ok := vp.parties[party]; !ok {
		// if from unknown party, ignore
		return false, nil
	}

	vp.reEncryptIDSetsFromOthers.LoadOrStore(party, reEncIDs)
	vp.finalReEncIDs = reEncIDs

	return true, nil
}

// ReEncryptIDSet re-encrypt ID list for other party using own private key
func (vp *vlTwoPartsPsi) ReEncryptIDSet(party string, encIDs []byte) ([]byte, error) {

	reEncIDs, errRe := vl_common.ReEncryptIDSet(encIDs, vp.privkey)

	// if from unknown party, don't care about any Error
	if _, ok := vp.parties[party]; !ok {
		if errRe != nil {
			reEncIDs = []byte{}
		}
		return reEncIDs, nil
	}

	if errRe != nil {
		return []byte{}, errorx.New(errcodes.ErrCodePSIReEncryptIDSet, "mistake[%s] happened when PSI encrypt EncryptedSampleIDSet for other party[%s]", errRe.Error(), party)
	}

	return reEncIDs, nil
}

func (vp *vlTwoPartsPsi) SetOtherFinalReEncryptIDSet(party string, reEncIDs []byte) error {
	// if from unknown party, don't store result
	if _, ok := vp.parties[party]; !ok {
		return nil
	}
	vp.finalReEncryptIDSetsOfOthers.LoadOrStore(party, reEncIDs)
	return nil
}

// IntersectParts calculate intersections of all parties samples, and re-arrange sample files
func (vp *vlTwoPartsPsi) IntersectParts() (bool, [][]string, []string, error) {
	if vp.done {
		return vp.done, vp.newRows, vp.intersect, nil
	}

	var newRows [][]string

	for party := range vp.parties {
		_, ok := vp.reEncryptIDSetsFromOthers.Load(party)
		if !ok {
			return false, newRows, nil, nil
		}
	}

	var finalReEncIDsOfOther []byte
	for party := range vp.parties {
		v, ok := vp.finalReEncryptIDSetsOfOthers.Load(party)
		if !ok {
			return false, newRows, nil, nil
		}
		finalReEncIDsOfOther = v.([]byte)
		break
	}

	intersect, err := vl_common.IntersectTwoParts(vp.ids, vp.finalReEncIDs, finalReEncIDsOfOther)
	if err != nil {
		return false, newRows, intersect, errorx.New(errcodes.ErrCodePSIIntersectParts, "mistake[%s] happened when PSI intersect all parts", err.Error())
	}

	newRows, err = vl_common.RearrangeFileWithIntersectIDs(vp.rows, vp.samplesIdName, intersect)
	if err != nil {
		return false, newRows, intersect, errorx.New(errcodes.ErrCodePSIRearrangeFile, "mistake[%s] happened when PSI rearrange file with intersected IDs", err.Error())
	}

	vp.newRows = newRows
	vp.intersect = intersect
	vp.done = true

	return vp.done, vp.newRows, intersect, nil
}

// readSamples retrieve ID list from sample file rows
func (vp *vlTwoPartsPsi) readSamples() error {
	rows, IDs, err := csv.ReadIDsFromFileRows(vp.samplesFile, vp.samplesIdName)

	if err != nil {
		return err
	}

	vp.ids = IDs
	vp.rows = rows

	return nil
}

// NewVLTowPartsPSI create a VLPSI instance and initiate it
// name is to name the PSI instance
// parties are names of other parties who participate MPC
// sampleFile is csv file content subjected to specified form
// sampleIdName is used to extract IDs
func NewVLTowPartsPSI(name string, samplesFile []byte, samplesIdName string, parties []string) (VLPSI, error) {
	if len(parties) <= 0 {
		return nil, errorx.New(errcodes.ErrCodeParam, "no parties in PSI")
	}

	p := &vlTwoPartsPsi{
		name:          name,
		samplesFile:   samplesFile,
		samplesIdName: samplesIdName,
	}

	p.parties = map[string]bool{parties[0]: true}

	// create local ecc private key and public key pari for ID encryption
	privkey, err := vl_common.GeneratePSIKeyPair()
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when PSI GeneratePSIKeyPair", err.Error())
	}

	p.privkey = privkey

	return p, nil
}
