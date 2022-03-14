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

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
)

// GenerateChallenge generate a random challenge using index numbers for a specified round
// challenge = {index}, {vi}, randNum
func GenerateChallenge(indexList []int, round int64, privkey *PrivateKey) ([]*big.Int, []*big.Int, []byte, error) {
	var indices []*big.Int
	var randomVs []*big.Int
	for _, index := range indexList {
		indices = append(indices, new(big.Int).SetInt64(int64(index)))
		v, err := RandomWithinOrder()
		if err != nil {
			return nil, nil, nil, err
		}
		randomVs = append(randomVs, v)
	}
	randThisRound := genRandNumByRound(round, privkey.X)
	return indices, randomVs, randThisRound, nil
}

// Prove generate a proof by challenge material
// sigma = v1*sigma1 + ... + vc*sigmac
// mu = (v1*SHA(m1||r_j) + ... + vc*SHA256(mc||r_j)) * g1
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

		// 2. SHA256(mi||r_j)
		hashMi := hash.HashUsingSha256(append(param.Content[i], param.RandThisRound...))
		hashMiInt := new(big.Int).SetBytes(hashMi)
		hashMiInt = hashMiInt.Mod(hashMiInt, order)

		// 3. vi*SHA256(mi||r_j)
		vmi := new(big.Int).Mul(param.RandomVs[i], hashMiInt)
		vmi = new(big.Int).Mod(vmi, order)
		vm = new(big.Int).Add(vm, vmi)
		vm = new(big.Int).Mod(vm, order)
	}

	mu := new(bls12_381_ecc.G1Affine).ScalarMultiplication(&g1Gen, vm)

	return sigma, mu, nil
}
