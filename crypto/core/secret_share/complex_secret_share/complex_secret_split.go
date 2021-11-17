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
	"errors"
	"math/big"

	polynomial "github.com/PaddlePaddle/PaddleDTX/crypto/common/math/big_polynomial"
	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/ecc"
)

var (
	InvalidTotalShareNumberError = errors.New("totalShareNumber must be greater than one")
	InvalidShareNumberError      = errors.New("minimumShareNumber must be smaller than the totalShareNumber")
)

// Shamir's Secret Sharing algorithm, can be considered as:
// A way to split a secret to W shares, the secret can only be retrieved if more than T(T <= W) shares are combined together.
//
// This is the split process:
// 1. Encode the secret to a number S
// 2. Choose a lot of random numbers as coefficients, in order to make a random polynomials F(x) of degree T-1,
//		 the variable is X, the const(x-intercept) is S
// 3. For this polynomial, Give x different values, for example, x++ each time, then compute y = F(x)
// 4. So we get W shares, which are (x, y) pairs
// 5. Now encode each pair to a byte slice
func ComplexSecretSplit(totalShareNumber, minimumShareNumber int, secret []byte, curve elliptic.Curve) (shares map[int]*big.Int, err error) {
	poly, err := ComplexSecretToPolynomial(totalShareNumber, minimumShareNumber, secret, curve)
	if err != nil {
		return nil, err
	}

	polynomialClient := polynomial.New(curve.Params().N)

	// evaluate the polynomial several times to get all shares
	shares = make(map[int]*big.Int, totalShareNumber)
	for x := 1; x <= totalShareNumber; x++ {
		shares[x] = polynomialClient.Evaluate(poly, big.NewInt(int64(x)))
	}

	return shares, nil
}

// ComplexSecretSplitWithVerifyPoints 生成带验证点的秘密碎片
func ComplexSecretSplitWithVerifyPoints(totalShareNumber, minimumShareNumber int, secret []byte, curve elliptic.Curve) (shares map[int]*big.Int, points []*ecc.Point, err error) {
	poly, err := ComplexSecretToPolynomial(totalShareNumber, minimumShareNumber, secret, curve)
	if err != nil {
		return nil, nil, err
	}

	for _, coefficient := range poly {
		x, y := elliptic.P256().ScalarBaseMult(coefficient.Bytes())
		point, err := ecc.NewPoint(curve, x, y)
		if err != nil {
			return nil, nil, err
		}
		points = append(points, point)
	}

	polynomialClient := polynomial.New(curve.Params().N)

	// evaluate the polynomial several times to get all the shares
	shares = make(map[int]*big.Int, totalShareNumber)
	for x := 1; x <= totalShareNumber; x++ {
		shares[x] = polynomialClient.Evaluate(poly, big.NewInt(int64(x)))
	}
	return shares, points, nil
}

// ComplexSecretToPolynomial 根据指定的碎片数量和门限值，随机生成多项式
func ComplexSecretToPolynomial(totalShareNumber, minimumShareNumber int, secret []byte, curve elliptic.Curve) ([]*big.Int, error) {
	// Check the parameters
	if totalShareNumber < 2 {
		return nil, InvalidTotalShareNumberError
	}

	if minimumShareNumber > totalShareNumber {
		return nil, InvalidShareNumberError
	}

	polynomialClient := polynomial.New(curve.Params().N)

	poly, err := polynomialClient.RandomGenerate(minimumShareNumber-1, secret)
	if err != nil {
		return nil, err
	}

	return poly, nil
}

// GetVerifyPointByPolynomial 为产生本地秘密的私钥碎片做准备，通过目标多项式生成验证点
func GetVerifyPointByPolynomial(poly []*big.Int, curve elliptic.Curve) (*ecc.Point, error) {
	x, y := elliptic.P256().ScalarBaseMult(poly[0].Bytes())
	point, err := ecc.NewPoint(curve, x, y)
	if err != nil {
		return nil, err
	}

	return point, nil
}

// GetSpecifiedSecretShareByPolynomial 为产生本地秘密的私钥碎片做准备，通过目标多项式和节点index生成对应的碎片
func GetSpecifiedSecretShareByPolynomial(poly []*big.Int, index *big.Int, curve elliptic.Curve) *big.Int {
	polynomialClient := polynomial.New(curve.Params().N)

	// evaluate the polynomial with specified index to get shares
	share := polynomialClient.Evaluate(poly, index)

	return share
}
