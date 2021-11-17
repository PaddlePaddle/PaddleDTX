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

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

type response struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func Post(ctx context.Context, url string, input io.Reader) (io.ReadCloser, error) {
	body, err := do(ctx, "POST", url, input)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func Get(ctx context.Context, url string) (io.ReadCloser, error) {
	body, err := do(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func GetResponse(ctx context.Context, url string, output interface{}) error {
	return doResponse(ctx, "GET", url, nil, output)
}

func PostResponse(ctx context.Context, url string, input io.Reader, output interface{}) error {
	return doResponse(ctx, "POST", url, input, output)
}

func doResponse(ctx context.Context, method string, url string, input io.Reader, output interface{}) error {
	body, err := do(ctx, method, url, input)
	if err != nil {
		return err
	}
	defer body.Close()

	var result response
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decode response")
	}

	if len(result.Code) > 0 {
		return errorx.New(result.Code, result.Message)
	}

	if err := json.Unmarshal(result.Data, output); err != nil {
		return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decode response data")
	}
	return nil
}

func do(ctx context.Context, method string, url string, input io.Reader) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, input)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to new request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to do request")
	}
	if resp.StatusCode != http.StatusOK {
		bs, _ := ioutil.ReadAll(resp.Body)
		code, message, success := errorx.TryParseFromString(string(bs))
		if !success {
			code = errorx.ErrCodeInternal
			message = fmt.Sprintf("http status: %d, message: %s", resp.StatusCode, string(bs))
		}
		return nil, errorx.New(code, message)
	}

	return resp.Body, nil
}
