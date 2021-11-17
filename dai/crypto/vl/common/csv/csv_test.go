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

package csv

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/utils"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
)

// TestCSV test csv read and write rows
func TestCSV(t *testing.T) {
	row1 := []string{"1", "aaa"}
	row2 := []string{"2", "bbb"}
	row3 := []string{"3", "ccc"}
	fileRows := [][]string{row1, row2, row3}

	path := "./tmp"
	defer os.Remove(path)

	err := WriteRowsToFile(fileRows, path)
	checkErr(err, t)

	content, err := ioutil.ReadFile(path)
	checkErr(err, t)

	newRows, err := ReadRowsFromFile(content)
	checkErr(err, t)

	if !reflect.DeepEqual(fileRows, newRows) {
		t.Logf("original rows: %v\n", fileRows)
		t.Logf("retrieved rows: %v\n", newRows)
		t.Error("TestCSV failed")
	}
}

func TestPSI(t *testing.T) {
	idName := "id"

	privkeyA, err := vl_common.GeneratePSIKeyPair()
	checkErr(err, t)
	privkeyB, err := vl_common.GeneratePSIKeyPair()
	checkErr(err, t)

	fileContentA, err := ioutil.ReadFile("../../testdata/psi/dataA.csv")
	checkErr(err, t)
	fileContentB, err := ioutil.ReadFile("../../testdata/psi/dataB.csv")
	checkErr(err, t)

	rowsA, IDsA, err := ReadIDsFromFileRows(fileContentA, idName)
	checkErr(err, t)
	rowsB, IDsB, err := ReadIDsFromFileRows(fileContentB, idName)
	checkErr(err, t)

	encIDsA, err := vl_common.EncryptSampleIDSet(IDsA, &privkeyA.PublicKey)
	checkErr(err, t)
	encIDsB, err := vl_common.EncryptSampleIDSet(IDsB, &privkeyB.PublicKey)
	checkErr(err, t)

	reEncIDsA, err := vl_common.ReEncryptIDSet(encIDsA, privkeyB)
	checkErr(err, t)
	reEncIDsB, err := vl_common.ReEncryptIDSet(encIDsB, privkeyA)
	checkErr(err, t)

	intersectA, err := vl_common.IntersectTwoParts(IDsA, reEncIDsA, reEncIDsB)
	checkErr(err, t)
	intersectB, err := vl_common.IntersectTwoParts(IDsB, reEncIDsB, reEncIDsA)
	checkErr(err, t)

	t.Logf("intersetcA: %v\n", intersectA)
	t.Logf("intersetcB: %v\n", intersectB)

	for _, id := range intersectA {
		if !utils.StringInSlice(id, intersectB) {
			t.Errorf("intersect sets are not equal ")
			t.FailNow()
		}
	}

	newRowsA, err := vl_common.RearrangeFileWithIntersectIDs(rowsA, idName, intersectA)
	checkErr(err, t)
	newRowsB, err := vl_common.RearrangeFileWithIntersectIDs(rowsB, idName, intersectB)
	checkErr(err, t)

	for i := 0; i < len(newRowsA); i++ {
		t.Log(newRowsA[i])
	}

	for i := 0; i < len(newRowsB); i++ {
		t.Log(newRowsB[i])
	}
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
