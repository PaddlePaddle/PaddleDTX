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
	"encoding/hex"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"google.golang.org/grpc"

	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
)

type Client struct {
	executorClient pbTask.TaskClient
	conn           *grpc.ClientConn
}

// GetExecutorClient returns executor client
func GetExecutorClient(executorPort string) (*Client, error) {
	// check blockchain config yaml
	conn, err := grpc.Dial(executorPort, grpc.WithInsecure())
	if err != nil {
		return nil, errorx.Wrap(err, "CAN_NOT_CONNECT_EXECUTOR_SERVER")
	}
	taskClient := pbTask.NewTaskClient(conn)

	return &Client{executorClient: taskClient, conn: conn}, nil
}

// GetTaskById gets task by id through executor server
func (c *Client) GetTaskById(ctx context.Context, id string) (*pbTask.FLTask, error) {
	if c.conn != nil {
		defer c.conn.Close()
	}

	in := &pbTask.GetTaskRequest{
		TaskID: id,
	}

	out, err := c.executorClient.GetTaskById(ctx, in)

	if err != nil {
		return &pbTask.FLTask{}, err
	}
	return out, nil
}

// ListTask lists tasks by requester or executor's public key hex string
// support listing tasks a requester published or tasks an executor involved
// status is task status to search
// only task published after "start" before "end" will be listed
// limit is the maximum number of tasks to response
func (c *Client) ListTask(ctx context.Context, rPubkeyStr, ePubkeyStr, status string, start, end,
	limit int64) (ts *pbTask.FLTasks, err error) {
	if c.conn != nil {
		defer c.conn.Close()
	}

	// check requester public key
	rPubkey, err := hex.DecodeString(rPubkeyStr)
	if err != nil {
		return ts, errorx.Wrap(err, "failed to decode requester public key")
	}
	// check executor public key
	ePubkey, err := hex.DecodeString(ePubkeyStr)
	if err != nil {
		return ts, errorx.Wrap(err, "failed to decode executor public key")
	}
	in := &pbTask.ListTaskRequest{
		PubKey:    rPubkey[:],
		EPubKey:   ePubkey[:],
		TimeStart: start,
		TimeEnd:   end,
		Status:    status,
		Limit:     limit,
	}

	ts, err = c.executorClient.ListTask(ctx, in)
	if err != nil {
		return ts, err
	}

	return ts, nil
}
