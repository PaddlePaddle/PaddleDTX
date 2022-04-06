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
	"io"

	"github.com/kataras/iris/v12"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	etype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// Handler defines all apis exposed
// The handler under the engine implements the following methods
type Handler interface {
	// The dataOwner node uses Write() and Read() to publish or download files
	Write(context.Context, etype.WriteOptions, io.Reader) (etype.WriteResponse, error)
	Read(context.Context, etype.ReadOptions) (io.ReadCloser, error)

	ListFiles(etype.ListFileOptions) ([]blockchain.File, error)
	ListExpiredFiles(etype.ListFileOptions) ([]blockchain.File, error)
	GetFileByID(ctx context.Context, id string) (blockchain.FileH, error)
	GetFileByName(ctx context.Context, pubkey, ns, name string) (blockchain.FileH, error)
	UpdateFileExpireTime(ctx context.Context, opt etype.UpdateFileEtimeOptions) error
	AddFileNs(opt etype.AddNsOptions) error
	UpdateNsReplica(ctx context.Context, opt etype.UpdateNsOptions) error
	ListFileNs(opt etype.ListNsOptions) ([]blockchain.Namespace, error)
	GetNsByName(ctx context.Context, pubkey, name string) (blockchain.NamespaceH, error)
	GetFileSysHealth(ctx context.Context, pubkey string) (blockchain.FileSysHealth, error)
	GetChallengeByID(id string) (blockchain.Challenge, error)
	GetChallenges(opt blockchain.ListChallengeOptions) ([]blockchain.Challenge, error)
	// The Storage node uses Push() or Pull() to store or provide ciphertext slices
	Push(etype.PushOptions, io.Reader) (etype.PushResponse, error)
	Pull(etype.PullOptions) (io.ReadCloser, error)
	// The dataOwner node uses the following methods to operate the applier's authorization request
	ListFileAuths(etype.ListFileAuthOptions) (blockchain.FileAuthApplications, error)
	ConfirmAuth(etype.ConfirmAuthOptions) error
	GetAuthByID(id string) (blockchain.FileAuthApplication, error)

	ListNodes() (blockchain.Nodes, error)
	GetNode([]byte) (blockchain.Node, error)
	GetHeartbeatNum([]byte, int64) (int, int, error)
	GetNodeHealth([]byte) (string, error)
	NodeOffline(etype.NodeOfflineOptions) error
	NodeOnline(etype.NodeOnlineOptions) error
	GetSliceMigrateRecords(opt *blockchain.NodeSliceMigrateOptions) (string, error)
}

// Server http server
type Server struct {
	app *iris.Application

	listenAddr string
	handler    Handler
}

// New initiate Server
func New(listenAddress string, h Handler) (*Server, error) {
	app := iris.New()
	if listenAddress == "" {
		return nil, errorx.New(errorx.ErrCodeConfig, "misssing config: listenAddress")
	}

	server := &Server{
		app:        app,
		listenAddr: listenAddress,
		handler:    h,
	}
	return server, nil
}

// setRoute define the routing of node's server
func (s *Server) setRoute(serverType string) (err error) {
	v1 := s.app.Party("/v1")
	nodeParty := v1.Party("/node")
	switch serverType {
	// storage node
	case config.NodeTypeStorage:
		sliceParty := v1.Party("/slice")
		sliceParty.Post("/push", s.push)
		sliceParty.Get("/pull", s.pull)

		nodeParty.Get("/list", s.listNodes)
		nodeParty.Get("/get", s.getNode)
		nodeParty.Get("/health", s.getNodeHealth)
		nodeParty.Post("/offline", s.nodeOffline)
		nodeParty.Post("/online", s.nodeOnline)
		nodeParty.Get("/getmrecord", s.getMRecord)
		nodeParty.Get("/gethbnum", s.getHeartbeatNum)
	// dataOwner node
	case config.NodeTypeDataOwner:
		fileParty := v1.Party("/file")
		fileParty.Post("/write", s.write)
		fileParty.Get("/read", s.read)
		fileParty.Get("/list", s.listFiles)
		fileParty.Get("/listexp", s.listExpiredFiles)
		fileParty.Get("/getbyid", s.getFileByID)
		fileParty.Get("/getbyname", s.getFileByName)
		fileParty.Post("/updatexptime", s.updateFileExpireTime)
		fileParty.Post("/addns", s.addFileNs)
		fileParty.Post("/ureplica", s.updateNsReplica)
		fileParty.Get("/listns", s.listFileNs)
		fileParty.Get("/getns", s.getNsByName)
		fileParty.Get("/getsyshealth", s.getSysHealth)
		fileParty.Get("/listauth", s.listFileAuths)
		fileParty.Post("/confirmauth", s.confirmAuth)
		fileParty.Get("/getauthbyid", s.getAuthByID)

		nodeParty.Get("/list", s.listNodes)
		nodeParty.Get("/get", s.getNode)
		nodeParty.Get("/health", s.getNodeHealth)
		nodeParty.Get("/getmrecord", s.getMRecord)
		nodeParty.Get("/gethbnum", s.getHeartbeatNum)

		challParty := v1.Party("/challenge")
		challParty.Get("/getbyid", s.getChallengeByID)
		challParty.Get("/toprove", s.getToProveChallenges)
		challParty.Get("/proved", s.getProvedChallenges)
		challParty.Get("/failed", s.getFailedChallenges)
	default:
		err = errorx.New(errorx.ErrCodeConfig, "wrong config: server.server-type")
	}
	s.app.OnAnyErrorCode(func(ictx iris.Context) {
		responseError(ictx, errorx.New(errorx.ErrCodeNotFound, "request url not found"))
	})
	return err
}

// Serve runs and blocks current routine
func (s *Server) Serve(ctx context.Context) error {
	if err := s.setRoute(config.GetServerType()); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		logrus.Info("server stops ...")
		s.app.Shutdown(context.TODO())
	}()

	logrus.Infof("server starts, and listens port %s", s.listenAddr)
	if err := s.app.Listen(s.listenAddr); err != nil {
		//error occurs when start server
		return err
	}

	return ctx.Err()
}
