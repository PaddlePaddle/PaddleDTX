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
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
)

var storage *Storage

func TestMain(m *testing.M) {
	os.RemoveAll("slices")
	conf := &config.LocalConf{
		RootPath: "slices",
	}
	stor, err := New(conf)
	if err != nil {
		log.Panic(err)
	}
	storage = stor
	m.Run()
	os.RemoveAll("slices")
}

func TestSave(t *testing.T) {
	content := "test file content"
	err := storage.Save("18f168b6-2ef2-491e-8b26-4aa6df18378a", bytes.NewReader([]byte(content)))
	require.NoError(t, err)
}

func TestLoad(t *testing.T) {
	f, err := storage.Load("18f168b6-2ef2-491e-8b26-4aa6df18378a")
	require.NoError(t, err)
	b, err := ioutil.ReadAll(f)
	f.Close()
	require.NoError(t, err)
	require.Equal(t, "test file content", string(b))
}

func TestExist(t *testing.T) {
	ex, err := storage.Exist("18f168b6-2ef2-491e-8b26-4aa6df18378a")
	require.NoError(t, err)
	require.Equal(t, true, ex)
}

func TestDelete(t *testing.T) {
	ex, err := storage.Delete("18f168b6-2ef2-491e-8b26-4aa6df18378a")
	require.NoError(t, err)
	require.Equal(t, true, ex)
}

func TestSaveAndUpdate(t *testing.T) {
	reader := bytes.NewReader([]byte("1244"))
	err := storage.SaveAndUpdate("18f168b6-2ef2-491e-8b26-4aa6df18378a", reader)
	require.NoError(t, err)
	f, err := storage.Load("18f168b6-2ef2-491e-8b26-4aa6df18378a")
	require.NoError(t, err)
	b, err := ioutil.ReadAll(f)
	f.Close()
	require.NoError(t, err)
	require.Equal(t, "1244", string(b))
}

func TestLoadStr(t *testing.T) {
	s, err := storage.LoadStr("18f168b6-2ef2-491e-8b26-4aa6df18378a")
	require.NoError(t, err)
	require.Equal(t, "1244", s)
}
