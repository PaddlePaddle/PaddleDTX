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

package validation

import (
	"crypto/md5"
	"errors"
	"math/rand"
	"sort"
)

// Split divides the file into two parts directly
// based on percentage which denotes the first part of return.
// The first row of `fileRows` contains just names of feature, and it should be kept in both parts of return
func Split(fileRows [][]string, percents int) ([2][][]string, error) {
	total := len(fileRows)
	if total < 1 {
		return [2][][]string{}, errors.New("invalid file")
	}

	if percents < 0 {
		percents = 0
	}
	if percents > 100 {
		percents = 100
	}

	l := (total - 1) * percents / 100

	//reserve capacity for names of feature
	firstP := make([][]string, 0, l+1)
	firstP = append(firstP, fileRows[0])
	secondP := make([][]string, 0, total-l)
	secondP = append(secondP, fileRows[0])

	for i, r := range fileRows[1:] {
		if i < l {
			firstP = append(firstP, r)
		} else {
			secondP = append(secondP, r)
		}
	}

	ret := [2][][]string{
		firstP,
		secondP,
	}
	return ret, nil
}

// sortById sorts file rows by IDs which extracted from file by `idName`
// in a stable way (while keeping the original order of equal IDs).
func sortById(fileRows [][]string, idName string) ([][]string, error) {
	// find where the IDs are
	idx := -1
	for i, v := range fileRows[0] {
		if v == idName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return [][]string{}, errors.New("no IDName found")
	}

	// extract IDs from file and use IDs as keys to build a map, which is the preparation for sort
	lenFile := len(fileRows)
	mapFileRows := make(map[string][][]string, lenFile-1)
	ids := make([]string, 0, lenFile-1)

	for _, r := range fileRows[1:] { // first row contains just names of feature, skip it
		if len(r) <= idx {
			return [][]string{}, errors.New("invalid file")
		}

		if _, ok := mapFileRows[r[idx]]; ok {
			mapFileRows[r[idx]] = append(mapFileRows[r[idx]], r)
		} else {
			mapFileRows[r[idx]] = [][]string{r}
			ids = append(ids, r[idx])
		}
	}

	// sort the IDs
	sort.Strings(ids)

	// rebuild file according to the reordered keys and return
	newFile := make([][]string, 0, lenFile)
	newFile = append(newFile, fileRows[0]) // add first row back to top
	for _, id := range ids {
		newFile = append(newFile, mapFileRows[id]...)
	}

	return newFile, nil
}

// shuffle shuffles rows of a file.
// seed is a string, and MD5 would be applied to seed,
//  then the result would be converted to type-int64 value which is as the real seed used to shuffle rows.
// returns the shuffled file.
func shuffle(fileRows [][]string, seed string) [][]string {
	ms := md5.Sum([]byte(seed))

	var rSeed int64

	// we take the first 8 bytes as input to calculate the real seed,
	// and we are likely to get a negtive number because of overflow.
	for i := 0; i < 8; i++ {
		rSeed <<= 8
		rSeed += int64(ms[i])
	}

	// first row contains just names of feature, skip it
	newFile := fileRows[1:]
	// shuffle the left rows
	r := rand.New(rand.NewSource(rSeed))
	r.Shuffle(len(newFile), func(i, j int) {
		newFile[i], newFile[j] = newFile[j], newFile[i]
	})

	return fileRows
}

// ShuffleSplit sorts file rows by IDs which extracted from file by `idName`,
// and shuffles the sorted rows,
// then divides the file into two parts
// based on `percents` which denotes the first part of return.
func ShuffleSplit(fileRows [][]string, idName string, percents int, seed string) ([2][][]string, error) {
	newFileRows, err := sortById(fileRows, idName)
	if err != nil {
		return [2][][]string{}, err
	}

	newFileRows = shuffle(newFileRows, seed)

	retFile, err := Split(newFileRows, percents)
	if err != nil {
		return [2][][]string{}, err
	}

	return retFile, nil
}

// KFoldsSplit divides the file into `k` parts directly.
// k is the number of parts that only could be 5 or 10.
// The first row of `fileRows` contains just names of feature, and it should be kept in all parts of return.
func KFoldsSplit(fileRows [][]string, k int) ([][][]string, error) {
	if k != 5 && k != 10 {
		return [][][]string{}, errors.New("k only could be 5 or 10")
	}

	total := len(fileRows)
	if total < 1 {
		return [][][]string{}, errors.New("invalid file")
	}

	if total < k+1 {
		return [][][]string{}, errors.New("file is too small for k")
	}

	subsets := make([][][]string, 0, k)

	remain := (total - 1) % k
	div := (total - 1) / k

	for i := 1; i < total; { // first row contains just names of feature, skip it
		j := i

		if remain > 0 {
			i += div + 1
			remain--
		} else {
			i += div
		}

		ss := make([][]string, 0, i-j+1)
		ss = append(ss, fileRows[0]) // add first row back to top of each subset
		ss = append(ss, fileRows[j:i]...)

		subsets = append(subsets, ss)
	}

	return subsets, nil
}

// ShuffleKFoldsSplit sorts file rows by IDs which extracted from file by `idName`,
// and shuffles the sorted rows,
// then divides the file into `k` parts.
// k is the number of parts that only could be 5 or 10.
func ShuffleKFoldsSplit(fileRows [][]string, idName string, k int, seed string) ([][][]string, error) {
	newFileRows, err := sortById(fileRows, idName)
	if err != nil {
		return [][][]string{}, err
	}

	newFileRows = shuffle(newFileRows, seed)

	retFile, err := KFoldsSplit(newFileRows, k)

	if err != nil {
		return [][][]string{}, err
	}

	return retFile, nil
}

// LooSplit sorts file rows by IDs which extracted from file by `idName`,
// then divides each row into a subset.
func LooSplit(fileRows [][]string, idName string) ([][][]string, error) {
	total := len(fileRows)
	if total < 2 {
		return [][][]string{}, errors.New("invalid file")
	}

	newFileRows, err := sortById(fileRows, idName)
	if err != nil {
		return [][][]string{}, err
	}

	subsets := make([][][]string, 0, total-1)
	for i := 1; i < total; i++ { // first row contains just names of feature, skip it
		ss := make([][]string, 0, 2)
		ss = append(ss, newFileRows[0], newFileRows[i]) // add first row back to top of each subset

		subsets = append(subsets, ss)

	}

	return subsets, nil
}
