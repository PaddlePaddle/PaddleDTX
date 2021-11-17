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
	"log"
	"sync"

	"google.golang.org/grpc"
)

// Peer defines peer
type Peer struct {
	// host of peer, like 127.0.0.1:8080
	address string
	// grpc Connection
	grpcConn *grpc.ClientConn
	// lock
	lock sync.Mutex
}

// getAddress returns peer address
func (p *Peer) getAddress() string {
	return p.address
}

// GetConnect gets rpc connection
func (p *Peer) GetConnect() (*grpc.ClientConn, error) {
	var err error = nil

	if p.needReconnect() {
		p.lock.Lock()
		if p.needReconnect() {
			err = p.getConn()
		}
		p.lock.Unlock()

		if err != nil {
			return nil, err
		}
	}

	return p.grpcConn, err
}

// needReconnect re-create the connection
//  that has been TRANSIENT_FAILURE、SHUTDOWN or Invalid-State
func (p *Peer) needReconnect() bool {
	if p.grpcConn == nil {
		return true
	}
	connState := p.grpcConn.GetState().String()
	if connState == "TRANSIENT_FAILURE" || connState == "SHUTDOWN" || connState == "Invalid-State" {
		return true
	}
	return false
}

// getConn creates grpc connection
func (p *Peer) getConn() error {
	conn, err := grpc.Dial(p.address, grpc.WithInsecure())
	if err != nil {
		log.Printf("Failed to connect server! error: %v", err.Error())
		return err
	}
	p.grpcConn = conn
	return nil
}

// closeConn closes grpc connection
func (p *Peer) closeConn() {
	p.grpcConn.Close()
}

// newPeer creates peer
func newPeer(address string) *Peer {
	p := &Peer{
		address: address,
	}

	return p
}
