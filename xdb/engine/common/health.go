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

package common

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// GetHealthNodes gets online healthy(green, yellow) nodes
func GetHealthNodes(chain CommonChain) (nodes blockchain.NodeHs, err error) {
	// Prepare
	allNodes, err := chain.ListNodes()
	if err != nil {
		return nodes, errorx.Wrap(err, "failed to list nodes from blockchain")
	}
	// get online nodes
	for _, n := range allNodes {
		if !n.Online {
			continue
		}
		// judge node health
		health, err := chain.GetNodeHealth(n.ID)
		if err != nil || blockchain.NodeHealthBad == health {
			continue
		}
		nh := blockchain.NodeH{
			Node:   n,
			Health: health,
		}

		nodes = append(nodes, nh)
	}
	if len(nodes) == 0 {
		return nodes, errorx.New(errorx.ErrCodeInternal, "empty healthy nodes")
	}

	return nodes, nil
}

// FindNewNodes selects a new healthy node for slice
func FindNewNodes(healthNodes blockchain.NodeHs, selected []string) (blockchain.Nodes, error) {
	// get green and yellow nodes set
	var greenNodeList blockchain.Nodes
	var yellowNodeList blockchain.Nodes
	for _, n := range healthNodes {
		if strInSet(selected, string(n.Node.ID)) {
			continue
		}
		if n.Health == blockchain.NodeHealthGood {
			greenNodeList = append(greenNodeList, n.Node)
		}
		if n.Health == blockchain.NodeHealthMedium {
			yellowNodeList = append(yellowNodeList, n.Node)
		}
	}
	// random order
	greenNodeList = rearrangeNodes(greenNodeList)
	yellowNodeList = rearrangeNodes(yellowNodeList)
	// green first
	newNodes := append(greenNodeList, yellowNodeList...)
	if len(newNodes) == 0 {
		return newNodes, errorx.New(errorx.ErrCodeNotFound, "no more available healthy nodes")
	}
	return newNodes, nil
}

// GetNsFilesHealth gets namespace health conditions
func GetNsFilesHealth(ctx context.Context, ns blockchain.Namespace, chain CommonChain) (nsh blockchain.NamespaceH, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	listOpt := blockchain.ListFileOptions{
		Owner:       ns.Owner,
		Namespace:   ns.Name,
		TimeStart:   ns.CreateTime,
		TimeEnd:     time.Now().UnixNano(),
		CurrentTime: time.Now().UnixNano(),
	}
	files, err := chain.ListFiles(&listOpt)
	if err != nil {
		return nsh, errorx.Wrap(err, "failed to list file on blockchain")
	}

	wg := sync.WaitGroup{}
	wg.Add(len(files))
	fhlist := make([]string, len(files))
	isErr := false
	for index, file := range files {
		// concurrent to get file health
		go func(index int, file blockchain.File) {
			defer wg.Done()
			fh, err := GetFileHealth(ctx, chain, file, ns.Replica)
			if err != nil {
				isErr = true
				return
			}
			fhlist[index] = fh
		}(index, file)
	}
	wg.Wait()
	if isErr {
		return nsh, errorx.New(errorx.ErrCodeInternal, "failed get ns file health")
	}
	var fGreenN, fYellowN = 0, 0
	for _, h := range fhlist {
		switch h {
		case blockchain.NodeHealthGood:
			fGreenN += 1
		case blockchain.NodeHealthMedium:
			fYellowN += 1
		}
	}
	nsh = blockchain.NamespaceH{
		Namespace:      ns,
		FileNormalNum:  len(fhlist),
		FileExpiredNum: int(ns.FileTotalNum) - len(fhlist),
		GreenFileNum:   fGreenN,
		YellowFileNum:  fYellowN,
		RedFileNum:     len(fhlist) - fYellowN - fGreenN,
	}
	return nsh, nil
}

