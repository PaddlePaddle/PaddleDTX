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
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	fabricblockchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain/fabric"
	xchainblockchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain/xchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine"
	merklechallenger "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle"
	pairingchallenger "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/pairing"
	randomcopier "github.com/PaddlePaddle/PaddleDTX/xdb/engine/copier/random"
	softencryptor "github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor/soft"
	simpleslicer "github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer/simple"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/peer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/server"
	localstorage "github.com/PaddlePaddle/PaddleDTX/xdb/storage/local"
)

var (
	configPath string
)

func appExit(err error) {
	logrus.WithError(err).Error("app exit")
	os.Exit(-1)
}

func init() {
	flag.StringVarP(&configPath, "conf", "c", "conf/config.toml",
		"path of the configuration file")
	flag.Parse()

	logrus.SetLevel(logrus.DebugLevel)

	config.InitConfig(configPath)
}

// main the function where execution of the program begins
func main() {

	logStd, level := initLog(config.GetLogConf())
	logrus.SetOutput(logStd)
	logrus.SetLevel(level)

	ctx, cancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-quit
		logrus.Info("stopping ...")
		cancel()
	}()

	serverConf := config.GetServerConf()
	blockchainConf := config.GetBlockchainConf()
	localNode := mustGetNode(serverConf)
	blockchainEngine := mustGetBlockchain(blockchainConf)
	var e *engine.Engine
	switch config.GetServerType() {
	case config.NodeTypeDataOwner:
		e = getDataOwnerEngine(localNode, blockchainEngine, config.GetDataOwnerConf())
	case config.NodeTypeStorage:
		e = getStorageEngine(localNode, blockchainEngine, config.GetStorageConf())
	default:
		appExit(errors.New("error server type"))
	}

	if err := e.Start(ctx); err != nil {
		appExit(err)
	}
	defer e.Close()

	// start http server
	if srv, err := server.New(serverConf.ListenAddress, e); err != nil {
		logrus.WithError(err).Error("failed to initiate server")
		cancel()
	} else {
		if err := srv.Serve(ctx); err != nil && err != context.Canceled {
			logrus.WithError(err).Error("failed to start server")
			cancel()
		}
	}
}

// getDataOwnerEngine initiates DataOwner Engine.
func getDataOwnerEngine(localNode peer.Local, blockchain engine.Blockchain, conf *config.DataOwnerConf) *engine.Engine {
	engineOption := engine.NewEngineOption{
		LocalNode: localNode,
		Chain:     blockchain,
	}
	engineOption.Slicer = mustGetSlicer(conf.Slicer)
	engineOption.Encryptor = mustGetEncryptor(conf.Encryptor)
	engineOption.Challenger = mustGetChallenger(conf.Challenger, localNode.PrivateKey)
	engineOption.Copier = mustGetCopier(conf.Copier, localNode.PrivateKey)
	engine, err := engine.NewEngine(conf.Monitor, &engineOption)
	if err != nil {
		appExit(err)
	}

	return engine
}

// getStorageEngine initiates Storage Engine.
func getStorageEngine(localNode peer.Local, blockchain engine.Blockchain, conf *config.StorageConf) *engine.Engine {
	engineOption := engine.NewEngineOption{
		LocalNode: localNode,
		Chain:     blockchain,
	}
	engineOption.Storage = mustGetStorage(conf)
	engine, err := engine.NewEngine(conf.Monitor, &engineOption)
	if err != nil {
		appExit(err)
	}
	return engine
}

// mustGetSlicer initiates Slicer, and its main work is to cut the data into blocks
func mustGetSlicer(conf *config.DataOwnerSlicerConf) engine.Slicer {
	var s engine.Slicer
	switch conf.Type {
	case "simpleSlicer":
		simpleSlicer, err := simpleslicer.New(conf.SimpleSlicer)
		if err != nil {
			appExit(err)
		}
		s = simpleSlicer
	default:
		appExit(errors.New("invalid slicer type: " + conf.Type))
	}
	return s
}

