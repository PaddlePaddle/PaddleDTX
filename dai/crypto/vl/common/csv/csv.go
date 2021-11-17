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
	"bytes"
	"encoding/csv"
	"fmt"
	"os"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
)

// ReadIDsFromFileRows read ID set from file content, return all rows, ID set, error
func ReadIDsFromFileRows(fileContent []byte, idName string) ([][]string, []string, error) {
	rows, err := ReadRowsFromFile(fileContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read rows from file: %v", err)
	}
	IDs, err := vl_common.RetrieveIDsFromFile(rows, idName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve ID set from file rows: %v", err)
	}
	return rows, IDs, err
}

// ReadRowsFromFile read all rows from csv file content
func ReadRowsFromFile(fileContent []byte) ([][]string, error) {
	r := csv.NewReader(bytes.NewReader(fileContent))
	return r.ReadAll()
}

// WriteRowsToFile write all rows to csv file
func WriteRowsToFile(fileRows [][]string, path string) error {
	if _, err := os.Create(path); err != nil {
		return err
	}
	writeFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer writeFile.Close()

	w := csv.NewWriter(writeFile)

	for i := 0; i < len(fileRows); i++ {
		w.Write(fileRows[i])
		w.Flush()
	}

	return nil
}
