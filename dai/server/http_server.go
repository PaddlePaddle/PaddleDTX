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
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
)

const (
	// InitialWindowSize window size, default 128 MB.
	InitialWindowSize int32 = 128 << 20
	// InitialConnWindowSize connection window size, default 64 MB.
	InitialConnWindowSize int32 = 64 << 20
	// ReadBufferSize buffer size, default 32 MB.
	ReadBufferSize = 32 << 20
	// WriteBufferSize write buffer size, default 32 MB.
	WriteBufferSize = 32 << 20
)

// response defines the return format of the http requests
type response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// HttpServer defines gRPC-Gateway for forwarding http requests
type HttpServer struct {
	server      *http.Server
	rpcEndpoint string
	httpPort    string
	allowCROS   bool
}

// NewHttpServer initiates gRPC-Gateway, allowCROS is used to determine whether to allow cross-domain requests
func NewHttpServer(conf *config.ExecutorConf) (*HttpServer, error) {
	if conf.HttpServer.HttpPort == "" || conf.PublicAddress == "" {
		return nil, errorx.New(errorx.ErrCodeConfig,
			"invalid httpserver config, httpPort or publicAddress can not be empty")
	}
	ser := &HttpServer{
		rpcEndpoint: conf.PublicAddress,
		httpPort:    conf.HttpServer.HttpPort,
		allowCROS:   conf.HttpServer.AllowCros,
	}

	return ser, nil
}

// Serve starts the gateway and blocks
func (s *HttpServer) Serve() error {
	err := s.runHttpServer()
	if err != nil {
		logger.WithError(err).Error("failed to start http server")
		return err
	}
	return nil
}

// httpSuccHandler used to rewrite resp body from gRPC Success Response
func httpSuccHandler(ctx context.Context, w http.ResponseWriter, p proto.Message) error {
	resp := response{
		Code: errorx.SuccessCode,
		Data: p,
	}
	bs, _ := json.Marshal(&resp)
	return errorx.New(errorx.SuccessCode, string(bs))
}

// httpErrorHandler used to rewrite resp body from gRPC Error Response
func httpErrorHandler(ctx context.Context, mux *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-type", m.ContentType())
	// parse error
	code, message := errorx.Parse(err)
	if code == errorx.SuccessCode {
		w.Write([]byte(message))
		return
	}
	// handler the error message returned by the contract
	indexStart := strings.Index(err.Error(), "{")
	indexStop := strings.Index(err.Error(), "}")
	if indexStart > 0 && indexStop > 0 && indexStop >= indexStart {
		errMes := err.Error()[indexStart : indexStop+1]
		if len(errMes) > 0 {
			if c, m, ok := errorx.TryParseFromString(errMes); ok {
				code = c
				message = m
			}
		}
	}
	resp := response{
		Code:    code,
		Data:    nil,
		Message: message,
	}
	bs, _ := json.Marshal(&resp)
	w.Write(bs)
}

// runHttpServer registers the http handlers and starts
// rpcEndpoint is the address of the server to be proxied
func (s *HttpServer) runHttpServer() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(httpSuccHandler),
		runtime.WithProtoErrorHandler(httpErrorHandler),
	)
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithInitialWindowSize(InitialWindowSize),
		grpc.WithWriteBufferSize(WriteBufferSize),
		grpc.WithInitialConnWindowSize(InitialConnWindowSize),
		grpc.WithReadBufferSize(ReadBufferSize),
	}
	// register the http handlers for service Task
	err := pbTask.RegisterTaskHandlerFromEndpoint(ctx, mux, s.rpcEndpoint, opts)
	if err != nil {
		return err
	}
	// listen on the port and start the httpServer
	s.server = &http.Server{
		Addr:    s.httpPort,
		Handler: s.handler(mux),
	}
	if err = s.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop exits the gateway service
func (s *HttpServer) Stop() {
	if s.server != nil {
		s.server.Shutdown(context.Background())
	}
}

// handler defines http request handler
func (s *HttpServer) handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// allow CROS requests
		// Note: CROS is kind of dangerous in production environment
		// don't use this without consideration
		if s.allowCROS {
			if origin := r.Header.Get("Origin"); origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
					s.preflightHandler(w, r)
					return
				}
			}
		}

		h.ServeHTTP(w, r)

		// Request log
		logger.Infof("http server access request, ip: %v, method: %v, url: %v", r.RemoteAddr, r.Method, r.URL.Path)
	})
}

// preflightHandler handles browser-initiated' OPTIONS preflight requests
// The request returns the browser whether the server allows cross-domain requests
func (s *HttpServer) preflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{"Content-Type", "Accept"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
}