// mustGetEncryptor initiates Encryptor to encrypt data or decrypt encoded data
func mustGetEncryptor(conf *config.DataOwnerEncryptorConf) engine.Encryptor {
	var err error
	var e engine.Encryptor
	switch conf.Type {
	case "softEncryptor":
		e, err = softencryptor.New(conf.SoftEncryptor)
	default:
		appExit(errors.New("invalid encryptor type: " + conf.Type))
	}

	if err != nil {
		appExit(errorx.Wrap(err, "failed to create encryptor"))
	}

	return e
}

// mustGetChallenger initiates Challenger
//  Pairing-based and MerkleTree-based are both supported
//  see more from engine.challenger
func mustGetChallenger(conf *config.DataOwnerChallenger, signer ecdsa.PrivateKey) engine.Challenger {
	var err error
	var c engine.Challenger
	switch conf.Type {
	case "pairing":
		c, err = pairingchallenger.New(conf.Pairing, signer)
	case "merkle":
		c, err = merklechallenger.New(conf.Merkle, signer)
	default:
		appExit(errors.New("invalid challenger type: " + conf.Type))
	}

	if err != nil {
		appExit(errorx.Wrap(err, "failed to create challenger"))
	}

	return c
}

// mustGetBlockchain initiates XChain client which is used for connecting and requesting blockchain
// XChain and Fabric are both supported
func mustGetBlockchain(conf *config.BlockchainConf) engine.Blockchain {
	var b engine.Blockchain
	var err error
	switch conf.Type {
	case "xchain":
		b, err = xchainblockchain.New(conf.Xchain)
	case "fabric":
		b, err = fabricblockchain.New(conf.Fabric)
	default:
		appExit(errors.New("invalid blockchain type: " + conf.Type))
	}
	if err != nil {
		appExit(err)
	}
	return b
}

// mustGetCopier initiates Copier,
// and see more from engine.copier
func mustGetCopier(conf *config.DataOwnerCopierConf, signer ecdsa.PrivateKey) engine.Copier {
	var c engine.Copier
	copierType := conf.Type
	switch copierType {
	case "random-copier":
		c = randomcopier.New(signer)
	default:
		appExit(errors.New("invalid copier type: " + copierType))
	}

	return c
}

// mustGetStorage initiates local storage
func mustGetStorage(conf *config.StorageConf) engine.Storage {

	var s engine.Storage
	storageType := conf.Mode.Type
	switch storageType {
	case "local":
		var err error
		s, err = localstorage.New(conf.Mode.Local)
		if err != nil {
			appExit(fmt.Errorf("failed to create storage, err: %v", err))
		}
	default:
		appExit(errors.New("invalid storage type: " + storageType))
	}

	return s
}

// mustGetNode initiates local account
func mustGetNode(conf *config.ServerConf) peer.Local {
	if conf == nil {
		appExit(errors.New("missing config"))
	}

	if conf.Name == "" {
		appExit(errors.New("missing config: name"))
	}

	if conf.PrivateKey == "" {
		appExit(errors.New("missing config: privateKey"))
	}

	if conf.PublicAddress == "" {
		appExit(errors.New("missing config: publicAddress"))
	}

	sk, err := ecdsa.DecodePrivateKeyFromString(conf.PrivateKey)
	if err != nil {
		appExit(errorx.Wrap(err, "failed to decode private key"))
	}

	pk := ecdsa.PublicKeyFromPrivateKey(sk)

	local := peer.Local{
		Address:    conf.PublicAddress,
		ID:         pk[:],
		Name:       conf.Name,
		PrivateKey: sk,
	}
	return local
}

// initLog loads logger
func initLog(conf *config.Log) (io.Writer, logrus.Level) {
	if conf == nil {
		appExit(errors.New("missing log config"))
	}
	path := conf.Path
	if path == "" {
		return os.Stderr, logrus.DebugLevel
	}
	if strings.LastIndex(path, "/") != len([]rune(path))-1 {
		path = path + "/"
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			return os.Stderr, logrus.DebugLevel
		}
	}
	fileName := path + "server.log"

	level, err := logrus.ParseLevel(conf.Level)
	if err != nil {
		level = logrus.InfoLevel
	}

	writer, _ := rotatelogs.New(
		fileName+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(fileName),
		rotatelogs.WithMaxAge(time.Duration(720)*time.Hour),
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour),
	)

	return writer, level
}
