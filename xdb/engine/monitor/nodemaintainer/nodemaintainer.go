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

package nodemaintainer

import (
	"context"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/peer"
)

const (
	defaultFileClearInterval = time.Hour * 24
)

var (
	logger = logrus.WithField("monitor", "nodemaintainer")
)

type Blockchain interface {
	AddNode(opt *blockchain.AddNodeOptions) error
	GetNode(id []byte) (blockchain.Node, error)
	NodeOnline(opt *blockchain.NodeOperateOptions) error
	Heartbeat(id, sig []byte, timestamp int64) error
	ListNodesExpireSlice(opt *blockchain.ListNodeSliceOptions) ([]string, error)
}

type SliceStorage interface {
	Load(key string) (io.ReadCloser, error)
	Delete(key string) (bool, error)
	Exist(key string) (bool, error)

	LoadStr(key string) (string, error)
	SaveAndUpdate(key string, value io.Reader) error
}

type NewNodeMaintainerOptions struct {
	LocalNode peer.Local

	Blockchain Blockchain

	SliceStorage SliceStorage
}

// NodeMaintainer runs if local node is storage-node, and its main work is to clean expired encrypted slices
//   and to send heartbeats regularly in order to claim it's alive
type NodeMaintainer struct {
	localNode peer.Local

	blockchain Blockchain

	sliceStorage SliceStorage

	heartbeatInterval  time.Duration
	fileClearInterval  time.Duration
	fileRetainInterval time.Duration

	doneHbC         chan struct{} //doneHbC will be closed when loop breaks
	doneSliceClearC chan struct{} //doneSliceClearC will be closed when loop breaks
}

func New(conf *config.MonitorConf, opt *NewNodeMaintainerOptions) (*NodeMaintainer, error) {
	heartbeatInterval := blockchain.HeartBeatFreq
	fileClearInterval := time.Duration(int64(conf.FileclearInterval)) * time.Hour
	if fileClearInterval == 0 {
		fileClearInterval = defaultFileClearInterval
	}

	logger.WithFields(logrus.Fields{
		"heartbeat-interval":  heartbeatInterval,
		"fileclear-interval":  fileClearInterval,
		"fileretain-interval": blockchain.FileRetainPeriod,
	}).Info("monitor initialize...")

	mm := &NodeMaintainer{
		localNode:          opt.LocalNode,
		blockchain:         opt.Blockchain,
		sliceStorage:       opt.SliceStorage,
		heartbeatInterval:  heartbeatInterval,
		fileClearInterval:  fileClearInterval,
		fileRetainInterval: blockchain.FileRetainPeriod,
	}

	return mm, nil
}

// HeartBeat sends heart beat onto blockchain
func (m *NodeMaintainer) HeartBeat(ctx context.Context) {
	go m.heartbeat(ctx)
}

// StopHeartBeat stops sending heart beats onto blockchain
func (m *NodeMaintainer) StopHeartBeat() {
	if m.doneHbC == nil {
		return
	}

	logger.Info("stops sending heart beats onto blockchain ...")

	select {
	case <-m.doneHbC:
		return
	default:
	}

	<-m.doneHbC
}

// NodeAutoRegister storage-node automatically register in blockchain
func (m *NodeMaintainer) NodeAutoRegister() error {
	return m.autoRegister()
}

// StartFileClear starts task to clear files
func (m *NodeMaintainer) StartFileClear(ctx context.Context) {
	go m.sliceClear(ctx)
}

// StopFileClear stops task clearing files
func (m *NodeMaintainer) StopFileClear() {
	if m.doneSliceClearC == nil {
		return
	}

	logger.Info("stops task clearing files ...")

	select {
	case <-m.doneSliceClearC:
		return
	default:
	}

	<-m.doneSliceClearC
}
