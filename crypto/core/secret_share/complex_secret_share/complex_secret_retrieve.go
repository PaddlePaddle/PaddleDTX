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

package complex_secret_share

import (
	"crypto/elliptic"
	"math/big"

	polynomial "github.com/PaddlePaddle/PaddleDTX/crypto/common/math/big_polynomial"
)

// Shamir's Secret Sharing algorithm, can be considered as:
// A way to split a secret to W shares, the secret can only be retrieved if more than T(T <= W) shares are combined together.
//
// This is the retrieve process:
// 1. Decode each share i.e. the byte slice to a (x, y) pair
// 2. Use lagrange interpolation formula, take the (x, y) pairs as input points to compute a polynomial f(x)
//		 which is able to match all the given points.
// 3. Give x = 0, then the secret number S can be computed
// 4. Now decode number S, then the secret is retrieved
func ComplexSecretRetrieve(shares map[int]*big.Int, curve elliptic.Curve) ([]byte, error) {
	secretInt := lagrangeInterpolate(shares, curve)

	secret := secretInt.Bytes()

	return secret, nil
}

// lagrangeInterpolate Lagrange Polynomial Interpolation Formula
func lagrangeInterpolate(points map[int]*big.Int, curve elliptic.Curve) *big.Int {
	// 通过这些坐标点来恢复出多项式
	polynomialClient := polynomial.New(curve.Params().N)
	result := polynomialClient.GetPolynomialByPoints(points)

	// 秘密就是常数项
	secret := result[len(result)-1]

	return secret
}
