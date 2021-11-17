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

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/engine"
	pbTask "github.com/PaddlePaddle/PaddleDTX/dai/protos/task"
	"github.com/PaddlePaddle/PaddleDTX/dai/server"
	"github.com/PaddlePaddle/PaddleDTX/dai/util/logging"
)

// init reads config file
func init() {
	err := config.InitConfig("conf/config.toml")
	if err != nil {
		appExit(err)
	}

	logConf := config.GetLogConf()
	logStd, err := logging.InitLog(logConf, "executor.log", true)
	if err != nil {
		appExit(err)
	}
	// writes the standard output to the log file
	logrus.SetOutput(logStd.Writer)
	logrus.SetLevel(logStd.Level)
	logrus.SetFormatter(logStd.Format)
}

// main is where execution of the program begins
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-quit
		cancel()
	}()

	executorConf := config.GetExecutorConf()
	taskEngine, err := engine.NewEngine(executorConf)
	if err != nil {
		appExit(err)
	}

	// start engine
	if err := taskEngine.Start(ctx); err != nil {
		appExit(err)
	}
	defer taskEngine.Close()

	srv, err := server.New(executorConf)
	if err != nil {
		logrus.WithError(err).Error("failed to initiate server")
		cancel()
	} else {
		// register executor service to gRPC server.
		pbTask.RegisterTaskServer(srv.Server, taskEngine)

		// register MPC service
		taskEngine.GetMpcService().RegisterClusterServer(srv.Server)

		// start server
		if err := srv.Serve(ctx); err != nil && err != context.Canceled {
			logrus.WithError(err).Error("failed to start server")
			cancel()
		}
	}
}

// appExit quits main function when an exception occurs
func appExit(err error) {
	logrus.WithError(err).Error("server exits")
	os.Exit(-1)
}
