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

package encryptor

// EncryptOptions use fileID, sliceID and nodeID info when encrypting slice content
type EncryptOptions struct {
	FileID  string
	SliceID string
	NodeID  []byte
}

type EncryptedSliceMeta struct {
	SliceID    string
	NodeID     []byte
	CipherHash []byte // hash of slice ciphertext
	Length     uint64 // length of slice ciphertext
}

type EncryptedSlice struct {
	EncryptedSliceMeta

	CipherText []byte // slice ciphertext
}

// RecoverOptions use fileID, sliceID and nodeID info when recovering slice content
type RecoverOptions struct {
	FileID  string
	SliceID string
	NodeID  []byte
}
