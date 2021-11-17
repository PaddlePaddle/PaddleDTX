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

package big_polynomial

import (
	"math/big"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/rand"
)

type PolynomialClient struct {
	// A big prime which is used for Galois Field computing
	prime *big.Int
}

// New new PolynomialClient with a prime
func New(prime *big.Int) *PolynomialClient {
	pc := new(PolynomialClient)
	pc.prime = prime

	return pc
}

// RandomGenerate make a random polynomials F(x) of Degree [degree], and the const(X-Intercept) is [intercept]
// 给定最高次方和x截距，生成一个系数随机的多项式
func (pc *PolynomialClient) RandomGenerate(degree int, secret []byte) ([]*big.Int, error) {
	// 字节数组转big int
	intercept := big.NewInt(0).SetBytes(secret)

	// 多项式参数格式是次方数+1（代表常数）
	result := make([]*big.Int, degree+1)

	// 多项式的常数项就是x截距
	// 多个bytes组成一个bigint，作为多项式的系数
	coefficientFactor := 32
	index := 0
	result[index] = intercept

	// 生成非最高次方位的随机参数
	if degree > 1 {
		randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, coefficientFactor*(degree-1))
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(randomBytes); i += coefficientFactor {
			byteSlice := randomBytes[i : i+coefficientFactor]
			result[index+1] = big.NewInt(0).SetBytes(byteSlice)
			index++
		}
	}

	// This coefficient can't be zero, otherwise it will be a polynomial,
	// the degree of which is [degree-1] other than [degree]
	// 生成最高次方位的随机参数，该值不能为0，否则最高次方会退化为次一级
	for {
		randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, coefficientFactor)
		if err != nil {
			return nil, err
		}

		highestDegreeCoefficient := big.NewInt(0).SetBytes(randomBytes)
		if highestDegreeCoefficient != big.NewInt(0) {
			result[degree] = highestDegreeCoefficient
			return result, nil
		}
	}
}

// Evaluate Given the specified value, get the compution result of the polynomial
// 给出指定x值，计算出指定多项式f(x)的值
func (pc *PolynomialClient) Evaluate(polynomialCoefficients []*big.Int, specifiedValue *big.Int) *big.Int {
	degree := len(polynomialCoefficients) - 1

	// 注意这里要用set，否则会出现上层业务逻辑的指针重复使用的问题
	result := big.NewInt(0).Set(polynomialCoefficients[degree])

	for i := degree - 1; i >= 0; i-- {
		result = result.Mul(result, specifiedValue)
		result = result.Add(result, polynomialCoefficients[i])
	}

	return result
}

// Add 对2个多项式进行加法操作
func (pc *PolynomialClient) Add(a []*big.Int, b []*big.Int) []*big.Int {
	degree := len(a)
	c := make([]*big.Int, degree)

	// 初始化big int数组
	for i := range c {
		c[i] = big.NewInt(0)
	}

	for i := 0; i < degree; i++ {
		c[i] = a[i].Add(a[i], b[i])
		c[i] = big.NewInt(0).Mod(c[i], pc.prime)
	}

	return c
}

// Multiply 对2个多项式进行乘法操作
func (pc *PolynomialClient) Multiply(a []*big.Int, b []*big.Int) []*big.Int {
	degA := len(a)
	degB := len(b)
	result := make([]*big.Int, degA+degB-1)

	// 初始化big int数组
	for i := range result {
		result[i] = big.NewInt(0)
	}

	for i := 0; i < degA; i++ {
		for j := 0; j < degB; j++ {
			temp := a[i].Mul(a[i], b[j])
			result[i+j] = result[i+j].Add(result[i+j], temp)
		}
	}

	return result
}

// Scale 将1个多项式与指定系数k进行乘法操作
func (pc *PolynomialClient) Scale(a []*big.Int, k *big.Int) []*big.Int {
	b := make([]*big.Int, len(a))

	for i := 0; i < len(a); i++ {
		b[i] = a[i].Mul(a[i], k)
		b[i] = big.NewInt(0).Mod(b[i], pc.prime)
	}

	return b
}

// GetLagrangeBasePolynomial 获取拉格朗日基本多项式（插值基函数）
func (pc *PolynomialClient) GetLagrangeBasePolynomial(xs []*big.Int, xpos int) []*big.Int {
	var poly []*big.Int
	poly = append(poly, big.NewInt(1))

	denominator := big.NewInt(1)

	for i := 0; i < len(xs); i++ {
		if i != xpos {
			currentTerm := make([]*big.Int, 2)
			currentTerm[0] = big.NewInt(1)
			currentTerm[1] = big.NewInt(0).Sub(big.NewInt(0), xs[i])
			denominator = denominator.Mul(denominator, big.NewInt(0).Sub(xs[xpos], xs[i]))
			poly = pc.Multiply(poly, currentTerm)
		}
	}

	inverser := big.NewInt(0).ModInverse(denominator, pc.prime)

	return pc.Scale(poly, inverser)
}

// GetPolynomialByPoints 利用Lagrange Polynomial Interpolation Formula，通过给定坐标点集合来计算多项式
func (pc *PolynomialClient) GetPolynomialByPoints(points map[int]*big.Int) []*big.Int {
	degree := len(points)
	bases := make([][]*big.Int, degree)
	result := make([]*big.Int, degree)

	for i := range result {
		result[i] = big.NewInt(0)
	}

	var xs []*big.Int
	var ys []*big.Int

	for k, v := range points {
		xs = append(xs, big.NewInt(int64(k)))
		ys = append(ys, v)
	}

	for i := 0; i < degree; i++ {
		bases[i] = pc.GetLagrangeBasePolynomial(xs, i)
	}

	for i := 0; i < degree; i++ {
		result = pc.Add(result, pc.Scale(bases[i], ys[i]))
	}

	return result
}
