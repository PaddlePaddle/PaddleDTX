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

package mock

import (
	"context"

	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
)

type MockChallenger struct {
}

func New() *MockChallenger {
	return &MockChallenger{}
}

func (m *MockChallenger) GenerateChallenge(maxIdx int) ([][]byte, [][]byte, error) {
	return nil, nil, nil
}

func (m *MockChallenger) Close() {
}

func (m *MockChallenger) GetChallengeConf() (a string, pdp types.PDP) {
	return a, pdp
}

func (m *MockChallenger) Setup(sliceData []byte, mount int) (c []ctype.RangeHash, err error) {
	return c, err
}

func (m *MockChallenger) NewSetup(sliceData []byte, rangeAmount int, merkleMaterialQueue chan<- ctype.Material, cm ctype.Material) error {
	return nil
}

func (m *MockChallenger) Save(ctx context.Context, cms []ctype.Material) error {
	return nil
}

func (m *MockChallenger) Take(ctx context.Context, fileID string, sliceID string, nodeID []byte) (c ctype.RangeHash, err error) {
	return c, err
}
