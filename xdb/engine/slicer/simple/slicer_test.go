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

package simple

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
)

func TestSimpleSlice(t *testing.T) {
	logrus.SetLevel(logrus.ErrorLevel)

	data := make([]byte, 23)
	r := bytes.NewReader(data)

	s, err := New(&config.SimpleSlicerConf{
		BlockSize: 10,
	})
	require.NoError(t, err)

	resCh := s.Slice(context.TODO(), r, &slicer.SliceOptions{}, func(err error) {
		require.NoError(t, err)
	})
	for i := 0; i < 4; i++ {
		s, ok := <-resCh

		switch i {
		case 0:
			require.Equal(t, 10, len(s.Data))
		case 1:
			require.Equal(t, 10, len(s.Data))
		case 2:
			require.Equal(t, 10, len(s.Data))
		case 3:
			require.Equal(t, false, ok)
		}
	}
}
