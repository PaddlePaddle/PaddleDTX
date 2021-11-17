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

package errorx

// error code list
const (
	ErrCodeInternal = "XDAT0001" // internal error
	ErrCodeParam    = "XDAT0002" // parameter error
	ErrCodeConfig   = "XDAT0003" // configuration error
	ErrCodeNotFound = "XDAT0004" // target not found
	ErrCodeEncoding = "XDAT0005" // encoding error

	ErrCodeNotAuthorized = "XDAT0006" // not authorized
	ErrCodeAlreadyExists = "XDAT0007" // duplicate item
	ErrCodeBadSignature  = "XDAT0008" // signature verification failed
	ErrCodeCrypto        = "XDAT0009" // cryptography computation error
	ErrCodeExpired       = "XDAT0010" // target expired

	ErrCodeReadBlockchain  = "XDAT0011" // errors occurred when reading data from blockchain
	ErrCodeWriteBlockchain = "XDAT0012" // errors occurred when writing data to blockchain
	ErrCodeAlreadyUpdate   = "XDAT0013" // duplicate updating error
)
