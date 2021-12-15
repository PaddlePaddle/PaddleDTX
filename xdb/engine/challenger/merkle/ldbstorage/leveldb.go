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

package ldbstorage

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

const (
	dbName = "materialDB"
)

type LevelDBStorage struct {
	root string
	db   *leveldb.DB
}

// New creat a levelDB to save challenge material
func New(root string) (*LevelDBStorage, error) {
	f := filepath.Join(root, dbName)
	db, err := leveldb.OpenFile(f, nil)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "cannot open leveldb")
	}

	ldb := &LevelDBStorage{
		root: root,
		db:   db,
	}

	return ldb, nil
}

// Save save a challenge material to levelDB
func (s *LevelDBStorage) Save(cms []types.Material) error {
	batch := leveldb.Batch{}
	ctime := time.Now().UnixNano()
	for _, m := range cms {
		key := makeMaterialKey(m.FileID, m.SliceID, m.NodeID, ctime)
		value, err := json.Marshal(m.Ranges)
		if err != nil {
			return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal ranges")
		}
		batch.Put(key, value)
	}
	if err := s.db.Write(&batch, nil); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to write batch")
	}

	return nil
}

// Update update challenge material by key
func (s *LevelDBStorage) Update(m types.Material, key []byte) error {
	batch := leveldb.Batch{}

	value, err := json.Marshal(m.Ranges)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to marshal ranges")
	}
	batch.Put(key, value)

	if err := s.db.Write(&batch, nil); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to write batch")
	}

	return nil
}

func (s *LevelDBStorage) NewIterator(prefix []byte) ([][]byte, error) {
	iter := s.db.NewIterator(util.BytesPrefix(prefix), nil)
	var keyList [][]byte
	for iter.Next() {
		keyList = append(keyList, iter.Key())
	}
	iter.Release()
	err := iter.Error()

	return keyList, err
}

// Load get a challenge material from levelDB
func (s *LevelDBStorage) Load(key []byte) (types.Material, error) {
	value, err := s.db.Get(key, nil)
	if err != nil {
		return types.Material{}, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to get")
	}

	m := types.Material{}
	if err := json.Unmarshal(value, &m.Ranges); err != nil {
		return m, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to unmarshal ranges")
	}

	return m, nil
}

func (s *LevelDBStorage) Close() {
	s.db.Close()
}

func makeMaterialKey(fileID, sliceID string, nodeID []byte, ctime int64) []byte {
	return []byte(fmt.Sprintf("%s:%s:%x:%d", fileID, sliceID, nodeID, ctime))
}
