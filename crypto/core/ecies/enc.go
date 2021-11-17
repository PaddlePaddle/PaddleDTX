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

package ecies

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/config"
	libecies "github.com/PaddlePaddle/PaddleDTX/crypto/core/ecies/libecies"
)

// Encrypt 非对称加密
func Encrypt(k *ecdsa.PublicKey, msg []byte) (cypherText []byte, err error) {
	// 判断是否是NIST标准的公钥
	isNistCurve := checkKeyCurve(k)
	if !isNistCurve {
		return nil, fmt.Errorf("this cryptography curve[%s] has not been supported yet", k.Params().Name)
	}

	pub := libecies.ImportECDSAPublic(k)

	ct, err := libecies.Encrypt(rand.Reader, pub, msg, nil, nil)
	if err != nil {
		return nil, err
	}

	return ct, nil
}

// checkKeyCurve 判断是否是NIST标准的公钥
func checkKeyCurve(k *ecdsa.PublicKey) bool {
	if k.X == nil || k.Y == nil {
		return false
	}

	switch k.Params().Name {
	case config.CurveNist: // NIST
		return true
	default: // 不支持的密码学类型
		return false
	}
}

// Decrypt 非对称解密
func Decrypt(k *ecdsa.PrivateKey, cypherText []byte) (msg []byte, err error) {
	// 判断是否是NIST标准的私钥
	isNistCurve := checkKeyCurve(&k.PublicKey)
	if !isNistCurve {
		return nil, fmt.Errorf("this cryptography curve[%s] has not been supported yet", k.Params().Name)
	}
	if k.D == nil {
		return nil, fmt.Errorf("param D cannot be nil")
	}

	prv := libecies.ImportECDSA(k)

	pt, err := prv.Decrypt(rand.Reader, cypherText, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return pt, nil
}
