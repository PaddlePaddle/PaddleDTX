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

package ipfs

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"
	"time"

	s "github.com/PaddlePaddle/PaddleDTX/xdb/storage"
)

var ipfs s.Storage
var index string
var key string = "1b369c87-2503-434a-9db1-2dac8e78a12a"

func TestMain(m *testing.M) {
	tem, err := New([]string{"localhost:5001", "localhost:5002", "localhost:5003", "127.0.0.1:5004", "127.0.0.1:5005"}, time.Millisecond*300)
	if err != nil {
		log.Panic(err)
	}
	ipfs = s.NewStorage(tem)
	m.Run()
}

func TestSave(t *testing.T) {

	content := "test file content"
	cid, err := ipfs.Save(key, bytes.NewReader([]byte(content)))
	checkErr(err, t)

	index = cid
	t.Logf("Saved data. Index(cid): %s", index)
}

func TestLoad(t *testing.T) {
	f, err := ipfs.Load(key, index)
	checkErr(err, t)
	defer f.Close()

	cont, err := ioutil.ReadAll(f)
	checkErr(err, t)

	t.Logf("Loaded data. Content: %s", string(cont))
}

func TestExist(t *testing.T) {
	e, err := ipfs.Exist(key, index)
	checkErr(err, t)

	t.Logf("Data: %s Exist: %t", key, e)
}

func TestUpdate(t *testing.T) {
	content := "new test file content"
	cid, err := ipfs.Update(key, index, bytes.NewReader([]byte(content)))
	checkErr(err, t)

	index = cid
	t.Logf("Updated data. Index(cid): %s", index)
}

func TestLoadStr(t *testing.T) {
	str, err := ipfs.LoadStr(key, index)
	checkErr(err, t)

	t.Logf("Loaded data. Content: %s", str)
}

func TestDelete(t *testing.T) {
	err := ipfs.Delete(key, index)
	checkErr(err, t)

	t.Logf("Data %s Deleted", key)
}
