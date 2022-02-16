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
	"fmt"
	"math/big"

	bls12_381_ecc "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

// Verify verify the proof
// e(sigma, g2) = e( (v1*H(v||index_1) + ... + vc*H(v||index_c)) + u*mu, pk)
func Verify(param VerifyParams) (bool, error) {
	left, err := bls12_381_ecc.Pair([]bls12_381_ecc.G1Affine{*param.Sigma}, []bls12_381_ecc.G2Affine{g2Gen})
	if err != nil {
		return false, err
	}

	vh := new(bls12_381_ecc.G1Affine)
	for i := 0; i < len(param.Indices); i++ {
		vi, err := concatBigInt([]*big.Int{param.RandomV, param.Indices[i]}, order)
		if err != nil {
			return false, fmt.Errorf("failed to concat %v and %v, err: %v", param.RandomV, param.Indices[i], err)
		}

		hi := hashToG1(vi)
		vhi := new(bls12_381_ecc.G1Affine).ScalarMultiplication(hi, param.RandomVs[i])
		if i == 0 {
			vh = vhi
		} else {
			vh = new(bls12_381_ecc.G1Affine).Add(vh, vhi)
		}
	}
	umu := new(bls12_381_ecc.G1Affine).ScalarMultiplication(param.Mu, param.RandomU)
	add := new(bls12_381_ecc.G1Affine).Add(vh, umu)

	right, err := bls12_381_ecc.Pair([]bls12_381_ecc.G1Affine{*add}, []bls12_381_ecc.G2Affine{*param.Pubkey.P})
	if err != nil {
		return false, err
	}

	return left.Equal(&right), nil
}
