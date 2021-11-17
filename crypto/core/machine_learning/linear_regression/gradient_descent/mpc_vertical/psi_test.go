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
	"crypto/rand"
	"encoding/json"
	"testing"
)

func TestPSI(t *testing.T) {
	var sampleIDsA []string
	sampleIDsA = append(sampleIDsA, "10000")
	sampleIDsA = append(sampleIDsA, "10001")
	sampleIDsA = append(sampleIDsA, "10002")
	sampleIDsA = append(sampleIDsA, "10003")
	sampleIDsA = append(sampleIDsA, "10004")
	sampleIDsA = append(sampleIDsA, "10005")
	sampleIDsA = append(sampleIDsA, "10006")

	var sampleIDsB []string
	sampleIDsB = append(sampleIDsB, "10000")
	sampleIDsB = append(sampleIDsB, "10001")
	sampleIDsB = append(sampleIDsB, "10005")
	sampleIDsB = append(sampleIDsB, "10006")
	sampleIDsB = append(sampleIDsB, "10007")
	sampleIDsB = append(sampleIDsB, "10008")
	sampleIDsB = append(sampleIDsB, "10009")

	var sampleIDsC []string
	sampleIDsC = append(sampleIDsC, "88888")
	sampleIDsC = append(sampleIDsC, "99999")
	sampleIDsC = append(sampleIDsC, "10001")
	sampleIDsC = append(sampleIDsC, "10005")
	sampleIDsC = append(sampleIDsC, "10008")
	sampleIDsC = append(sampleIDsC, "10010")
	sampleIDsC = append(sampleIDsC, "10011")

	privateKeyA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Errorf("privateKeyA generation failed: %v", err)
	}
	privateKeyB, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Errorf("privateKeyB generation failed: %v", err)
	}
	privateKeyC, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Errorf("privateKeyC generation failed: %v", err)
	}

	encSetA := EncryptSampleIDSet(sampleIDsA, &privateKeyA.PublicKey)
	encSetB := EncryptSampleIDSet(sampleIDsB, &privateKeyB.PublicKey)
	encSetC := EncryptSampleIDSet(sampleIDsC, &privateKeyC.PublicKey)

	// 计算A和B的隐私交集
	reEncSetB := ReEncryptIDSet(encSetB, privateKeyA)
	reEncSetA := ReEncryptIDSet(encSetA, privateKeyB)

	var reEncSetOthers []*EncSet
	reEncSetOthers = append(reEncSetOthers, reEncSetB)

	intersection := Intersect(sampleIDsA, reEncSetA, reEncSetOthers)

	jsonIntersection, err := json.Marshal(intersection)
	if err != nil {
		t.Errorf("failed to marshal A and B intersection: %v", err)
	}
	t.Logf("intersection of A and B is %s", jsonIntersection)

	// 计算A、B、C的隐私交集
	reEncSetBA := ReEncryptIDSet(encSetB, privateKeyA)
	reEncSetBAC := ReEncryptIDSet(reEncSetBA, privateKeyC)

	reEncSetAB := ReEncryptIDSet(encSetA, privateKeyB)
	reEncSetABC := ReEncryptIDSet(reEncSetAB, privateKeyC)

	reEncSetCA := ReEncryptIDSet(encSetC, privateKeyA)
	reEncSetCAB := ReEncryptIDSet(reEncSetCA, privateKeyB)

	var reEncSetOthers2 []*EncSet
	reEncSetOthers2 = append(reEncSetOthers2, reEncSetBAC)
	reEncSetOthers2 = append(reEncSetOthers2, reEncSetCAB)

	intersection = Intersect(sampleIDsA, reEncSetABC, reEncSetOthers2)

	jsonIntersection, err = json.Marshal(intersection)
	if err != nil {
		t.Errorf("failed to marshal A, B and C intersection: %v", err)
	}
	t.Logf("intersection of A、B、C is %s", jsonIntersection)
}
