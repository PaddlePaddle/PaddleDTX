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

	"github.com/sirupsen/logrus"
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

var (
	logger = logrus.WithField("module", "server")
)

// Server defines grpc server and http server
type Server struct {
	listenAddr string
	GrpcServer *grpc.Server

	httpServer *HttpServer
}

// New creates GRPC and HTTP server which has no service registered and has not
// started to accept requests yet.
func New(conf *config.ExecutorConf) (*Server, error) {
	// define grpc server
	ser := grpc.NewServer(grpc.MaxRecvMsgSize(MaxRecvMsgSize),
		grpc.MaxConcurrentStreams(MaxConcurrentStreams), grpc.ConnectionTimeout(time.Second*time.Duration(GRPCTIMEOUT)))
	server := &Server{
		listenAddr: conf.ListenAddress,
		GrpcServer: ser,
	}
	// if conf.HttpServer.Switch is "on", initialize httpserver
	// httpserver is a gateway which forwards http requests to grpc server
	if conf.HttpServer.Switch == "on" {
		httpServe, err := NewHttpServer(conf)
		if err != nil {
			return nil, err
		}
		server.httpServer = httpServe
	}
	return server, nil
}

// Serve runs Server and blocks current routine
func (s *Server) Serve(ctx context.Context) error {
	errCh := make(chan error)

	// start grpc server
	go func() {
		errCh <- s.StartGrpcServe(ctx)
	}()

	// if conf.HttpServer.Switch == "on, start http server
	if s.httpServer != nil {
		go func() {
			errCh <- s.startHttpServer(ctx)
		}()
	}

	// interrupt signal, gracefully shuts down the server
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	// monitor the running status of the server
	// return when get an error signal
	err := <-errCh
	return err
}

// startGrpcServe runs GerpcServer, block go-routine until get Stop signal
func (s *Server) StartGrpcServe(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		logger.WithError(err).Errorf("listen tcp error: %v\n", err)
		return err
	}

	if err := s.GrpcServer.Serve(lis); err != nil {
		logger.WithError(err).Errorf("failed to start grpc serve: %v\n", err)
		return err
	}
	return ctx.Err()
}

// startHttpServer runs httpServer and blocks current routine
func (s *Server) startHttpServer(ctx context.Context) error {
	if err := s.httpServer.Serve(); err != nil {
		logger.WithError(err).Errorf("failed to start http serve: %v\n", err)
		return err
	}
	return ctx.Err()
}

// Stop when get interrupt signal, stop grpc server and http server
func (s *Server) Stop() {
	if s.GrpcServer != nil {
		s.GrpcServer.Stop()
	}

	if s.httpServer != nil {
		s.httpServer.Stop()
	}
}
