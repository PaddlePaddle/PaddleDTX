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

package strings

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const SIGNATURE_NAME = "signature"

// IsContain checks if item exists in a list
func IsContain(items []string, item string) bool {
	for _, every := range items {
		if every == item {
			return true
		}
	}
	return false
}

// IsContainDuplicateItems checks if duplicate items exist in a list
func IsContainDuplicateItems(items []string) bool {
	temp := map[string]struct{}{}
	for _, item := range items {
		if _, ok := temp[item]; ok {
			return true
		}
		temp[item] = struct{}{}
	}
	return false
}

// GetSigMessage gets a messages used to generate signature
// param: signMes is struct or map
// return: the key-value set concatenated with '&'
// example:
// ifce := FLTask{
//	TaskID: "26e02df4-b0ea-492e-84aa-6f1a4f547c9b",
//	Name: "house-predict",
// }
// return "name=house_preidct&taskID=26e02df4-b0ea-492e-84aa-6f1a4f547c9b"
func GetSigMessage(signMes interface{}) (string, error) {
	// get sorted json
	m, err := convertToMap(signMes)
	if err != nil {
		return "", err
	}
	// sort by map keys
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// concatenate key-value
	sigMes := ""
	for _, k := range keys {
		v := m[k]
		if k == SIGNATURE_NAME || v == nil {
			continue
		}
		switch v.(type) {
		case string, json.Number, bool:
			sigMes += fmt.Sprintf("%s=%v&", k, v)
		default:
			val, _ := json.Marshal(v)
			sigMes += fmt.Sprintf("%s=%s&", k, val)
		}
	}
	return strings.TrimRight(sigMes, "&"), nil
}

// convertToMap used to convert struct to map
func convertToMap(struc interface{}) (map[string]interface{}, error) {
	bytes, err := json.Marshal(struc)
	if err != nil {
		return nil, err
	}
	var ifce map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	if err := decoder.Decode(&ifce); err != nil {
		return nil, err
	}
	return ifce, nil
}
