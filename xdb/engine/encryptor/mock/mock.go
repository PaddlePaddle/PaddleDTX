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

package mock

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/crypto/hash"
)

type MockEncrypter struct {
}

func New() *MockEncrypter {
	return &MockEncrypter{}
}

func (m *MockEncrypter) Encrypt(ctx context.Context, r io.Reader, opt *encryptor.EncryptOptions) (
	encryptor.EncryptedSlice, error) {

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return encryptor.EncryptedSlice{}, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read")
	}

	// fake encrypt
	for i := range data {
		data[i] += 1
	}

	h := hash.Hash(data)

	es := encryptor.EncryptedSlice{
		EncryptedSliceMeta: encryptor.EncryptedSliceMeta{
			SliceID:    opt.SliceID,
			NodeID:     opt.NodeID,
			CipherHash: h,
			Length:     uint64(len(data)),
		},

		CipherText: data,
	}

	return es, nil
}

func (m *MockEncrypter) Recover(ctx context.Context, r io.Reader, opt *encryptor.RecoverOptions) (
	[]byte, error) {

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read")
	}

	// fake recover
	for i := range data {
		data[i] -= 1
	}

	return data, nil
}
