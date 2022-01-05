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

package pdp

import (
	"fmt"
	"math/big"

	bls12_381_ecc "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

// GenerateChallenge generate a random challenge using index numbers
// challenge = {index}, {vi}
func GenerateChallenge(indexList []int) ([]*big.Int, []*big.Int, error) {
	var indices []*big.Int
	var randomVs []*big.Int
	for _, index := range indexList {
		indices = append(indices, new(big.Int).SetInt64(int64(index)))
		v, err := RandomWithinOrder()
		if err != nil {
			return nil, nil, err
		}
		randomVs = append(randomVs, v)
	}
	return indices, randomVs, nil
}

// Prove generate a proof by challenge
// sigma = v1*sigma1 + ... + vc*sigmac
// mu = (v1*m1 + ... + vc*mc) * g1
// proof = {sigma, mu}
func Prove(param ProofParams) (*bls12_381_ecc.G1Affine, *bls12_381_ecc.G1Affine, error) {
	if len(param.Indices) != len(param.RandomVs) || len(param.Content) != len(param.Indices) || len(param.Indices) == 0 {
		return nil, nil, fmt.Errorf("invalid challenge: %v", param)
	}

	sigma := new(bls12_381_ecc.G1Affine)
	vm := new(big.Int)
	for i := 0; i < len(param.Indices); i++ {
		// 1. vi*sigma_i
		vs := new(bls12_381_ecc.G1Affine).ScalarMultiplication(param.Sigmas[i], param.RandomVs[i])
		if i == 0 {
			sigma = vs
		} else {
			sigma = new(bls12_381_ecc.G1Affine).Add(sigma, vs)
		}

		// convert file content to int
		miInt := new(big.Int).SetBytes(param.Content[i])
		miInt = new(big.Int).Mod(miInt, order)

		// 2. vi*mi
		vmi := new(big.Int).Mul(param.RandomVs[i], miInt)
		vmi = new(big.Int).Mod(vmi, order)
		vm = new(big.Int).Add(vm, vmi)
		vm = new(big.Int).Mod(vm, order)
	}

	mu := new(bls12_381_ecc.G1Affine).ScalarMultiplication(&g1Gen, vm)

	return sigma, mu, nil
}
