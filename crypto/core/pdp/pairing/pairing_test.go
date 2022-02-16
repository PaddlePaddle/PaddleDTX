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

package pairing

import (
	"crypto/rand"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	bls12_381_ecc "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

var fileNames = []string{"./testdata0", "./testdata1", "./testdata2"}

func TestPairing(t *testing.T) {
	// create random test files
	createFiles(t, fileNames)
	defer removeFiles(fileNames)

	indexList := []int{0, 1, 2}
	challengeRound := int64(100)

	// 1. generate key pair
	sk, pk, err := GenKeyPair()
	if err != nil {
		t.Errorf("failed to generate random client keypair, err: %v", err)
	}

	// 2. get random U and V
	randomU, err := RandomWithinOrder()
	if err != nil {
		t.Errorf("failed to generate random U, err: %v", err)
	}
	randomV, err := RandomWithinOrder()
	if err != nil {
		t.Errorf("failed to generate random V, err: %v", err)
	}

	// 3. calculate sigmas
	var sigmas []*bls12_381_ecc.G1Affine
	for idx, fileName := range fileNames {
		index := new(big.Int).SetInt64(int64(idx))
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			t.Errorf("failed to read file %s, err: %v", fileName, err)
		}

		param := CalculateSigmaIParams{
			Content: content,
			Index:   index,
			RandomV: randomV,
			RandomU: randomU,
			Privkey: sk,
			Round:   challengeRound,
		}
		sigma, err := CalculateSigmaI(param)
		if err != nil {
			t.Errorf("failed to calculate sigma%d, err: %v", idx, err)
		}
		sigmas = append(sigmas, sigma)
	}

	// 4. generate challenge
	indices, vs, randSeed, err := GenerateChallenge(indexList, challengeRound, sk)
	if err != nil {
		t.Errorf("failed to generate challenge, err: %v", err)
	}

	// 5. calculate proof
	var contents [][]byte
	for _, fileName := range fileNames {
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			t.Errorf("failed to read file %s, err: %v", fileName, err)
		}
		contents = append(contents, content)
	}

	proveParam := ProofParams{
		Content:       contents,
		Indices:       indices,
		RandomVs:      vs,
		Sigmas:        sigmas,
		RandThisRound: randSeed,
	}
	sigma, mu, err := Prove(proveParam)
	if err != nil {
		t.Errorf("failed to generate proof, err: %v", err)
	}

	// 6. verify proof
	verifyParam := VerifyParams{
		Sigma:    sigma,
		Mu:       mu,
		RandomV:  randomV,
		RandomU:  randomU,
		Indices:  indices,
		RandomVs: vs,
		Pubkey:   pk,
	}
	v, err := Verify(verifyParam)
	if err != nil {
		t.Errorf("failed to verify, err: %v", err)
	}

	if !v {
		t.Errorf("verification failed!")
	} else {
		t.Log("verification passed!")
	}
}

func createFiles(t *testing.T, fileNames []string) {
	for _, fileName := range fileNames {
		data := make([]byte, 102400)
		if _, err := io.ReadFull(rand.Reader, data); err != nil {
			t.Errorf("failed to read random bytes: %v", err)
		}
		err := ioutil.WriteFile(fileName, data, 0666)
		if err != nil {
			t.Errorf("failed to write to file %s, err: %v", fileName, err)
		}
	}
}

func removeFiles(fileNames []string) {
	for _, fileName := range fileNames {
		os.Remove(fileName)
	}
}
