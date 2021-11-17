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
	"errors"
	"log"
	"math/big"

	cryptoRand "crypto/rand"
)

// 加法半同态算法 - paillier, 用于纵向联合学习
// 原理步骤参见: https://en.wikipedia.org/wiki/Paillier_cryptosystem

var (
	DefaultPrimeLength = 512
)

var (
	// 质数p等于q
	ErrPrimePEqualsQ = errors.New("prime P should not equal Q")
	ErrMsgOutOfRange = errors.New("msg to be encrypted must within [0,N)")
)

// PrivateKey 同态加解密私钥
type PrivateKey struct {
	PublicKey
	Lambda *big.Int // λ
	Mu     *big.Int // μ
}

// PublicKey 同态加解密公钥
type PublicKey struct {
	N *big.Int
	G *big.Int
}

// GeneratePrivateKey 生成同态加密公私钥
// 通过指定比特长度来保证产生的质数p和q的长度相同，以便采取快速方法来构造公钥和私钥
func GeneratePrivateKey(primeLength int) (*PrivateKey, error) {
	// 首先生成2个大质数p和q, p!=q
	// Choose two large prime numbers p and q randomly and independently of each other,
	// such that gcd(pq,(p-1)(q-1))=1.
	// Note: This property is assured if both primes are of equal length.
	var p, q *big.Int
	var errChanFindP = make(chan error, 1)

	// 启动协程寻找p
	go func() {
		var err error
		p, err = cryptoRand.Prime(cryptoRand.Reader, primeLength)
		errChanFindP <- err
	}()

	// 寻找q
	q, errFindQ := cryptoRand.Prime(cryptoRand.Reader, primeLength)
	if errFindQ != nil {
		return nil, errFindQ
	}

	errFindP := <-errChanFindP
	if errFindP != nil {
		return nil, errFindP
	}

	// p 不能等于 q
	if p.Cmp(q) == 0 {
		return nil, ErrPrimePEqualsQ
	}

	// 注意：因为我们保证了质数p和q的长度是一样的，所以，可以采取快速方法来构造公钥和私钥

	// n = p*q
	n := new(big.Int).Mul(p, q)

	// g = n+1
	g := new(big.Int).Add(n, big.NewInt(1))

	// 加密公钥
	publicKey := &PublicKey{
		N: n,
		G: g,
	}

	// λ = lcm(p-1)(q-1), LCM(Least Common Multiple) 表示最小公倍数
	// 最小公倍数 = 两数之积/最大公约数
	// lambda := lcm(p-1, q-1)
	pMinus1 := new(big.Int).Add(p, big.NewInt(-1))
	qMinus1 := new(big.Int).Add(q, big.NewInt(-1))
	lambda := new(big.Int).Mul(pMinus1, qMinus1)

	// 计算μ，也就是λ在群n中的乘法逆元
	mu := new(big.Int).ModInverse(lambda, n)

	// 解密私钥
	privateKey := &PrivateKey{
		PublicKey: *publicKey,
		Lambda:    lambda,
		Mu:        mu,
	}

	return privateKey, nil
}

// Encrypt 加密正数
// 1. Let m be a message to be encrypted where 0<=m<n
// 2. Select a random number r where 0<r<n and ensure gcd(r,n)=1
// 3. Compute ciphertext as: c = g^m * r^n mod(n^2)
// 符号表示：E(m1,r1)
func (publicKey *PublicKey) Encrypt(m *big.Int) (*big.Int, error) {
	// generate a random r where 0<r<n and ensure gcd(r,n)=1
	r := big.NewInt(0)
	var errForR error

	for {
		// generate a random number r where 0<=r<n
		r, errForR = cryptoRand.Int(cryptoRand.Reader, publicKey.N)
		if errForR != nil {
			return nil, errForR
		}

		// ensure r!=0 and gcd(r,n)=1
		if big.NewInt(0).Cmp(r) != 0 && big.NewInt(1).Cmp(new(big.Int).GCD(nil, nil, r, publicKey.N)) == 0 {
			// 如果r符合条件，继续进行后续步骤的计算
			// if r matches the requirements, break and continue...
			break
		}
	}

	// Compute ciphertext as: c = g^m * r^n mod(n^2)
	// ensure 0<=m<n
	// if 0>m or !(m<n)
	if big.NewInt(0).Cmp(m) > 0 || m.Cmp(publicKey.N) != -1 {
		log.Printf("m is %d", m)

		checkResult := big.NewInt(0).Cmp(m)
		log.Printf("checkResult[big.NewInt(0).Cmp(m)] is %d", checkResult)

		checkResult = m.Cmp(publicKey.N)
		log.Printf("checkResult[m.Cmp(publicKey.N)] is %d", checkResult)

		return nil, ErrMsgOutOfRange
	}

	// 计算n^2, 也就是有限域的范围
	nSquare := new(big.Int).Mul(publicKey.N, publicKey.N)

	// 计算 gExpM = g^m mod(n^2)，mod后提升后续乘法性能
	gExpM := new(big.Int).Exp(publicKey.G, m, nSquare)

	// 计算 rExpN = r^N mod(n^2)，mod后提升后续乘法性能
	rExpN := new(big.Int).Exp(r, publicKey.N, nSquare)

	// 计算 ciphertext as: c = g^m * r^n mod(n^2)
	cypher := new(big.Int).Mod(new(big.Int).Mul(gExpM, rExpN), nSquare)

	return cypher, nil
}

