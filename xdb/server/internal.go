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

package server

import (
	"io"
	"net/http"

	"github.com/kataras/iris/v12"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

func responseError(ctx iris.Context, err error) {
	logrus.WithError(err).Warn("error from server")

	ctx.StatusCode(http.StatusInternalServerError)
	ctx.JSON(errorx.ParseAndWrap(err, "from xdb api"))
}

type response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func responseJSON(ctx iris.Context, o interface{}) {
	ctx.StatusCode(http.StatusOK)
	ctx.JSON(response{
		Data: o,
	})
}

func responseBytes(ctx iris.Context, bs []byte) {
	ctx.StatusCode(http.StatusOK)
	ctx.Binary(bs)
}

func responseStream(ctx iris.Context, r io.Reader) {
	// check first byte in case of error
	firstByte := make([]byte, 1)
	if n, err := r.Read(firstByte); err != nil {
		responseError(ctx, err)
		return
	} else if n == 0 {
		return
	}

	ctx.StatusCode(http.StatusOK)
	ctx.ResponseWriter().Write(firstByte)
	io.Copy(ctx.ResponseWriter(), r)
}
