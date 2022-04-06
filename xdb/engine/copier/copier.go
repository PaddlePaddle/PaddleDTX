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

package copier

import (
	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
)

// LocatedSlice contains Nodes selected to store Slice
type LocatedSlice struct {
	Slice slicer.Slice
	Nodes blockchain.Nodes
}

// SelectOptions contains some options for selecting Storage Nodes
//  Replica is the number of replicas
//  Excludes is a reserved field, not used
type SelectOptions struct {
	Replica  uint32
	Excludes map[string]struct{} // nodeID -> struct{}
}

// ReplicaExpOptions contains some options for expanding replicas.
type ReplicaExpOptions struct {
	SliceID       string                       // expand sliceID of file
	PrivateKey    []byte                       // used for signature
	SelectedNodes blockchain.Nodes             // slice exist nodes list
	NewReplica    int                          // new replica number
	NodesList     blockchain.NodeHs            // all node lists
	SliceMetas    []blockchain.PublicSliceMeta // slice metas
	PairingConf   types.PairingChallengeConf   // pairing based challenge config
}
