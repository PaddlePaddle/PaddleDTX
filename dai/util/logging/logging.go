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

package logging

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
)

// Logging maintains the state associated with the PaddleDTX logging system
type Logging struct {
	// Format is the log record format specifier for the Logging instance
	// If Format is not provided, a default format that provides basic information will
	// be used.
	Format *logrus.TextFormatter

	// logrus log level, default info
	Level logrus.Level

	// Writer is the sink for encoded and formatted log records.
	Writer io.Writer
}

const (
	TimeFormat   = "2006-01-02 15:04:05"
	DefaultLevel = logrus.InfoLevel
)

// InitLog initiates Logging instance.
func InitLog(conf *config.Log, fileName string, isSetFormat bool) (*Logging, error) {
	logging := &Logging{}
	logPath, level, err := logging.checkLogConf(conf)
	if err != nil {
		return nil, errorx.Wrap(err, "check log conf error")
	}
	logging.Level = level

	writer, err := logging.writer(logPath, fileName)
	if err != nil {
		return nil, errorx.Wrap(err, "get log writer error")
	}
	logging.Writer = writer

	if isSetFormat {
		logging.Format = &logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: TimeFormat,
		}
	}
	return logging, nil
}

// writer satisfies the io.Write contract. It delegates to the writer argument
// of SetWriter or the Writer field of Config. The Core uses this when encoding
// log records.
func (l *Logging) writer(logPath, fileName string) (io.Writer, error) {
	logFileName := filepath.Join(logPath, fileName)

	// Generate a soft chain and point to the latest log file
	// Keep log files for 30 days
	// Log cutting interval 1 hour
	logStd, err := rotatelogs.New(
		logFileName+".%Y%m%d%H",
		rotatelogs.WithLinkName(logFileName),
		rotatelogs.WithMaxAge(720*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)

	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "new rotatelogs error")
	}
	return logStd, nil
}

// checkLogConf is used to verify the log configuration in the configuration file
// Level is used to log the extremely detailed message. If the level name is
// empty, default level is info
func (l *Logging) checkLogConf(conf *config.Log) (string, logrus.Level, error) {
	path := conf.Path
	if len(path) == 0 {
		return "", 0, errorx.New(errorx.ErrCodeConfig, "missing config: log.path")
	}

	if strings.LastIndex(path, "/") != len([]rune(path))-1 {
		path = filepath.Join(path, "/")
	}
	// create if the log file directory does not exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			return "", 0, errorx.New(errorx.ErrCodeConfig, "mkdir logs error, err :%v", err)
		}
	}
	var level logrus.Level
	level, err := logrus.ParseLevel(conf.Level)
	if err != nil {
		level = DefaultLevel
	}

	return path, level, nil
}
