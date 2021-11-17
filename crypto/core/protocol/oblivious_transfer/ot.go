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

package oblivious_transfer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecies"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
)

// 1 of 2 不经意传输协议 - 基于CDH假设和椭圆曲线ECC
// 1 of 2 Oblivious Transfer Protocol - based on the CDH assumption
// 用于联邦学习或多方隐私计算等方案的机密数据传输
//
// TODO: 实现 1 of N Oblivious Transfer Protocol
//
// Computational Diffie-Hellman assumption(CDH assumption):
// An algorithm that solves the computational Diffie-Hellman problem
// is a probabilistic polynomial time Turing machine, on input g,g^x,g^y,
// outputs g^xy with non-negligible probability.
// Computational Diffie-Hellman assumption means that there is
// no such a probabilistic polynomial time Turing machine.
//
// CDH问题：
// 给出任意的g,g^x,g^y，求g^xy.
//
// CDH假设:
// 不存在一个概率多项式时间图灵机能够解决CDH问题。
//
// 本项目实现的1 of 2 OT协议，建立在CDH假设成立的基础上。椭圆曲线ECC也满足离散对数Discrete logarithm谜题。
//
// 1 of 2 不经意传输协议的定义：
// 数据接收方Bob想从数据发送方Alice处获取某份数据M(i)，但不想让Alice知道自己想要的是哪份数据。
// Bob通知Alice，想获取2份数据M(0),M(1)，其中自己的目标数据M(i)就在其中；
// 数据发送方Alice通过OT协议向数据接收方Bob发送全部的2份数据：M(0),M(1)；
// 数据接收方Bob仅能从这2份数据中选择一份数据进行查看（当然他会选择M(i)），但是Alice无法知晓Bob选择的是哪份数据。
//
// 1 of N 不经意传输协议的定义：
// 数据接收方Bob想从数据发送方Alice处获取某份数据M(i)，但不想让Alice知道自己想要的是哪份数据。
// Bob通知Alice，想获取N份数据M(0),M(1),M(2),...M(N)，其中自己的目标数据M(i)就在其中；
// 数据发送方Alice通过OT协议向数据接收方Bob发送全部的N份数据：M(0),M(1),M(2),...M(N)；
// 数据接收方Bob仅能从这N份数据中选择一份数据进行查看（当然他会选择M(i)），但是Alice无法知晓Bob选择的是哪份数据。
//
// 数据传输的步骤：
// 前提注意：
//	1. Alice和Bob在计算过程中使用同样的椭圆曲线；
//	2. Alice持有多份文件，Bob向Alice同时索取M(0)和M(1)，但是仅能获取M(0)或M(1)；
//	3. Alice无法感知到Bob究竟获取了M(0)还是M(1)。
//
//	HP is a special hash function which converts a publicKey to a curve point. HP(P) = Hash(P)*G
//
// Step 1：Alice产生1个公钥-私钥组合(Pub'A,Prv'A)，然后将公钥Pub'A发送给Bob。
// Step 2：Bob产生1个公钥-私钥组合(Pub'B,Prv'B)，接着选择需要哪份数据，也就是做1 of 2选择：
// 			2.1 选择M(0)，将公钥Pub'B发送给Alice。
// 			2.2 选择M(1)，将Pub'A+Pub'B作为公钥Pub'B发送给Alice。也就是说，Alice看到的Pub'B，其实是Pub'A+Pub'B
// Step 3：Alice计算Pub'0和Pub'1，
//			3.1 Alice使用Bob传来的Pub'B进行如下计算：Pub'0=HP(Pub'B^Prv'A)，Pub'1=HP((Pub'B-Pub'A)^Prv'A)
//			3.2 但是实际上，Alice是这么算的：
//				3.2.1 如果Bob选择的是M(0)，那么
//						Pub'0=HP(Pub'B^Prv'A)=Hash(Pub'B^Prv'A)*G,
//						Pub'1=HP((Pub'B-Pub'A)^Prv'A)=Hash((Pub'B-Pub'A)^Prv'A)*G
//				3.2.2 如果Bob选择的是M(1)，那么
//						Pub'0=HP((Pub'A+Pub'B)^Prv'A)=Hash((Pub'A+Pub'B)^Prv'A)*G,
//						Pub'1=HP((Pub'A+Pub'B-Pub'A)^Prv'A)=HP(Pub'B^Prv'A)=Hash(Pub'B^Prv'A)*G,
// Step 4：Alice用Pub'0加密M(0)得到s0,Pub'1加密M(1)得到s1。
// Step 5：Alice将s0和s1发给Bob。
// Step 6：Bob根据之前的选择结果，获取自己需要的数据：
//			5.1 如果之前选择的是M(0)，那么
//				5.1.1 Bob有能力计算出Prv'0，解密s0，从而得到M(0)。
//						Pub'B^Prv'A = (g^b)^a = (g^a)^b = Pub'A^Prv'B
//						Prv'0 = Hash(Pub'A^Prv'B)
//						M(0) = Dec(s0, Prv'0)
//				5.1.2 Bob没有能力解密s1
//			5.2 如果之前选择的是M(1)，那么
//				5.2.1 Bob有能力计算出Prv'1，解密s1，从而得到M(1)。
//						(Pub'A+Pub'B-Pub'A)^Prv'A = Pub'B^Prv'A = (g^b)^a = (g^a)^b = Pub'A^Prv'B
//						Prv'1 = Hash(Pub'A^Prv'B)
//						M(1) = Dec(s1, Prv'1)
//				5.2.2 Bob没有能力解密s0

