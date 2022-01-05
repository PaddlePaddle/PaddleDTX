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
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	bls12_381_ecc "github.com/consensys/gnark-crypto/ecc/bls12-381"
	bls12_381_fr "github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

func init() {
	_, _, g1Gen, g2Gen = bls12_381_ecc.Generators()
	order = bls12_381_fr.Modulus()
}

// GenRandomKeyPair generate a random private/public key pair for client
func GenRandomKeyPair() (*PrivateKey, *PublicKey, error) {
	sk, err := RandomWithinOrder()
	if err != nil {
		return nil, nil, err
	}

	pk := new(bls12_381_ecc.G2Affine).ScalarMultiplication(&g2Gen, sk)

	privkey := &PrivateKey{
		X: sk,
	}
	pubkey := &PublicKey{
		P: pk,
	}
	return privkey, pubkey, nil
}

// hashToG1 define a hash function from big int to point in G1
func hashToG1(data *big.Int) *bls12_381_ecc.G1Affine {
	hash := sha256.Sum256(data.Bytes())
	scalar := new(big.Int).SetBytes(hash[:])
	return new(bls12_381_ecc.G1Affine).ScalarMultiplication(&g1Gen, scalar)
}

// RandomWithinOrder generate a random number smaller than the order of G1/G2
func RandomWithinOrder() (*big.Int, error) {
	return rand.Int(rand.Reader, order)
}

// concatBigInt concat big integers and mod, i.e., (a||b||c..) mod m
func concatBigInt(list []*big.Int, modulus *big.Int) (*big.Int, error) {
	ret := new(big.Int)
	for _, n := range list {
		s := ret.String() + n.String()
		concatN, v := new(big.Int).SetString(s, 10)
		if !v {
			return nil, fmt.Errorf("failed to retrieve big int from string: %s", s)
		}
		ret = new(big.Int).Mod(concatN, modulus)
	}
	return ret, nil
}

// CalculateSigmaI calculate sigma_i using each segment and private key
// sigma_i = sk * ( H(v||i) + mi*u*g1 )
func CalculateSigmaI(param CalculateSigmaIParams) (*bls12_381_ecc.G1Affine, error) {
	// 1. H(v||i)
	vi, err := concatBigInt([]*big.Int{param.RandomV, param.Index}, order)
	if err != nil {
		return nil, err
	}
	hvi := hashToG1(vi)

	// 2. mi mod order
	miInt := new(big.Int).SetBytes(param.Content)
	miInt = new(big.Int).Mod(miInt, order)

	// 3. mi*u*g1
	mig1 := new(bls12_381_ecc.G1Affine).ScalarMultiplication(&g1Gen, miInt)
	miug1 := new(bls12_381_ecc.G1Affine).ScalarMultiplication(mig1, param.RandomU)

	// 4. H(v||i) + mi*u*g1
	add := new(bls12_381_ecc.G1Affine).Add(hvi, miug1)

	// 5. sk * (H(v||i) + mi*u*g1)
	sigmaI := new(bls12_381_ecc.G1Affine).ScalarMultiplication(add, param.Privkey.X)
	return sigmaI, nil
}
