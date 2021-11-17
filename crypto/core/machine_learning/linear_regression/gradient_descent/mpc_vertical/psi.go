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

package mpc_vertical

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
)

// 加密样本对齐 - 一种基于ECC和Diffie-Hellman方案的PSI协议
// 用于纵向联合学习

// 加密样本对齐的步骤(2方)：
// Step 1: 前提条件，多方的样本拥有可以被对齐的特征ID，例如身份证号/手机号等等
// Step 2: 参与方分别对自己的样本特征ID进行计算，通过如下方式获得每个样本特征ID的加密集合：
//			Alice: Pub'Ai=HP(ID-Ai)^Prv'A=Hash(ID-Ai)*G^Prv'A=Hash(ID-Ai)*Pub'A
//			  Bob: Pub'Bi=HP(ID-Bi)^Prv'B=Hash(ID-Bi)*G^Prv'B=Hash(ID-Bi)*Pub'B
// Step 3: 参与方交换自己的样本特征ID加密集合，每方都可以获得其它方的本特征ID加密集合
// Step 4: 参与方使用自己的私钥对其它方的样本特征ID加密集合进行二次加密，例如：
//			Alice: Pub'Bi-A=Pub'Bi^Prv'A=Hash(ID-Bi)*G^Prv'B^Prv'A
//			  Bob: Pub'Ai-B=Pub'Ai^Prv'B=Hash(ID-Ai)*G^Prv'A^Prv'B
// Step 5: 参与方交换经过二次加密的样本特征ID加密集合
//			Alice和Bob均获得：Pub'Bi-A和Pub'Ai-B
// Step 6: 参与方分别对比样本特征ID加密集合，获得数值相等的部分，这就是交集

// 加密样本对齐的步骤(3方)：
// Step 1: 前提条件，多方的样本拥有可以被对齐的特征ID，例如身份证号/手机号等等
// Step 2: 参与方分别对自己的样本特征ID进行计算，通过如下方式获得每个样本特征ID的加密集合：
//			Alice: Pub'Ai=HP(ID-Ai)^Prv'A=Hash(ID-Ai)*G^Prv'A=Hash(ID-Ai)*Pub'A
//			  Bob: Pub'Bi=HP(ID-Bi)^Prv'B=Hash(ID-Bi)*G^Prv'B=Hash(ID-Bi)*Pub'B
//			Carol: Pub'Ci=HP(ID-Ci)^Prv'C=Hash(ID-Ci)*G^Prv'C=Hash(ID-Ci)*Pub'C
// Step 3: 参与方交换自己的样本特征ID加密集合，每方都可以获得其它方的本特征ID加密集合
// Step 4: 参与方使用自己的私钥对其它方的样本特征ID加密集合进行二次加密，例如：
//			Alice: Pub'Bi-A=Pub'Bi^Prv'A=Hash(ID-Bi)*G^Prv'B^Prv'A
//			  Bob: Pub'Ci-B=Pub'Ci^Prv'B=Hash(ID-Ci)*G^Prv'C^Prv'B
//			Carol: Pub'Ai-C=Pub'Ai^Prv'C=Hash(ID-Ai)*G^Prv'A^Prv'C
// Step 5: 参与方交换经过二次加密的样本特征ID加密集合
//			Alice广播：Pub'Bi-A
//			  Bob广播：Pub'Ai-B
//			Carol广播：Pub'Ai-C
// Step 6: 参与方进行再次加密
//			Alice: Pub'Ci-B-A=Pub'Ci^Prv'B=Hash(ID-Ci)*G^Prv'C^Prv'B^Prv'A
//			  Bob: Pub'Ai-C-B=Pub'Ai^Prv'C=Hash(ID-Ai)*G^Prv'A^Prv'C^Prv'B
//			Carol: Pub'Bi-A-C=Pub'Bi-A^Prv'C=Hash(ID-Bi)*G^Prv'B^Prv'A^Prv'C
// Step 7: 参与方交换经过三次加密的样本特征ID加密集合
//			Alice广播：Pub'Ci-B-A
//			  Bob广播：Pub'Ai-C-B
//			Carol广播：Pub'Bi-A-C
// Step 8: 参与方分别对比样本特征ID加密集合，获得数值相等的部分，这就是交集

// Empty 定义一个空struct，用来降低map的存储开销
type Empty struct{}

// EncSet 样本加密集合
type EncSet struct {
	EncIDs map[string]int
}

// EncryptSampleIDSet 参与方分别对自己的样本特征ID进行计算，通过如下方式获得每个样本特征ID的加密集合：
// Alice: Pub'Ai=HP(ID-Ai)^Prv'A=Hash(ID-Ai)*G^Prv'A=Hash(ID-Ai)*Pub'A
// Bob: Pub'Bi=HP(ID-Bi)^Prv'B=Hash(ID-Bi)*G^Prv'B=Hash(ID-Bi)*Pub'B
func EncryptSampleIDSet(sampleID []string, publicKey *ecdsa.PublicKey) *EncSet {
	curve := publicKey.Curve

	//	encIDs := make(map[string]Empty)
	encIDs := make(map[string]int)

	for i := 0; i < len(sampleID); i++ {
		// Hash(ID)*Pub
		newX, newY := curve.ScalarMult(publicKey.X, publicKey.Y, hash.HashUsingSha256([]byte(sampleID[i])))
		id := string(elliptic.Marshal(curve, newX, newY))
		encIDs[id] = i
	}

	encSet := &EncSet{
		EncIDs: encIDs,
	}

	return encSet
}

// ReEncryptIDSet 参与方使用自己的私钥对其它方的样本特征ID加密集合进行二次加密，如：
// Alice: Pub'Bi-A=Pub'Bi^Prv'A=Hash(ID-Bi)*G^Prv'B^Prv'A
// Bob: Pub'Ai-B=Pub'Ai^Prv'B=Hash(ID-Ai)*G^Prv'A^Prv'B
func ReEncryptIDSet(encSet *EncSet, privateKey *ecdsa.PrivateKey) *EncSet {
	curve := privateKey.PublicKey.Curve

	//	var encIDs []*ID
	encIDs := make(map[string]int)

	for idstr, value := range encSet.EncIDs {
		// Pub'Bi^Prv'A
		x, y := elliptic.Unmarshal(curve, []byte(idstr))
		newX, newY := curve.ScalarMult(x, y, privateKey.D.Bytes())
		id := string(elliptic.Marshal(curve, newX, newY))

		encIDs[id] = value
	}

	newEncSet := &EncSet{
		EncIDs: encIDs,
	}

	return newEncSet
}

// Intersect 加密样本对齐，支持多方求交
func Intersect(sampleID []string, reEncSetLocal *EncSet, reEncSetOthers []*EncSet) []string {
	idSetLocal := reEncSetLocal.EncIDs
	var intersection []string

	// 遍历样本集合A
	for id, value := range idSetLocal {
		isExist := false
		for _, reEncSetOther := range reEncSetOthers {
			idSetOther := reEncSetOther.EncIDs
			// 如果ID在某一方不存在，则继续判断下一个ID
			if _, isExist = idSetOther[id]; !isExist {
				break
			}
		}

		// 如果A中的元素在B中也存在，那么放入交集中
		if isExist {
			intersection = append(intersection, sampleID[value])
		}
	}

	return intersection
}
