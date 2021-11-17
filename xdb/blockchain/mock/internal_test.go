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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackChallengeFilter(t *testing.T) {
	user1 := []byte{1}
	user2 := []byte{2}

	// both owner and target
	f := packChallengeFilter(user1, user2)
	require.Equal(t, f, fmt.Sprintf("%s/%x/%x", prefixChallengeIndex4Owner, user1, user2))

	// only owner
	f = packChallengeFilter(user1, nil)
	require.Equal(t, f, fmt.Sprintf("%s/%x", prefixChallengeIndex4Owner, user1))

	// only target
	f = packChallengeFilter(nil, user2)
	require.Equal(t, f, fmt.Sprintf("%s/%x", prefixChallengeIndex4Target, user2))

	// both nil
	f = packChallengeFilter(nil, nil)
	require.Equal(t, f, prefixChallengeIndex4Owner)
}