// message index
const (
	IndexOne = iota
	IndexTwo
)

var (
	IndexError = errors.New("chosenIndex is invalid. Must be 0 or 1")
)

// ReceiverChoose 接收方Bob选择需要哪份数据，也就是做1 of 2选择
func ReceiverChoose(receiverPrivateKey *ecdsa.PrivateKey, senderPublicKey *ecdsa.PublicKey, chosenIndex int) (*ecdsa.PublicKey, error) {
	// 判断chooseIndex是否合法
	if chosenIndex != IndexOne && chosenIndex != IndexTwo {
		return nil, IndexError
	}

	// 如果选择M(0)，将公钥Pub'B发送给Alice。
	if chosenIndex == IndexOne {
		return &receiverPrivateKey.PublicKey, nil
	}

	// 如果选择M(1)，将Pub'A+Pub'B发送给Alice
	curve := receiverPrivateKey.Curve

	senderX := senderPublicKey.X
	senderY := senderPublicKey.Y

	newX, newY := curve.Add(senderX, senderY, receiverPrivateKey.PublicKey.X, receiverPrivateKey.PublicKey.Y)

	newPubKey := new(ecdsa.PublicKey)
	newPubKey.Curve = curve
	newPubKey.X = newX
	newPubKey.Y = newY

	return newPubKey, nil
}

