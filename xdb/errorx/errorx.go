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

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Error error of tc-worker
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	s, _ := json.Marshal(e)
	return string(s)
}

// New Create an error from code and message
func New(code, format string, args ...interface{}) error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// NewCode Create an error by wrapping a existed error from outer package
//  will drop original code
func NewCode(err error, code, format string, args ...interface{}) error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...) + ": " + err.Error(),
	}
}

// Wrap wrap error with a message
func Wrap(err error, format string, args ...interface{}) error {
	m := fmt.Sprintf(format, args...)
	return fmt.Errorf(m+": %w", err)
}

// ParseAndWrap decode and wrap errors from outer components
func ParseAndWrap(err error, format string, args ...interface{}) error {
	code, message := Parse(err)
	newMsg := fmt.Sprintf(format, args...)
	return New(code, newMsg+": %s", message)
}

// Is compare if two codes are same
func Is(err error, code string) bool {
	c, _ := Parse(err)
	return c == code
}

// Parse retrieve code and message
// 1. try to retrieve errorx.Error from err chain
// 2. try to unmarshal err.Error() into errorx.Error
// 3. or else an internal error with original err.Error()
func Parse(err error) (code, message string) {
	var newErr *Error
	if errors.As(err, &newErr) {
		return newErr.Code, newErr.Message
	}

	var e Error
	if nerr := json.Unmarshal([]byte(err.Error()), &e); nerr == nil && len(e.Code) > 0 {
		return e.Code, e.Message
	}

	return ErrCodeInternal, err.Error()
}

func TryParseFromString(s string) (code, message string, success bool) {
	var e Error
	if nerr := json.Unmarshal([]byte(s), &e); nerr == nil && len(e.Code) > 0 {
		return e.Code, e.Message, true
	}

	return "", "", false
}

func Internal(err error, format string, args ...interface{}) error {
	if err != nil {
		return NewCode(err, ErrCodeInternal, format, args...)
	}
	return New(ErrCodeInternal, format, args...)
}

// Example noop
func Example() {
	err := &Error{
		Code:    ErrCodeInternal,
		Message: "example error",
	}

	{
		nErr := Wrap(err, "this is a message")
		_ = nErr.Error()
	}

	{
		nErr := NewCode(err, ErrCodeInternal, "this is a error with new code")
		_ = nErr.Error()
	}

	{
		raw := errors.New(err.Error())
		nErr := ParseAndWrap(raw, "this is an error from outer components")
		_ = nErr.Error()
	}
}
