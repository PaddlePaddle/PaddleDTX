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
	"testing"
)

func TestShuffle(t *testing.T) {
	fileRows := [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5"},
		[]string{"11", "12", "13", "14", "15"},
		[]string{"21", "22", "23", "24", "25"},
		[]string{"31", "32", "33", "34", "35"},
		[]string{"41", "42", "43", "44", "45"},
		[]string{"51", "52", "53", "54", "55"},
		[]string{"61", "62", "63", "64", "65"},
	}

	retFile := shuffle(fileRows, "18f168b6-2ef2-491e-8b26-4aa6df18378a")
	t.Logf("Original file is %v, and shuffled file is %v", fileRows, retFile)
}

func TestSortAndSplit(t *testing.T) {
	fileRows := [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "id"},
		[]string{"11", "12", "13", "14", "15", "3"},
		[]string{"21", "22", "23", "24", "25", "6"},
		[]string{"31", "32", "33", "34", "35", "1"},
		[]string{"41", "42", "43", "44", "45", "4"},
		[]string{"51", "52", "53", "54", "55", "2"},
		[]string{"61", "62", "63", "64", "65", "5"},
	}

	sortedFile, err := sortById(fileRows, "id")
	checkErr(err, t)

	retFile, err := Split(sortedFile, -1)
	checkErr(err, t)

	t.Logf("Original file is %v, and sorted file is %v, and split file is %v", fileRows, sortedFile, retFile)

	fileRows = [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "id"},
		[]string{"11", "12", "13", "14", "15", "car"},
		[]string{"21", "22", "23", "24", "25", "face"},
		[]string{"31", "32", "33", "34", "35", "apple"},
		[]string{"41", "42", "43", "44", "45", "dog"},
		[]string{"51", "52", "53", "54", "55", "bus"},
		[]string{"61", "62", "63", "64", "65", "egg"},
	}

	sortedFile, err = sortById(fileRows, "id")
	checkErr(err, t)

	retFile, err = Split(sortedFile, 40)
	checkErr(err, t)

	t.Logf("Original file is %v, and sorted file is %v, and split file is %v", fileRows, sortedFile, retFile)

}

func TestShuffleSplit(t *testing.T) {
	fileRows := [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "id"},
		[]string{"11", "12", "13", "14", "15", "3"},
		[]string{"21", "22", "23", "24", "25", "6"},
		[]string{"31", "32", "33", "34", "35", "1"},
		[]string{"41", "42", "43", "44", "45", "4"},
		[]string{"51", "52", "53", "54", "55", "2"},
		[]string{"61", "62", "63", "64", "65", "5"},
	}

	splitFile, err := ShuffleSplit(fileRows, "id", 40, "18f168b6-2ef2-491e-8b26-4aa6df18378a")
	checkErr(err, t)

	t.Logf("Original file is %v, and split file is %v", fileRows, splitFile)
}

func TestShuffleKFoldsSplit(t *testing.T) {
	fileRows := [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "id"},
		[]string{"11", "12", "13", "14", "15", "3"},
		[]string{"21", "22", "23", "24", "25", "6"},
		[]string{"31", "32", "33", "34", "35", "1"},
		[]string{"41", "42", "43", "44", "45", "4"},
		[]string{"51", "52", "53", "54", "55", "2"},
		[]string{"61", "62", "63", "64", "65", "5"},
	}

	splitFile, err := ShuffleKFoldsSplit(fileRows, "id", 5, "18f168b6-2ef2-491e-8b26-4aa6df18378a")
	checkErr(err, t)

	t.Logf("Original file is %v, and split file is %v", fileRows, splitFile)

	fileRows = [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "id"},
		[]string{"11", "12", "13", "14", "15", "3"},
		[]string{"21", "22", "23", "24", "25", "6"},
		[]string{"31", "32", "33", "34", "35", "1"},
		[]string{"41", "42", "43", "44", "45", "4"},
		[]string{"51", "52", "53", "54", "55", "2"},
		[]string{"61", "62", "63", "64", "65", "5"},
		[]string{"11", "12", "13", "14", "15", "13"},
		[]string{"21", "22", "23", "24", "25", "16"},
		[]string{"31", "32", "33", "34", "35", "11"},
		[]string{"41", "42", "43", "44", "45", "14"},
		[]string{"51", "52", "53", "54", "55", "12"},
		[]string{"61", "62", "63", "64", "65", "15"},
	}

	splitFile, err = ShuffleKFoldsSplit(fileRows, "id", 10, "18f168b6-2ef2-491e-8b26-4aa6df18378a")
	checkErr(err, t)

	t.Logf("Original file is %v, and split file is %v", fileRows, splitFile)
}

func TestLooSplit(t *testing.T) {
	fileRows := [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "id"},
		[]string{"11", "12", "13", "14", "15", "3"},
		[]string{"21", "22", "23", "24", "25", "6"},
		[]string{"31", "32", "33", "34", "35", "1"},
		[]string{"41", "42", "43", "44", "45", "4"},
		[]string{"51", "52", "53", "54", "55", "2"},
		[]string{"61", "62", "63", "64", "65", "5"},
	}

	splitFile, err := LooSplit(fileRows, "id")
	checkErr(err, t)

	t.Logf("Original file is %v, and split file is %v", fileRows, splitFile)

}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
