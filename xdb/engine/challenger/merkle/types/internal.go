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

package types

type Range struct {
	Start uint64
	End   uint64
}

type RangeHash struct {
	Ranges []Range
	Hash   []byte
	Used   bool
}

type Material struct {
	FileID  string
	SliceID string
	NodeID  []byte
	Ranges  []RangeHash
}

type CalculateOptions struct {
	RangeHash []byte
	Timestamp int64
}

type AnswerCalculateOptions struct {
	RangeHashes [][]byte
	Timestamp   int64
}

type MaterialStorage interface {
	Save(cms []Material) error
	Load(key []byte) (Material, error)
	NewIterator(prefix []byte) ([][]byte, error)
	Update(cms Material, key []byte) error

	Close()
}