// GetFileHealth gets file heath status
func GetFileHealth(ctx context.Context, chain CommonChain, file blockchain.File, replica int) (health string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// check file legal
	if len(file.Slices) == 0 {
		return health, errorx.New(errorx.ErrCodeInternal, "failed get file health, file is illegal")
	}

	sliceList := make(map[string][]string)
	used := make(map[string]struct{})
	var fileNodes []string
	for _, eslice := range file.Slices {
		sliceList[eslice.ID] = append(sliceList[eslice.ID], string(eslice.NodeID))
		if _, exist := used[string(eslice.NodeID)]; !exist {
			used[string(eslice.NodeID)] = struct{}{}
			fileNodes = append(fileNodes, string(eslice.NodeID))
		}
	}

	fileHealthNodes, err := getFileNodesHealth(ctx, fileNodes, chain)
	if err != nil {
		return health, errorx.Wrap(err, "failed to get file node health")
	}

	var sliceH []string
	var mutex sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(sliceList))
	for _, nodes := range sliceList {
		select {
		case <-ctx.Done():
			return health, errorx.New(errorx.ErrCodeInternal, "context is canceled")
		default:
		}
		// concurrent to get slice health
		go func(nodes []string) {
			defer wg.Done()
			shealth, err := GetSliceAvgHealth(nodes, fileHealthNodes)
			if err != nil {
				return
			}
			mutex.Lock()
			sliceH = append(sliceH, shealth)
			mutex.Unlock()
		}(nodes)
	}
	wg.Wait()
	if len(sliceH) != len(sliceList) {
		return health, errorx.New(errorx.ErrCodeInternal, "failed to get slice avg health")
	}

	greenSln := 0
	for _, h := range sliceH {
		// file cannot be recovered
		if h == blockchain.NodeHealthBad {
			return h, nil
		}
		if h == blockchain.NodeHealthGood {
			greenSln += 1
		}
	}
	// if replica > 1 and the replica policy match namespaces replica and slice health is green
	if replica > 1 && (len(file.Slices) == len(sliceList)*replica) && greenSln == len(sliceH) {
		return blockchain.NodeHealthGood, nil
	}
	// if replica == 1, file health is yellow
	// if replica > 1, but the replica policy does match namespaces replica, slice health is yellow
	// if file most of the slices are yellow, slice health is yellow
	return blockchain.NodeHealthMedium, nil
}

// GetSliceAvgHealth get slice health status
func GetSliceAvgHealth(nodes []string, fhns map[string]string) (health string, err error) {
	var nhlist []string
	for _, node := range nodes {
		if _, exist := fhns[node]; !exist {
			return health, errorx.New(errorx.ErrCodeInternal, "failed get slice health")
		}
		nhlist = append(nhlist, fhns[node])
	}

	greenNl := 0
	redNl := 0
	for _, h := range nhlist {
		if h == blockchain.NodeHealthGood {
			greenNl += 1
		}
		if h == blockchain.NodeHealthBad {
			redNl += 1
		}
	}
	if redNl == len(nhlist) {
		return blockchain.NodeHealthBad, nil
	}
	if greenNl > 1 && greenNl == len(nhlist) {
		return blockchain.NodeHealthGood, nil
	}
	return blockchain.NodeHealthMedium, nil
}

