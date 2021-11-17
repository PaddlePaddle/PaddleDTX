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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	// new
	err := New(ErrCodeEncoding, "123")
	code, message := Parse(err)
	require.Equal(t, code, ErrCodeEncoding)
	require.Equal(t, message, "123")

	// wrap
	err2 := Wrap(err, "456")
	code, message = Parse(err2)
	require.Equal(t, code, ErrCodeEncoding)
	require.Equal(t, message, "123")

	// json
	err3 := errors.New(err.Error())
	code, message = Parse(err3)
	require.Equal(t, code, ErrCodeEncoding)
	require.Equal(t, message, "123")

	// new code
	err4 := NewCode(err, ErrCodeEncoding, "xxx")
	code, _ = Parse(err4)
	require.Equal(t, code, ErrCodeEncoding)
	var errOfThisPackage *Error
	require.True(t, errors.As(err4, &errOfThisPackage))

	// is
	err5 := Wrap(ErrNotFound, "")
	require.True(t, errors.Is(err5, ErrNotFound))

	// wrap outer error
	{
		err := errors.New(`{"code":"XXX","message":"123"}`)
		nErr := ParseAndWrap(err, "my message")
		code, _ := Parse(nErr)
		require.Equal(t, code, "XXX")
	}

	Example()
}
