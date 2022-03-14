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

package evaluator

import (
	"bytes"
	"encoding/csv"
	"testing"
)

func TestCSV(t *testing.T) {
	var binClassFileRows = [][]string{
		[]string{"name1", "name2", "name3", "name4", "name5", "label", "id"},
		[]string{"11", "12", "13", "14", "15", "yes", "3"},
		[]string{"21", "22", "23", "24", "25", "yes", "6"},
		[]string{"31", "32", "33", "34", "35", "no", "1"},
		[]string{"41", "42", "43", "44", "45", "no", "4"},
		[]string{"51", "52", "53", "54", "55", "yes", "2"},
		[]string{"61", "62", "63", "64", "65", "yes", "5"},
		[]string{"11", "12", "13", "14", "15", "no", "13"},
		[]string{"21", "22", "23", "24", "25", "yes", "16"},
		[]string{"31", "32", "33", "34", "35", "no", "11"},
		[]string{"41", "42", "43", "44", "45", "no", "14"},
		[]string{"51", "52", "53", "54", "55", "yes", "12"},
		[]string{"61", "62", "63", "64", "65", "no", "15"},
	}
	var b bytes.Buffer

	w := csv.NewWriter(&b)
	w.WriteAll(binClassFileRows)

	bRes := b.Bytes()
	t.Logf("a list of string convert to bytes: %v, %s", bRes, string(bRes))

	r := csv.NewReader(&b)
	ss, err := r.ReadAll()
	checkErr(err, t)

	for _, s := range ss {
		t.Logf("bytes convert to a list of string: %v", s)
	}
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