// GetFileSysHealth get system health status of a file owner, include files and nodes health status
func GetFileSysHealth(ctx context.Context, nss []blockchain.Namespace, nodes blockchain.Nodes, chain CommonChain) (
	fh blockchain.FileSysHealth, err error) {
	fh.NsNum = len(nss)
	fh.NodeNum = len(nodes)

	// get ns files health
	var isErr error
	wg := sync.WaitGroup{}
	wg.Add(len(nss))
	cnsh := make(chan blockchain.NamespaceH, len(nss))
	for _, ns := range nss {
		select {
		case <-ctx.Done():
			return fh, errorx.New(errorx.ErrCodeInternal, "context is canceled")
		default:
		}
		go func(ns blockchain.Namespace) {
			defer wg.Done()
			efh, err := GetNsFilesHealth(ctx, ns, chain)
			if err != nil {
				isErr = err
				return
			}
			cnsh <- efh
		}(ns)
	}
	wg.Add(len(nodes))
	cnh := make(chan string, len(nodes))
	for _, n := range nodes {
		select {
		case <-ctx.Done():
			return fh, errorx.New(errorx.ErrCodeInternal, "context is canceled")
		default:
		}
		go func(n blockchain.Node) {
			defer wg.Done()
			status, err := chain.GetNodeHealth(n.ID)
			if err != nil {
				isErr = err
				return
			}
			cnh <- status
		}(n)
	}
	wg.Wait()
	close(cnsh)
	close(cnh)
	if isErr != nil {
		return fh, errorx.Wrap(isErr, "failed to get file ns health")
	}
	for efh := range cnsh {
		fh.FileNum += efh.FileNormalNum
		fh.FileExpiredNum += efh.FileExpiredNum
		fh.GreenFileNum += efh.GreenFileNum
		fh.YellowFileNum += efh.YellowFileNum
		fh.RedFileNum += efh.RedFileNum
	}
	if fh.FileNum > 0 {
		fh.FilesHealthRate = float64(fh.GreenFileNum) / float64(fh.FileNum)
	}
	if fh.FileNum == fh.GreenFileNum {
		fh.SysHealth = blockchain.NodeHealthGood
	} else if fh.RedFileNum >= 1 {
		fh.SysHealth = blockchain.NodeHealthBad
	} else {
		fh.SysHealth = blockchain.NodeHealthMedium
	}

	for s := range cnh {
		switch s {
		case blockchain.NodeHealthGood:
			fh.GreenNodeNum += 1
		case blockchain.NodeHealthBad:
			fh.RedNodeNum += 1
		case blockchain.NodeHealthMedium:
			fh.YellowNodeNum += 1
		}
	}
	if fh.NodeNum > 0 {
		fh.NodeHealthRate = float64(fh.GreenNodeNum) / float64(fh.NodeNum)
	}
	return fh, nil
}

// getFileNodesHealth get nodes health status
func getFileNodesHealth(ctx context.Context, nodes []string, chain CommonChain) (map[string]string, error) {
	nhlist := make(map[string]string)
	wg := sync.WaitGroup{}
	wg.Add(len(nodes))
	var isErr error
	var mutex sync.Mutex
	for _, node := range nodes {
		select {
		case <-ctx.Done():
			return nhlist, nil
		default:
		}
		// concurrent to get node health
		go func(node string) {
			defer wg.Done()
			enh, err := chain.GetNodeHealth([]byte(node))
			if err != nil {
				isErr = err
				return
			}
			mutex.Lock()
			nhlist[node] = enh
			mutex.Unlock()
		}(node)
	}
	wg.Wait()
	if isErr != nil {
		return nhlist, errorx.Wrap(isErr, "failed to get slice health")
	}
	return nhlist, nil
}

// GetSliceNodes get storage nodes for each slice
func GetSliceNodes(ss []blockchain.PublicSliceMeta, nodesMap map[string]blockchain.Node) map[string][]blockchain.Node {
	sl := make(map[string][]blockchain.Node)
	for _, slice := range ss {
		if _, exist := nodesMap[string(slice.NodeID)]; !exist {
			continue
		}
		sl[slice.ID] = append(sl[slice.ID], nodesMap[string(slice.NodeID)])
	}
	return sl
}

// rearrangeNodes arranges nodes in random order
func rearrangeNodes(nodes blockchain.Nodes) blockchain.Nodes {
	num := len(nodes)
	for i := 0; i < num; i++ {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(num)))
		k, _ := rand.Int(rand.Reader, big.NewInt(int64(num)))
		nodes[j.Int64()], nodes[k.Int64()] = nodes[k.Int64()], nodes[j.Int64()]
	}
	return nodes
}

// strInSet checks if s is in set
func strInSet(set []string, str string) bool {
	for _, s := range set {
		if s == str {
			return true
		}
	}
	return false
}
