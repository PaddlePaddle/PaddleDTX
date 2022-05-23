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

package local

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	s "github.com/PaddlePaddle/PaddleDTX/xdb/storage"
)

var localStorage s.Storage
var key string = "1b369c87-2503-434a-9db1-2dac8e78a12a"
var index string

func TestSaveV2(t *testing.T) {
	localStorage = s.NewStorage(&storageV2{*storage})

	content := "test file content"
	ind, err := localStorage.Save(key, bytes.NewReader([]byte(content)))
	require.NoError(t, err)

	index = ind
	t.Logf("Saved data. Index: %s", index)
}

func TestLoadV2(t *testing.T) {
	f, err := localStorage.Load(key, index)
	require.NoError(t, err)
	b, err := ioutil.ReadAll(f)
	f.Close()
	require.NoError(t, err)
	require.Equal(t, "test file content", string(b))
}

func TestExistV2(t *testing.T) {
	ex, err := localStorage.Exist(key, index)
	require.NoError(t, err)
	require.Equal(t, true, ex)
}

func TestLoadStrV2(t *testing.T) {
	str, err := localStorage.LoadStr(key, index)
	require.NoError(t, err)

	require.Equal(t, "test file content", str)
}

func UpdateV2(t *testing.T) {
	reader := bytes.NewReader([]byte("I love China!"))
	ind, err := localStorage.Update(key, index, reader)
	require.NoError(t, err)
	index = ind
	f, err := localStorage.Load(key, index)
	require.NoError(t, err)
	b, err := ioutil.ReadAll(f)
	f.Close()
	require.NoError(t, err)
	require.Equal(t, "I love China!", string(b))
}

func TestDeleteV2(t *testing.T) {
	err := localStorage.Delete(key, index)
	require.NoError(t, err)
}
