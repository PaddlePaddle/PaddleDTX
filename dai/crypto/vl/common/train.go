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

package common

import (
	"fmt"
	"sort"

	pb_common "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
)

// GetBatchSetBySize get train set for specific round by batch size
// - trainSet is sample set for training
// - params is training task params
// - round is loop round for training task
// - needCheckReorder indicates whether reorder is needed
func GetBatchSetBySize(trainSet [][]float64, params pb_common.TrainParams, round int, needCheckReorder bool) ([][]float64, [][]float64) {
	sampleNum := len(trainSet)

	// if batch size is zero or greater than sample num, return all train set
	var trainSetThisRound [][]float64
	if params.BatchSize == 0 || int(params.BatchSize) >= sampleNum {
		trainSetThisRound = append(trainSetThisRound, trainSet...)
		return trainSetThisRound, trainSet
	}

	// number of train set for this round is batch size
	trainSetThisRound = make([][]float64, params.BatchSize)

	segmentIdx := round % (sampleNum / int(params.BatchSize))
	// if this loop just started, check if train set need to be reordered
	// if all samples already used once, reorder train set and start from the first segment
	if needCheckReorder && segmentIdx == 0 {
		// determine if all samples are already used
		newSet := randTrainSet(trainSet, round)
		copy(trainSet[0:], newSet)
	}

	start := segmentIdx * int(params.BatchSize)
	copy(trainSetThisRound[0:], trainSet[start:start+int(params.BatchSize)])

	return trainSetThisRound, trainSet
}

// randTrainSet rearrange train set in deterministic random order
func randTrainSet(trainSet [][]float64, round int) [][]float64 {
	reOrderedSet := make([][]float64, len(trainSet))
	// map hash(idx+round) to idx
	idxHashMap := make(map[string]int)
	var hashes []string

	for i := 0; i < len(trainSet); i++ {
		msg := fmt.Sprintf("%d+%d", i, round)
		s := string(xchainCryptoClient.HashUsingSha256([]byte(msg)))
		idxHashMap[s] = i
		hashes = append(hashes, s)
	}

	// sort hash(idx, round)
	sort.Strings(hashes)

	// reorder train set
	for i := 0; i < len(hashes); i++ {
		idx := idxHashMap[hashes[i]]
		reOrderedSet[i] = trainSet[idx]
	}

	return reOrderedSet
}
