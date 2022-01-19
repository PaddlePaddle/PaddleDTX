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
	"io"
	"os"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	localstorage "github.com/PaddlePaddle/PaddleDTX/xdb/storage/local"
)

// Storage stores files locally
type Storage struct {
	localstorage.Storage
}

// New initiates Storage
func New(rootPath string) (*Storage, error) {
	// create dir if not exist
	// only create the outer dir, for example "models" in "/root/train/models"
	// if "/root/train/models" does not exist, we should panic
	// because maybe the operator forgot to mount "/root/train/models" from host machine
	if _, err := os.Stat(rootPath); err != nil {
		if err := os.Mkdir(rootPath, 0777); err != nil {
			return nil, errorx.NewCode(err, errorx.ErrCodeConfig, "failed to mkdir for storage")
		}
	}

	storage := &Storage{
		Storage: localstorage.Storage{
			RootPath: rootPath,
		},
	}
	return storage, nil
}

// Write writes target to local
func (s *Storage) Write(value io.Reader, key string) (string, error) {
	if err := s.Save(key, value); err != nil {
		return "", err
	}
	return "", nil
}

// Read retrieves a target from local
func (s *Storage) Read(key string) (io.ReadCloser, error) {
	return s.Load(key)
}
