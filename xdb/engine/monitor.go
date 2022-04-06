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

package engine

import (
	"context"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/monitor/challenging"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/monitor/filemaintainer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/monitor/nodemaintainer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// Monitor includes ChallengingMonitor, NodeMaintainer, and FileMaintainer
//  ChallengingMonitor's main work is to publish challenge requests if local node is dataOwner-node,
//     otherwise is to listen challenge requests and answer them in order to prove specified files are stored
//  NodeMaintainer runs if local node is storage-node, and its main work is to clean expired encrypted slices
//     and to send heartbeats regularly in order to claim it's alive
//  FileMaintainer runs if local node is dataOwner-node, and its main work is to check storage-nodes health conditions
//     and migrate slices from bad nodes to healthy nodes.
type Monitor struct {
	challengingMonitor *challenging.ChallengingMonitor
	nodeMaintainer     *nodemaintainer.NodeMaintainer
	fileMaintainer     *filemaintainer.FileMaintainer
}

// newMonitor initiates Monitor
func newMonitor(conf *config.MonitorConf, opt *NewEngineOption) (*Monitor, error) {
	challengingMonitor, err := newChallengingMonitor(conf, opt)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to new challenging monitor")
	}
	nodeMaintainer, err := newNodeMaintainer(conf, opt)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to new nodemaintainer monitor")
	}

	fileMaintainer, err := newFileMaintainer(conf, opt, challenging.DefaultRequestInterval.Nanoseconds())
	if err != nil {
		return nil, errorx.Wrap(err, "failed to new filemaintainer monitor")
	}

	m := &Monitor{
		challengingMonitor: challengingMonitor,
		nodeMaintainer:     nodeMaintainer,
		fileMaintainer:     fileMaintainer,
	}

	return m, nil
}

// newChallengingMonitor initiates ChallengingMonitor
// It's used for file's replicas retaining proof, nodes can start this through the switch
// the replicas retaining proof supports two algorithms, Merkle tree and bilinear pairing
// for dataOwner node, generate challenges request regularly
// for storage node, answer dataOwner's challenges request
func newChallengingMonitor(conf *config.MonitorConf, opt *NewEngineOption) (
	*challenging.ChallengingMonitor, error) {

	challengingSwitch := conf.ChallengingSwitch
	if challengingSwitch != "on" {
		return nil, nil
	}

	cmOpt := challenging.NewChallengingMonitorOptions{
		PrivateKey:   opt.LocalNode.PrivateKey,
		Blockchain:   opt.Chain,
		ChallengeDB:  opt.Challenger,
		SliceStorage: opt.Storage,
	}
	challengingMonitor, err := challenging.New(conf, &cmOpt)
	if err != nil {
		return nil, err
	}

	return challengingMonitor, nil
}

// newNodeMaintainer initiates NodeMaintainer
// for storage node, nodeMaintainer can register node's address into blockchain,
// clean expired file's slices and heartbeat
func newNodeMaintainer(conf *config.MonitorConf, opt *NewEngineOption) (*nodemaintainer.NodeMaintainer, error) {
	nodemaintainerSwitch := conf.NodemaintainerSwitch
	if nodemaintainerSwitch != "on" {
		return nil, nil
	}

	mmOpt := nodemaintainer.NewNodeMaintainerOptions{
		Blockchain:   opt.Chain,
		LocalNode:    opt.LocalNode,
		SliceStorage: opt.Storage,
	}

	nodeMaintainer, err := nodemaintainer.New(conf, &mmOpt)
	if err != nil {
		return nil, err
	}
	return nodeMaintainer, nil
}

// newFileMaintainer initiates FileMaintainer
// for dataOwner node, fileMaintainer used to check file's health and migrate unhealthy slices
func newFileMaintainer(conf *config.MonitorConf, opt *NewEngineOption, interval int64) (*filemaintainer.FileMaintainer, error) {
	filemaintainerSwitch := conf.FilemaintainerSwitch
	if filemaintainerSwitch != "on" {
		return nil, nil
	}

	fmOpt := filemaintainer.NewFileMaintainerOptions{
		LocalNode:  opt.LocalNode,
		Blockchain: opt.Chain,
		Copier:     opt.Copier,
		Encryptor:  opt.Encryptor,
		Challenger: opt.Challenger,
	}

	fileMaintainer, err := filemaintainer.New(conf, &fmOpt, interval)
	if err != nil {
		return nil, err
	}
	return fileMaintainer, nil
}

// Start starts Monitor
func (m *Monitor) Start(ctx context.Context) error {
	if m.challengingMonitor != nil {
		serType := config.GetServerType()
		switch serType {
		case config.NodeTypeDataOwner:
			m.challengingMonitor.StartChallengeRequest(ctx)
			m.fileMaintainer.Migrate(ctx)
		case config.NodeTypeStorage:
			if err := m.nodeMaintainer.NodeAutoRegister(); err != nil {
				return err
			}
			m.nodeMaintainer.StartFileClear(ctx)
			m.nodeMaintainer.HeartBeat(ctx)
			m.challengingMonitor.StartChallengeAnswer(ctx)
		}
	}

	return nil
}

// Close stops all inner services gracefully
//  could be called in main()
func (m *Monitor) Close() {
	if m.challengingMonitor != nil {
		m.challengingMonitor.StopChallengeRequest()
		m.challengingMonitor.StopChallengeAnswer()
	}

	if m.fileMaintainer != nil {
		m.fileMaintainer.StopMigrate()
	}

	if m.nodeMaintainer != nil {
		m.nodeMaintainer.StopFileClear()
		m.nodeMaintainer.StopHeartBeat()
	}
}
