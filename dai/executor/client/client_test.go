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

package client

import (
	"context"
	"testing"
)

var (
	// server grpc address of the executor node, example '127.0.0.1:8184'
	executorNode = "127.0.0.1:8184"
	rPubkeyStr   = "b826561293479b01b194c1ac271331d7f9e22a365c82faeb9922a348e56492f644fd4b357ed581af677ce5ac05b6c8713f88d8a26279f761ea52317cb1180f71"
	ePubkeyStr   = "6a5ba56bccf843c591a3a32baa5aa76deebffe2695d48521799b77fb0a32e286ae493143560d1f548dd494bf266d4df39375f755f6008e8db7444cd8a96258c6"
)

func TestGetTaskById(t *testing.T) {
	client, err := GetExecutorClient(executorNode)
	checkErr(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	task, err := client.GetTaskById(ctx, "1")
	checkErr(t, err)
	cancel()
	t.Logf("task: %v\n", task)
}

func TestListTask(t *testing.T) {
	client, err := GetExecutorClient(executorNode)
	checkErr(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	tasks, err := client.ListTask(ctx, rPubkeyStr, ePubkeyStr, "Finished", 1, 5, 3)
	checkErr(t, err)
	cancel()
	t.Logf("tasks: %v\n", tasks.String())
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