// SenderEncryptMsg 发送方Alice根据接收方Bob发来的公钥，做进一步计算
func SenderEncryptMsg(senderPrivateKey *ecdsa.PrivateKey, receiverPublicKey *ecdsa.PublicKey, msgs []string) ([]string, error) {
	curve := senderPrivateKey.Curve

	receiverX := receiverPublicKey.X
	receiverY := receiverPublicKey.Y

	// step 1: 为M(0)计算s0
	// 加密密钥Pub'0 = Hash(Pub'B^Prv'A)*G

	// 计算Pub'B^Prv'A
	newX, newY := curve.ScalarMult(receiverX, receiverY, senderPrivateKey.D.Bytes())

	newPubKey := new(ecdsa.PublicKey)
	newPubKey.Curve = curve
	newPubKey.X = newX
	newPubKey.Y = newY

	// 获取新公钥的hash标志
	// 加密密钥Pub'0 = Hash(Pub'B^Prv'A)*G
	hashKey := getHashKey(newPubKey)

	// 加密消息M(0)
	ct0, err := ecies.Encrypt(hashKey, []byte(msgs[0]))
	if err != nil {
		return nil, err
	}

	// step 2: 为M(1)计算s1
	// 加密密钥Pub'1 = Hash((Pub'B-Pub'A)^Prv'A)*G

	// 1.3 计算-Pub'A，如果Pub'A = (x,y)，则 -Pub'A = (x, -y mod P)
	negativeOne := big.NewInt(-1)
	y2 := new(big.Int).Mod(new(big.Int).Mul(negativeOne, senderPrivateKey.PublicKey.Y), curve.Params().P)

	// 计算Pub'B-Pub'A
	x, y := curve.Add(receiverX, receiverY, senderPrivateKey.PublicKey.X, y2)

	// 计算(Pub'B-Pub'A)^Prv'A
	newX, newY = curve.ScalarMult(x, y, senderPrivateKey.D.Bytes())

	newPubKey.X = newX
	newPubKey.Y = newY

	// 获取新公钥的hash标志
	// 加密密钥Pub'0 = Hash((Pub'B-Pub'A)^Prv'A)*G
	hashKey = getHashKey(newPubKey)

	// 加密消息M(1)
	ct1, err := ecies.Encrypt(hashKey, []byte(msgs[1]))
	if err != nil {
		return nil, err
	}

	// 组装加密消息M(0)和M(1)
	var cts []string
	cts = append(cts, string(ct0))
	cts = append(cts, string(ct1))

	return cts, nil
}

// getHashKey 获取公钥的hash标志
func getHashKey(publicKey *ecdsa.PublicKey) *ecdsa.PublicKey {
	curve := publicKey.Curve

	// Hash(P)
	hashP := hash.HashUsingSha256(elliptic.Marshal(curve, publicKey.X, publicKey.Y))

	// Point(Hash(P)) = Hash(P) * G
	hashX, hashY := curve.ScalarBaseMult(hashP)

	hashKey := new(ecdsa.PublicKey)
	hashKey.Curve = curve
	hashKey.X = hashX
	hashKey.Y = hashY

	return hashKey
}

// ReceiverRetrieveMsg Bob根据之前的选择结果，解密并获取自己需要的数据
func ReceiverRetrieveMsg(receiverPrivateKey *ecdsa.PrivateKey, senderPublicKey *ecdsa.PublicKey, cts []string, chosenIndex int) (string, error) {
	// 判断chooseIndex是否合法
	if chosenIndex != IndexOne && chosenIndex != IndexTwo {
		return "", IndexError
	}

	// 计算Pub'A^Prv'B
	curve := receiverPrivateKey.Curve

	senderX := senderPublicKey.X
	senderY := senderPublicKey.Y

	// 计算Pub'B^Prv'A
	newX, newY := curve.ScalarMult(senderX, senderY, receiverPrivateKey.D.Bytes())

	// 解密密钥Prv'0 = Hash(Pub'B^Prv'A)
	hashP := hash.HashUsingSha256(elliptic.Marshal(curve, newX, newY))

	privateKey := new(ecdsa.PrivateKey)
	privateKey.Curve = curve
	privateKey.X, privateKey.Y = curve.ScalarBaseMult(hashP)
	privateKey.D = new(big.Int).SetBytes(hashP)

	// 如果之前选择的是M(0)
	if chosenIndex == IndexOne {
		msg, err := ecies.Decrypt(privateKey, []byte(cts[0]))
		if err != nil {
			return "", err
		}

		return string(msg), nil
	}

	// 如果之前选择的是M(1)
	msg, err := ecies.Decrypt(privateKey, []byte(cts[1]))
	if err != nil {
		return "", err
	}

	return string(msg), nil
}
