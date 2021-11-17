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

package p2p

import (
	"errors"
	"log"
	"sync"
)

// State is the state of the P2P
type State uint8

const (
	// NEW indicates that the P2P is new and ready for providing service
	NEW State = iota

	// CLOSED indicates that the P2P has been closed
	CLOSED
)

// P2P defines p2p network
type P2P struct {
	peers sync.Map       // list of nodes, key is 'ip:port', and value is '*Peer'
	state State          // state of p2p network
	wg    sync.WaitGroup // for waiting all connections closed when get stop signal
}

// NewP2P creates P2P instance
func NewP2P(addrs ...string) *P2P {
	p := &P2P{
		state: NEW,
	}
	for _, a := range addrs {
		p.peers.Store(a, newPeer(a))
	}
	return p
}

// Stop stops p2p, and closes all the grpc connection
func (p *P2P) Stop() {
	// change state
	p.state = CLOSED

	// waiting for all peer getting free
	log.Println("Start to shut down P2P, please wait...")
	p.wg.Wait()

	// close all connection
	p.peers.Range(func(k, v interface{}) bool {
		v.(*Peer).closeConn()
		return true
	})
}

// GetPeer gets an available peer
func (p *P2P) GetPeer(address string) (*Peer, error) {
	if p.state == CLOSED {
		return nil, errors.New("service closed")
	}
	peer := p.getPeerNotExist(address)

	p.wg.Add(1)
	return peer, nil
}

// getPeerNotExist gets an available peer, creates one if peer does not exist
func (p *P2P) getPeerNotExist(address string) *Peer {
	peer, _ := p.peers.LoadOrStore(address, newPeer(address))
	return peer.(*Peer)
}

// FreePeer frees a peer
func (p *P2P) FreePeer() {
	p.wg.Done()
}
