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

package paillier

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestPaillier(t *testing.T) {

	// 密钥生成
	paillierPrivateKey, err := GeneratePrivateKey(DefaultPrimeLength)
	if err != nil {
		t.Errorf("GeneratePaillierPrivateKey failed: %v", err)
		return
	}

	jsonPaillierPrivateKey, _ := json.Marshal(paillierPrivateKey)
	t.Logf("paillierPrivateKey: %s", jsonPaillierPrivateKey)

	paillierPublicKey := paillierPrivateKey.PublicKey

	// 同态测试

	// 加密数据
	x2, err := paillierPublicKey.Encrypt(big.NewInt(2))
	if err != nil {
		t.Errorf("Paillier Encrypt failed: %v", err)
		return
	}
	t.Logf("Paillier Encrypt result[2]: %d", x2)

	x3, err := paillierPublicKey.Encrypt(big.NewInt(3))
	if err != nil {
		t.Errorf("Paillier Encrypt failed: %v", err)
		return
	}
	x5, err := paillierPublicKey.Encrypt(big.NewInt(5))
	if err != nil {
		t.Errorf("Paillier Encrypt failed: %v", err)
		return
	}

	// 测试编码，加密负数
	// when m is negative, use E(m mod(n)) instead of E(m)
	minus6 := new(big.Int).Mod(big.NewInt(-6), paillierPublicKey.N)
	xMinus6, err := paillierPublicKey.Encrypt(minus6)
	if err != nil {
		t.Errorf("Paillier Encrypt failed: %v", err)
		return
	}

	// 测试解码，解密原始负数
	// Decryption is modified to D′(c)=[D(c)]n with by definition [x]n = ((x+(n/2))mod(n) − (n/2).
	cMinus6 := paillierPrivateKey.Decrypt(xMinus6)
	tmpN := new(big.Int).Div(paillierPublicKey.N, big.NewInt(2))
	tmp6 := new(big.Int).Add(cMinus6, tmpN)
	tmp6 = new(big.Int).Mod(tmp6, paillierPublicKey.N)
	minusN := new(big.Int).Mul(tmpN, big.NewInt(-1))
	pMinus6 := new(big.Int).Add(tmp6, minusN)
	t.Logf("paillier math operation[Decrypt] result should be -6, and result is: %d", pMinus6)

	// 密文同态运算

	// 负数同态运算
	// 密文明文乘法
	cMinus24 := paillierPublicKey.CypherPlainMultiply(xMinus6, big.NewInt(4))
	p24 := paillierPrivateKey.Decrypt(cMinus24)
	tmp24 := new(big.Int).Add(p24, tmpN)
	tmp24 = new(big.Int).Mod(tmp24, paillierPublicKey.N)
	pMinus24 := new(big.Int).Add(tmp24, minusN)
	t.Logf("paillier math operation[negative number CypherPlainMultiply] result should be -24, and result is: %d", pMinus24)

	// 正数同态运算
	// 密文明文乘法
	c8 := paillierPublicKey.CypherPlainMultiply(x2, big.NewInt(4))
	p8 := paillierPrivateKey.Decrypt(c8)
	t.Logf("paillier math operation[CypherPlainMultiply] result should be 8, and result is: %d", p8)

	// 密文明文加法
	c9 := paillierPublicKey.CypherPlainAdd(x3, big.NewInt(6))
	p9 := paillierPrivateKey.Decrypt(c9)
	t.Logf("paillier math operation[CypherPlainAdd] result should be 9, and result is: %d", p9)

	// 密文明文乘法
	c40 := paillierPublicKey.CypherPlainMultiply(x5, big.NewInt(8))
	p40 := paillierPrivateKey.Decrypt(c40)
	t.Logf("paillier math operation[CypherPlainMultiply] result should be 40, and result is: %d", p40)

	// 密文加法
	c57 := paillierPublicKey.CyphersAdd(c8, c9, c40)
	p57 := paillierPrivateKey.Decrypt(c57)
	t.Logf("paillier math operation[CyphersAdd] result should be 57, and result is: %d", p57)
}
