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
	"reflect"

	"github.com/cloudflare/bn256"
)

// Verify verify the proof
// e(sigma, g2) = e( (v1*H(v||index_1) + ... + vc*H(v||index_c)) + u*mu, pk)
func Verify(param VerifyParams) (bool, error) {
	g2 := new(bn256.G2).ScalarBaseMult(new(big.Int).SetInt64(1))
	left := bn256.Pair(param.Sigma, g2)

	vh := new(bn256.G1)
	for i := 0; i < len(param.Indices); i++ {
		vi, err := concatBigInt([]*big.Int{param.RandomV, param.Indices[i]}, bn256.Order)
		if err != nil {
			return false, fmt.Errorf("failed to concat %v and %v, err: %v", param.RandomV, param.Indices[i], err)
		}

		hi := hashToG1(vi)
		vhi := new(bn256.G1).ScalarMult(hi, param.RandomVs[i])
		if i == 0 {
			vh = vhi
		} else {
			vh = new(bn256.G1).Add(vh, vhi)
		}
	}
	umu := new(bn256.G1).ScalarMult(param.Mu, param.RandomU)
	add := new(bn256.G1).Add(vh, umu)
	right := bn256.Pair(add, param.Pubkey.P)

	return reflect.DeepEqual(left.Marshal(), right.Marshal()), nil
}
