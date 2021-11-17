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
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

const (
	defaultRootPath = "/root/xdata/data/slices"
)

// Storage stores files locally
type Storage struct {
	RootPath string
}

// New new Storage by configuration
func New(conf *config.LocalConf) (*Storage, error) {
	rootPath := conf.RootPath
	if len(rootPath) == 0 {
		rootPath = defaultRootPath
	}

	// create dir if not exist
	// only create the outer dir, for example "slices" in "/root/xdata/data/slices"
	// if "/root/xdata/data" doesn't exist, we should panic
	// because maybe the operator forgot to mount "/root/xdata/data" from host machine
	if _, err := os.Stat(rootPath); err != nil {
		if err := os.Mkdir(rootPath, 0777); err != nil {
			return nil, errorx.NewCode(err, errorx.ErrCodeConfig, "failed to mkdir for storage")
		}
	}

	storage := &Storage{
		RootPath: rootPath,
	}

	return storage, nil
}

// Save saves target to local
func (s *Storage) Save(ctx context.Context, key string, value io.Reader) error {
	if !isValidKey(key) {
		return errorx.New(errorx.ErrCodeParam, "invalid key: %s", key)
	}

	exist, err := s.Exist(key)
	if err != nil {
		return err
	}
	if exist {
		return errorx.New(errorx.ErrCodeAlreadyExists, "key already exist")
	}

	filePath := filepath.Join(s.RootPath, key)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to open file")
	}
	defer f.Close()

	if _, err := io.Copy(f, value); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to write")
	}

	return nil
}

// Load retrieves a target from local
func (s *Storage) Load(ctx context.Context, key string) (io.ReadCloser, error) {
	if !isValidKey(key) {
		return nil, errorx.New(errorx.ErrCodeParam, "invalid key: %s", key)
	}

	exist, err := s.Exist(key)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errorx.New(errorx.ErrCodeNotFound, "key not found")
	}

	filePath := filepath.Join(s.RootPath, key)
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to open file")
	}

	return f, nil
}

// Exist checks if target exists in local
func (s *Storage) Exist(key string) (bool, error) {
	if !isValidKey(key) {
		return false, errorx.New(errorx.ErrCodeParam, "invalid key: %s", key)
	}
	filePath := filepath.Join(s.RootPath, key)
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to check file")
}

// Delete delete a target from local by key
func (s *Storage) Delete(key string) (bool, error) {
	if !isValidKey(key) {
		return false, errorx.New(errorx.ErrCodeParam, "invalid key: %s", key)
	}

	filePath := filepath.Join(s.RootPath, key)
	if err := os.Remove(filePath); err != nil {
		return false, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to delete file")
	}
	return true, nil
}

// SaveAndUpdate update a traget in local
func (s *Storage) SaveAndUpdate(ctx context.Context, key, value string) error {
	filePath := filepath.Join(s.RootPath, key)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to open file")
	}
	defer f.Close()
	_, err = f.WriteString(value)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to write")
	}

	return nil
}

func (s *Storage) LoadStr(ctx context.Context, key string) (string, error) {
	exist, err := s.Exist(key)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", errorx.New(errorx.ErrCodeNotFound, "key not found")
	}

	filePath := filepath.Join(s.RootPath, key)
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return "", errorx.NewCode(err, errorx.ErrCodeInternal, "failed to open file")
	}
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return "", errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read file")
	}
	return string(content), nil
}

func isValidKey(key string) bool {
	// we know the key(slice id) is a uuid, use uuid.Parse to defend path attacking
	_, err := uuid.Parse(key)
	return err == nil
}
