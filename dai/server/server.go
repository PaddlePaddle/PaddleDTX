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
	"context"
	"net"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"google.golang.org/grpc"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
)

const (
	// MaxRecvMsgSize max message size
	MaxRecvMsgSize = 1024 * 1024 * 1024
	// MaxConcurrentStreams max concurrent
	MaxConcurrentStreams = 1000
	// GRPCTIMEOUT grpc timeout
	GRPCTIMEOUT = 20
)

// Server defines grpc server
type Server struct {
	listenAddr string
	Server     *grpc.Server
}

// New creates a gRPC server which has no service registered and has not
// started to accept requests yet.
func New(conf *config.ExecutorConf) (*Server, error) {
	ser := grpc.NewServer(grpc.MaxRecvMsgSize(MaxRecvMsgSize),
		grpc.MaxConcurrentStreams(MaxConcurrentStreams), grpc.ConnectionTimeout(time.Second*time.Duration(GRPCTIMEOUT)))
	server := &Server{
		listenAddr: conf.ListenAddress,
		Server:     ser,
	}
	return server, nil
}

// Serve runs Server and blocks current routine
func (s *Server) Serve(ctx context.Context) error {
	// interrupt signal
	go func() {
		<-ctx.Done()
		s.Server.Stop()
	}()

	// start grpc server
	lis, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return errorx.Wrap(err, "StartServer failed")
	}

	// Server.Serve() block go-routine until get Stop signal
	if err := s.Server.Serve(lis); err != nil {
		return err
	}
	return ctx.Err()
}
