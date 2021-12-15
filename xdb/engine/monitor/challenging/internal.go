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

package challenging

import (
	"strconv"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"

	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
)

type randomProof struct {
	Sigma     []byte
	Mu        []byte
	Signature []byte
}

type rangeProof struct {
	Proof     []byte
	Signature []byte
}

func Calculate(opt *ctype.CalculateOptions) []byte {
	ts := strconv.FormatInt(opt.Timestamp, 10)
	data := append(append([]byte{}, opt.RangeHash...), []byte(ts)...)

	return hash.HashUsingSha256(data)
}
