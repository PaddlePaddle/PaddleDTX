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
	"fmt"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain/fabric/chaincode/core"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type Xdata struct {
	core.Xdata
}

func main() {
	err := shim.Start(new(Xdata))
	if err != nil {
		fmt.Printf("failed to start Xdata chaincode: %s", err)
	}
}

func (x *Xdata) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fn, args := stub.GetFunctionAndParameters()
	switch fn {
	// xdb function
	case "SetValue":
		return x.SetValue(stub, args)
	case "GetValue":
		return x.GetValue(stub, args)
	case "AddNode":
		return x.AddNode(stub, args)
	case "ListNodes":
		return x.ListNodes(stub, args)
	case "GetNode":
		return x.GetNode(stub, args)
	case "NodeOffline":
		return x.NodeOffline(stub, args)
	case "NodeOnline":
		return x.NodeOnline(stub, args)
	case "Heartbeat":
		return x.Heartbeat(stub, args)
	case "GetHeartbeatNum":
		return x.GetHeartbeatNum(stub, args)
	case "ListNodesExpireSlice":
		return x.ListNodesExpireSlice(stub, args)
	case "GetSliceMigrateRecords":
		return x.GetSliceMigrateRecords(stub, args)
	case "PublishFile":
		return x.PublishFile(stub, args)
	case "AddFileNs":
		return x.AddFileNs(stub, args)
	case "UpdateNsReplica":
		return x.UpdateNsReplica(stub, args)
	case "UpdateFilePublicSliceMeta":
		return x.UpdateFilePublicSliceMeta(stub, args)
	case "GetFileByName":
		return x.GetFileByName(stub, args)
	case "GetFileByID":
		return x.GetFileByID(stub, args)
	case "UpdateFileExpireTime":
		return x.UpdateFileExpireTime(stub, args)
	case "SliceMigrateRecord":
		return x.SliceMigrateRecord(stub, args)
	case "ListFiles":
		return x.ListFiles(stub, args)
	case "ListExpiredFiles":
		return x.ListExpiredFiles(stub, args)
	case "ListFileNs":
		return x.ListFileNs(stub, args)
	case "GetNsByName":
		return x.GetNsByName(stub, args)
	case "PublishFileAuthApplication":
		return x.PublishFileAuthApplication(stub, args)
	case "ConfirmFileAuthApplication":
		return x.ConfirmFileAuthApplication(stub, args)
	case "RejectFileAuthApplication":
		return x.RejectFileAuthApplication(stub, args)
	case "ListFileAuthApplications":
		return x.ListFileAuthApplications(stub, args)
	case "GetAuthApplicationByID":
		return x.GetAuthApplicationByID(stub, args)
	case "ListChallengeRequests":
		return x.ListChallengeRequests(stub, args)
	case "ChallengeRequest":
		return x.ChallengeRequest(stub, args)
	case "ChallengeAnswer":
		return x.ChallengeAnswer(stub, args)
	case "GetChallengeByID":
		return x.GetChallengeByID(stub, args)
	case "GetChallengeNum":
		return x.GetChallengeNum(stub, args)
	// dai function
	case "RegisterExecutorNode":
		return x.RegisterExecutorNode(stub, args)
	case "ListExecutorNodes":
		return x.ListExecutorNodes(stub, args)
	case "GetExecutorNodeByID":
		return x.GetExecutorNodeByID(stub, args)
	case "GetExecutorNodeByName":
		return x.GetExecutorNodeByName(stub, args)
	case "PublishTask":
		return x.PublishTask(stub, args)
	case "ListTask":
		return x.ListTask(stub, args)
	case "GetTaskById":
		return x.GetTaskById(stub, args)
	case "ConfirmTask":
		return x.ConfirmTask(stub, args)
	case "RejectTask":
		return x.RejectTask(stub, args)
	case "ExecuteTask":
		return x.ExecuteTask(stub, args)
	case "StartTask":
		return x.StartTask(stub, args)
	case "FinishTask":
		return x.FinishTask(stub, args)
	default:
		return shim.Error("Invalid invoke function name.")
	}
}
