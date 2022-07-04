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

// define success code
const SuccessCode = "0"

// error code list
const (
	ErrCodeInternal = "10001" // internal error
	ErrCodeParam    = "10002" // parameter error
	ErrCodeConfig   = "10003" // configuration error
	ErrCodeNotFound = "10004" // target not found
	ErrCodeEncoding = "10005" // encoding error

	ErrCodeNotAuthorized = "10006" // not authorized
	ErrCodeAlreadyExists = "10007" // duplicate item
	ErrCodeBadSignature  = "10008" // signature verification failed
	ErrCodeCrypto        = "10009" // cryptography computation error
	ErrCodeExpired       = "10010" // target expired

	ErrCodeReadBlockchain  = "10011" // errors occurred when reading data from blockchain
	ErrCodeWriteBlockchain = "10012" // errors occurred when writing data to blockchain
	ErrCodeAlreadyUpdate   = "10013" // duplicate updating error
)
