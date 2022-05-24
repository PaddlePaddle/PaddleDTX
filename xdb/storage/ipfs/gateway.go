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

package ipfs

import (
	"bytes"
	"context"
	"crypto/md5"
	"io"
	"sort"
	"strings"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

type Gateway interface {
	// Add performs `ipfs add`
	Add(key string, r io.Reader) (cid string, err error)
	// Cat performs `ipfs cat` to show file data
	Cat(key string, cid string) (closer io.ReadCloser, err error)
	// Unpin performs `ipfs pin rm -r`
	Unpin(key string, cid string) (err error)
	// PinLs performs `ipfs pin ls`
	PinLs(key string, cid string) (pinInfo map[string]shell.PinInfo, err error)
}

type client struct {
	shID []byte // hash of peer host
	sh   *shell.Shell
}

// gateway is a simple implementation of a Gateway
// based on `Consistent Hashing``
type gateway struct {
	backups    int                       // number of backups
	successors map[string][]*shell.Shell // successors of hash
	clients    []client                  // clients ordered in asending
}

// NewGateway returns gateway instance or error if mistakes occur
// timeout specifies a time limit for requests, a Timeout of zero means no timeout.
func NewGateway(hosts []string, timeout time.Duration) (Gateway, error) {
	if len(hosts) == 0 {
		return nil, errorx.New(errorx.ErrCodeParam, "no ipfs hosts")
	}
	num := len(hosts)

	gw := gateway{
		backups:    4,
		successors: make(map[string][]*shell.Shell, num),
		clients:    make([]client, 0, num),
	}

	if num < gw.backups {
		gw.backups = num
	}

	// create Shells for input hosts
	// the input hosts should be different with each other
	for _, h := range hosts {
		ms := md5.Sum([]byte(h))
		bms := make([]byte, 0, len(ms))
		for i := 0; i < len(ms); i++ {
			bms = append(bms, ms[i])
		}
		if _, ok := gw.successors[string(bms)]; !ok {
			gw.successors[string(bms)] = make([]*shell.Shell, 0, gw.backups)
			sh := shell.NewShell(h)
			sh.SetTimeout(timeout)
			gw.clients = append(gw.clients, client{bms, sh})
		} else {
			return nil, errorx.New(errorx.ErrCodeParam, "duplicated hash[%v] and its host is[%s]", bms, h)
		}
	}

	// sort ShellID in ascending
	sort.Slice(gw.clients, func(i, j int) bool {
		if bytes.Compare(gw.clients[i].shID, gw.clients[j].shID) < 0 {
			return true
		}
		return false
	})

	// set successors for all Shells
	for i, cli := range gw.clients {
		strShID := string(cli.shID)

		j := i
		for c := 0; c < gw.backups; c++ {
			if j >= num {
				j -= num
			}
			gw.successors[strShID] = append(gw.successors[strShID], gw.clients[j].sh)
			j++
		}
	}

	logger.WithFields(logrus.Fields{
		"backups":    gw.backups,
		"successors": gw.successors,
	}).Info("new IPFS gateway")
	return &gw, nil
}

// getShells return successors for input key
func (gw *gateway) getShells(key string) []*shell.Shell {
	ms := md5.Sum([]byte(key))
	bms := make([]byte, 0, len(ms))
	for i := 0; i < len(ms); i++ {
		bms = append(bms, ms[i])
	}

	// find the first one not smaller than input hashed key
	index := sort.Search(len(gw.clients), func(i int) bool {
		if bytes.Compare(gw.clients[i].shID, bms) >= 0 {
			return true
		}
		return false
	})

	// imagine the Slice as a circle
	// if it is not found in the Slice, the successor is number-0
	if index >= len(gw.clients) {
		index = 0
	}

	shID := gw.clients[index].shID
	return gw.successors[string(shID)]
}

// Add performs `ipfs add` on successors until successful
func (gw *gateway) Add(key string, r io.Reader) (cid string, err error) {
	shs := gw.getShells(key)
	for _, sh := range shs {
		cid, err = sh.Add(r)
		if err == nil {
			return cid, err
		}
	}
	return "", err
}

// Cat performs `ipfs cat` to show file data on successors until successful
// cid is CID of a file
func (gw *gateway) Cat(key string, cid string) (closer io.ReadCloser, err error) {

	shs := gw.getShells(key)

	for _, sh := range shs {
		closer, err = sh.Cat(cid)
		if err == nil {
			return closer, err
		}
	}
	return nil, err
}

// PinLs performs `ipfs pin ls`
// Request to successors first,
// and try to request to all peers until get success response
func (gw *gateway) PinLs(key string, cid string) (pinInfo map[string]shell.PinInfo, err error) {
	// checkUnpinned is a function used to determine the Error is ‘is not pinned’,
	// or other kinds, such as 'context deadline exceeded'.
	// if the failure is not due to ‘is not pinned’, the process should break and return the Error to caller
	checkUnpinned := func(errStr string) bool {
		if strings.HasSuffix(errStr, "is not pinned") {
			return true
		}
		return false
	}

	pinls := func(sh *shell.Shell, cid string) (map[string]shell.PinInfo, error) {
		var raw struct{ Keys map[string]shell.PinInfo }
		err := sh.Request("pin/ls", cid).Option("type", "recursive").Exec(context.Background(), &raw)
		if err != nil {
			return nil, err
		}
		return raw.Keys, nil
	}

	shs := gw.getShells(key)
	for _, sh := range shs {
		info, err := pinls(sh, cid)
		if err == nil {
			return info, err
		} else if !checkUnpinned(err.Error()) {
			return nil, err
		}
	}

	// If `pin ls` fails on all successors,
	// one likely reason is that the large-scale expansion causes a large number of changes in the successor peers,
	// the other reason is that the file is not pinned
	// So, execute `pin ls` on all nodes to make sure whether the file is pinned or not
	for _, cli := range gw.clients {
		info, err := pinls(cli.sh, cid)
		if err == nil {
			return info, err
		} else if !checkUnpinned(err.Error()) {
			return nil, err
		}
	}

	return map[string]shell.PinInfo{}, nil
}

// Unpin performs `ipfs pin rm -r`
// Request to successors first,
// and try to request to all peers until get success response
func (gw *gateway) Unpin(key string, cid string) (err error) {
	// checkUnpinned is a function used to determine the Error is ‘not pinned or pinned indirectly’,
	// or other kinds, such as 'context deadline exceeded'.
	// if the failure is not due to ‘is not pinned’, the process should break and return the Error to caller
	checkUnpinned := func(errStr string) bool {
		if strings.HasSuffix(errStr, "not pinned or pinned indirectly") {
			return true
		}
		return false
	}

	shs := gw.getShells(key)
	for _, sh := range shs {
		err = sh.Unpin(cid)
		if err == nil {
			return err
		} else if !checkUnpinned(err.Error()) {
			return err
		}
	}

	// If `Unpin` fails on all successors,
	// the most likely reason is that the large-scale expansion causes a large number of changes in the successor peers
	// So, execute Unpin on all nodes until successful
	for _, cli := range gw.clients {
		err = cli.sh.Unpin(cid)
		if err == nil {
			return err
		} else if !checkUnpinned(err.Error()) {
			return err
		}
	}

	return err
}
