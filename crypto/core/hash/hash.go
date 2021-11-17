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

package hash

import (
	"crypto/sha256"
)

// HashUsingSha256 使用标准SHA2-256算法计算哈希
func HashUsingSha256(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	out := h.Sum(nil)

	return out
}

// DoubleSha256 执行2次SHA256
func DoubleSha256(data []byte) []byte {
	return HashUsingSha256(HashUsingSha256(data))
}