// EncryptSupNegNum 加密负数
func (publicKey *PublicKey) EncryptSupNegNum(m *big.Int) (*big.Int, error) {
	// generate a random r where 0<r<n and ensure gcd(r,n)=1
	r := big.NewInt(0)
	var errForR error

	for {
		// generate a random number r where 0<=r<n
		r, errForR = cryptoRand.Int(cryptoRand.Reader, publicKey.N)
		if errForR != nil {
			return nil, errForR
		}

		// ensure r!=0 and gcd(r,n)=1
		if big.NewInt(0).Cmp(r) != 0 && big.NewInt(1).Cmp(new(big.Int).GCD(nil, nil, r, publicKey.N)) == 0 {
			// 如果r符合条件，继续进行后续步骤的计算
			// if r matches the requirements, break and continue...
			break
		}
	}

	// Compute ciphertext as: c = g^m * r^n mod(n^2)
	// ensure 0>m
	// if !(m<n)
	if m.Cmp(publicKey.N) != -1 {
		log.Printf("m is %d", m)
		log.Printf("N is %d", publicKey.N)

		checkResult := big.NewInt(0).Cmp(m)
		log.Printf("checkResult[m.Cmp(publicKey.N)] is %d", checkResult)

		return nil, ErrMsgOutOfRange
	}

	// 测试编码，加密负数
	// when m is negative, use E(m mod(n)) instead of E(m)
	m = new(big.Int).Mod(m, publicKey.N)

	// 计算n^2, 也就是有限域的范围
	nSquare := new(big.Int).Mul(publicKey.N, publicKey.N)

	// 计算 gExpM = g^m mod(n^2)，mod后提升后续乘法性能
	gExpM := new(big.Int).Exp(publicKey.G, m, nSquare)

	// 计算 rExpN = r^N mod(n^2)，mod后提升后续乘法性能
	rExpN := new(big.Int).Exp(r, publicKey.N, nSquare)

	// 计算 ciphertext as: c = g^m * r^n mod(n^2)
	cypher := new(big.Int).Mod(new(big.Int).Mul(gExpM, rExpN), nSquare)

	return cypher, nil
}

// CyphersAdd 纯密文加法
// The product of two ciphertexts will decrypt to the sum of their corresponding plaintexts
// D(E(m1,r1)*E(m2,r2) mod(n^2)) = m1+m2 mod(n)
func (pk *PublicKey) CyphersAdd(cyphers ...*big.Int) *big.Int {
	result := big.NewInt(1)

	// 计算n^2, 也就是有限域的范围
	nSquare := new(big.Int).Mul(pk.N, pk.N)

	// 原文的加法，其实等同于密文乘法后再解密
	for _, cypher := range cyphers {
		result = new(big.Int).Mod(new(big.Int).Mul(result, cypher), nSquare)
	}

	return result
}

// CypherPlainAdd 密文与原文的加法
// The product of a ciphertext with a plaintext raising g will decrypt to the sum of the corresponding plaintexts
// D(E(m1,r1)*g^m2 mod(n^2)) = m1+m2 mod(n)
func (pk *PublicKey) CypherPlainAdd(cypher, plain *big.Int) *big.Int {
	// 计算n^2, 也就是有限域的范围
	nSquare := new(big.Int).Mul(pk.N, pk.N)

	// 计算 gExpM = g^m2 mod(n^2)，mod后提升后续乘法性能
	gExpM := new(big.Int).Exp(pk.G, plain, nSquare)

	// 密文与原文的加法
	result := new(big.Int).Mod(new(big.Int).Mul(cypher, gExpM), nSquare)

	return result
}

