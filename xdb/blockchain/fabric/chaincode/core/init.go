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

package core

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type Xdata struct{}

func (x *Xdata) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("init xdata chaincode")
	return shim.Success(nil)
}

func (x *Xdata) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fn, args := stub.GetFunctionAndParameters()
	switch fn {
	case "setValue":
		return x.setValue(stub, args)
	case "getValue":
		return x.getValue(stub, args)
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
	default:
		return shim.Error("Invalid invoke function name.")
	}
}

func (x *Xdata) setValue(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	err := stub.PutState(args[0], []byte(args[1]))
	if err != nil {
		return shim.Error("set right fail " + err.Error())
	}
	return shim.Success(nil)
}

func (x *Xdata) getValue(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	result, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error("query" + args[0] + " fail:" + err.Error())
	}
	return shim.Success(result)
}
