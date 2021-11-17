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

package rand

import (
	"crypto/rand"
	"crypto/sha512"

	"golang.org/x/crypto/pbkdf2"
)

// 定义不同int类型对应的key length
const (
	// int8 类型
	KeyLengthInt8 = 8

	// int16 类型
	KeyLengthInt16 = 16

	// int32 类型
	KeyLengthInt32 = 32

	// int64 类型
	KeyLengthInt64 = 64
)

const (
	// 安全强度低
	KeyStrengthEasy = iota

	// 安全强度中
	KeyStrengthMiddle

	// 安全强度高
	KeyStrengthHard
)

// GenerateEntropy 底层调用跟操作系统相关的函数（读取系统熵）来产生一些伪随机数，
// 对外建议管这个返回值叫做“熵”
func GenerateEntropy(bitSize int) ([]byte, error) {
	err := validateEntropyBitSize(bitSize)
	if err != nil {
		return nil, err
	}

	entropy := make([]byte, bitSize/8)
	_, err = rand.Read(entropy)
	return entropy, err
}

//  validateEntropyBitSize 检查指定Entropy的比特长度是否符合规范要求：
//  在128-256之间，并且是32的倍数
//  设计背景详见比特币改进计划第39号提案的数学模型
//
//  checksum length (CS)
//  entropy length (ENT)
//  mnemonic sentence (MS)
//
//	CS = ENT / 32
//	MS = (ENT + CS) / 11
//
//	|  ENT  | CS | ENT+CS |  MS  |
//	+-------+----+--------+------+
//	|  128  |  4 |   132  |  12  |
//	|  160  |  5 |   165  |  15  |
//	|  192  |  6 |   198  |  18  |
//	|  224  |  7 |   231  |  21  |
//	|  256  |  8 |   264  |  24  |
func validateEntropyBitSize(bitSize int) error {
	if (bitSize%32) != 0 || bitSize < 128 || bitSize > 256 {
		return ErrInvalidEntropyLength
	}
	return nil
}

// generateSeedWithRandomPassword 生成一个指定长度的随机数种子
func generateSeedWithRandomPassword(randomPassword []byte, keyLen int) []byte {
	salt := "jingbo is handsome."
	seed := pbkdf2.Key(randomPassword, []byte(salt), 2048, keyLen, sha512.New)

	return seed
}

// GenerateSeedWithStrengthAndKeyLen 生成指定强度和长度的随机熵
func GenerateSeedWithStrengthAndKeyLen(strength int, keyLength int) ([]byte, error) {
	var entropyBitLength = 0
	//根据强度来判断随机数长度
	switch strength {
	case KeyStrengthEasy: // 弱
		entropyBitLength = 128
	case KeyStrengthMiddle: // 中
		entropyBitLength = 192
	case KeyStrengthHard: // 高
		entropyBitLength = 256
	default: // 不支持的语言类型
	}

	// 判断强度是否合法
	if entropyBitLength == 0 {
		return nil, ErrStrengthNotSupported
	}

	// 产生随机熵
	entropyByte, err := GenerateEntropy(entropyBitLength)
	if err != nil {
		return nil, err
	}

	return generateSeedWithRandomPassword(entropyByte, keyLength), nil
}
