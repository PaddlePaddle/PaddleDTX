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

package sortable

import (
	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
)

type Challenges []blockchain.Challenge

// descending order
func (c Challenges) Less(i, j int) bool {
	return c[i].ChallengeTime > c[j].ChallengeTime
}

func (c Challenges) Len() int {
	return len(c)
}

func (c Challenges) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
