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

package ecc

import (
	"crypto/elliptic"
	"encoding/json"
	"errors"
	"math/big"
)

// Point 定义椭圆曲线上的点
type Point struct {
	Curve elliptic.Curve
	X     *big.Int
	Y     *big.Int
}

type ECPoint struct {
	CurveName string
	X, Y      *big.Int
}

// NewPoint 根据坐标和曲线类型构造Point
func NewPoint(curve elliptic.Curve, x, y *big.Int) (*Point, error) {
	if !curve.IsOnCurve(x, y) {
		return nil, errors.New("NewPoint: the given point is not on the elliptic curve")
	}
	return &Point{Curve: curve, X: x, Y: y}, nil
}

func newPoint(curve elliptic.Curve, x, y *big.Int) *Point {
	return &Point{Curve: curve, X: x, Y: y}
}

// ToString 将Point转换为string类型
func (p *Point) ToString() (string, error) {
	// 转换为自定义的数据结构
	point := getECPoint(p)

	// 转换json
	data, err := json.Marshal(point)

	return string(data), err
}

func getECPoint(p *Point) *ECPoint {
	point := new(ECPoint)
	point.CurveName = p.Curve.Params().Name
	point.X = p.X
	point.Y = p.Y

	return point
}

// Add 计算两个点的加法
func (p *Point) Add(p1 *Point) (*Point, error) {
	x, y := p.Curve.Add(p.X, p.Y, p1.X, p1.Y)

	return NewPoint(p.Curve, x, y)
}

// ScalarMult 计算一个点的数乘
func (p *Point) ScalarMult(k *big.Int) *Point {
	x, y := p.Curve.ScalarMult(p.X, p.Y, k.Bytes())
	newP := newPoint(p.Curve, x, y)

	return newP
}

// Equals 判断两个点是否相同
func (p *Point) Equals(p1 *Point) bool {
	if p == nil || p1 == nil {
		return false
	}

	if p.X.Cmp(p1.X) != 0 || p.Y.Cmp(p1.Y) != 0 {
		return false
	}

	return true
}