// CypherPlainsAdd 密文与原文的加法
// The product of a ciphertext with a plaintext raising g will decrypt to the sum of the corresponding plaintexts
// D(E(m1,r1)*g^m2 mod(n^2)) = m1+m2 mod(n)
func (pk *PublicKey) CypherPlainsAdd(cypher *big.Int, plains ...*big.Int) *big.Int {
	result := cypher

	// 计算n^2, 也就是有限域的范围
	nSquare := new(big.Int).Mul(pk.N, pk.N)

	// 密文与原文的加法
	for _, plain := range plains {
		// 计算 gExpM = g^m2 mod(n^2)，mod后提升后续乘法性能
		gExpM := new(big.Int).Exp(pk.G, plain, nSquare)

		// 密文与原文的加法
		result = new(big.Int).Mod(new(big.Int).Mul(result, gExpM), nSquare)
	}

	return result
}

// CypherPlainMultiply 密文与原文的乘法
// An encrypted plaintext raised to a constant k will decrypt to the product of the plaintext and the constant
// D(E(m,r)^k mod(n^2) = k*m mod(n)
func (pk *PublicKey) CypherPlainMultiply(cypher, plain *big.Int) *big.Int {
	// 计算n^2, 也就是有限域的范围
	nSquare := new(big.Int).Mul(pk.N, pk.N)

	// 计算 E(m1,r1)^k mod(n^2)
	result := new(big.Int).Exp(cypher, plain, nSquare)

	return result
}

// Decrypt 解密数据 - 正数
// Compute the plaintext message as: m = L(c^λ mod(n^2)) * μ mod(n), L(x) = (x-1)/n
func (privateKey *PrivateKey) Decrypt(cypher *big.Int) *big.Int {
	// 计算n^2, 也就是有限域的范围
	nSquare := new(big.Int).Mul(privateKey.N, privateKey.N)

	// 计算 c^λ mod(n^2)
	cExpLambda := new(big.Int).Exp(cypher, privateKey.Lambda, nSquare)

	// 计算 L(c^λ mod(n^2)), L(x) = (x-1)/n
	// lx = (cExpLambda - 1) / n
	// 计算分子
	numerator := new(big.Int).Add(cExpLambda, big.NewInt(-1))
	// 分子除以分母，获得结果
	lx := new(big.Int).Div(numerator, privateKey.N)

	// 计算L(c^λ mod(n^2)) * μ mod(n)
	result := new(big.Int).Mod(new(big.Int).Mul(lx, privateKey.Mu), privateKey.N)

	return result
}

// DecryptSupNegNum 解密数据 - 负数
// Compute the plaintext message as: m = D(c) = L(c^λ mod(n^2)) * μ mod(n), L(x) = (x-1)/n
// Decryption is modified to D′(c)=[D(c)]n with by definition [x]n = ((x+(n/2))mod(n) − (n/2).
func (privateKey *PrivateKey) DecryptSupNegNum(cypher *big.Int) *big.Int {
	// 计算n^2, 也就是有限域的范围
	nSquare := new(big.Int).Mul(privateKey.N, privateKey.N)

	// 计算 c^λ mod(n^2)
	cExpLambda := new(big.Int).Exp(cypher, privateKey.Lambda, nSquare)

	// 计算 L(c^λ mod(n^2)), L(x) = (x-1)/n
	// lx = (cExpLambda - 1) / n
	// 计算分子
	numerator := new(big.Int).Add(cExpLambda, big.NewInt(-1))
	// 分子除以分母，获得结果
	lx := new(big.Int).Div(numerator, privateKey.N)

	// 计算L(c^λ mod(n^2)) * μ mod(n)
	result := new(big.Int).Mod(new(big.Int).Mul(lx, privateKey.Mu), privateKey.N)

	tmpN := new(big.Int).Div(privateKey.N, big.NewInt(2))
	result = new(big.Int).Add(result, tmpN)
	result = new(big.Int).Mod(result, privateKey.N)
	minusN := new(big.Int).Mul(tmpN, big.NewInt(-1))
	result = new(big.Int).Add(result, minusN)

	return result
}
