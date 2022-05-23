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
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/storage"
)

var (
	logger = logrus.WithField("module", "storage.ipfs")
)

// Storage is implemented with IPFS cluster.
type Storage struct {
	gw Gateway
}

// New creates Storage with ipfs hosts and timeout(in milliseconds)
// returns error if any mistake occured
func New(hosts []string, timeout time.Duration) (storage.BasicStorage, error) {
	gw, err := NewGateway(hosts, timeout)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeConfig, "failed to create gateway for IPFS")
	}
	s := &Storage{
		gw: gw,
	}
	logger.WithFields(logrus.Fields{
		"hosts":   hosts,
		"timeout": timeout,
	}).Info("new storage")
	return s, nil
}

// Save saves a piece of `Data`
// key is the identification of a piece of `Data`, and it's decided by end-users
// value contains content of a piece of `Data`
// returns index, which is the index of stored data(cid in IPFS)
func (s *Storage) Save(key string, value io.Reader) (string, error) {
	cid, err := s.gw.Add(key, value)
	if err != nil {
		return "", errorx.NewCode(err, errorx.ErrCodeInternal, "failed to save data")
	}

	logger.WithFields(logrus.Fields{
		"index": cid,
		"key":   key,
	}).Debug("successfully saved")
	return cid, nil
}

// Load loads a piece of `Data`
// key is the identification of a piece of `Data`, and it's decided by end-users
// index is the index of stored data(cid in IPFS)
func (s *Storage) Load(key string, index string) (io.ReadCloser, error) {
	reader, err := s.gw.Cat(key, index)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to load data")
	}

	logger.WithFields(logrus.Fields{
		"index": index,
		"key":   key,
	}).Debug("successfully loaded")
	return reader, nil
}

// Exist checks existence of a piece of `Data`
// key is the identification of a piece of `Data`, and it's decided by end-users
// index is the index of stored data(cid in IPFS)
func (s *Storage) Exist(key string, index string) (bool, error) {
	pins, err := s.gw.PinLs(key, index)
	if err != nil {
		return false, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to determine data is existing or not")
	}

	if len(pins) == 0 {
		return false, nil
	}
	return true, nil
}

// Delete deletes a piece of `Data`
// key is the identification of a piece of `Data`, and it's decided by end-users
// index is the index of stored data(cid in IPFS)
// It takes a while for the data to really be wiped
func (s *Storage) Delete(key string, index string) error {
	err := s.gw.Unpin(key, index)
	if err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to delete data")
	}

	logger.WithFields(logrus.Fields{
		"index": index,
		"key":   key,
	}).Debug("successfully deleted")
	return nil
}

// Update updates a piece of `Data`
// key is the identification of a piece of `Data`, and it's decided by end-users
// index is the index of stored data(cid in IPFS)
// returns new index
func (s *Storage) Update(key string, index string, value io.Reader) (string, error) {
	// delete original data and ignore errors（always caused by non-existance）
	_ = s.Delete(key, index)

	newIndex, err := s.Save(key, value)
	if err != nil {
		return "", err
	}

	logger.WithFields(logrus.Fields{
		"index":     index,
		"new index": newIndex,
		"key":       key,
	}).Debug("successfully updated")
	return newIndex, nil
}
