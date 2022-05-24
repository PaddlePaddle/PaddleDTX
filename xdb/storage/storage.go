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

package storage

import (
	"io"
	"io/ioutil"
)

// BasicStorage is an abstraction used to refer to any underlying system or device
// that XuperDB will store its data to.
// key is the identification of a piece of `Data`, and it's decided by end-users
// index is the index of stored data, and it's created by `BasicStorage`
// value contains content of a piece of `Data`
type BasicStorage interface {
	// Save saves a piece of `Data`
	Save(key string, value io.Reader) (string, error)

	// Load loads a piece of `Data`
	Load(key string, index string) (io.ReadCloser, error)

	// Exist checks existence of a piece of `Data`
	Exist(key string, index string) (bool, error)

	// Delete deletes a piece of `Data`
	Delete(key string, index string) error

	// Update updates a piece of `Data`
	Update(key string, index string, value io.Reader) (string, error)
}

type Storage interface {
	BasicStorage

	//LoadStr loads a piece of `Data`, and convert it to a string
	LoadStr(key string, index string) (string, error)
}

type storage struct {
	BasicStorage
}

func (s *storage) LoadStr(key string, index string) (string, error) {
	f, err := s.Load(key, index)
	if err != nil {
		return "", err
	}
	defer f.Close()

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func NewStorage(s BasicStorage) Storage {
	return &storage{s}
}
