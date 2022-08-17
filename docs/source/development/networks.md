## 网络集成
PaddleDTX提供了可信分布式AI网络的 **标准通信协议**，计算需求节点可以直接和区块链节点交互，进行任务发布、启动、查询，也可以通过HTTP/RPC API与任务执行节点交互，进行预测结果的下载。为方便用户深度使用系统的各项功能，这里以 **计算需求节点** 为例，介绍如何在区块链上发布任务、检索样本文件，以及如何和任务执行节点交互进行预测结果下载。


### 区块链网络
!!! info ""
    DAI使用的XuperChain网络，其提供了多语言版本的SDK（JS，Golang，C#，Java，Python），这里以Golang为例来介绍一下基于XuperChain的可信分布式AI合约调用流程。
	合约调用源码可参考 [计算需求节点](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/requester/client/client.go)。

#### 1.PublishTask
合约方法PublishTask用于发布计算任务：
``` go
// PublishFLTaskOptions contains parameters for publishing tasks
type PublishFLTaskOptions struct {
	FLTask    FLTask
	Signature []byte
}

// protos/task/task.proto
// FLTask is a message received from Executor and defines Federated Learning Task based on MPC
message FLTask {
    string iD = 1;
    string name = 2;
    string description = 3;

    bytes requester = 4;
    repeated DataForTask dataSets = 5;
	common.TaskParams algoParam = 6; // fl algorithm related params

	string status = 7;
	string errMessage = 8;
	string result = 9;
	int64 publishTime = 10;
	int64 startTime = 11;
	int64 endTime = 12;
}
```
#### 2.ListTask
ListTask用于查询计算任务列表：
``` go
// ListFLTaskOptions contains parameters for listing tasks
// support listing tasks a requester published or tasks an executor involved
type ListFLTaskOptions struct {
	PubKey    []byte // requester or executor's public key
	Status    string // task status
	TimeStart int64  // task publish time period, only task published after TimeStart and before TimeEnd will be listed
	TimeEnd   int64  
	Limit     int64  // limit number of tasks in list request, default 'all'
}
```

#### 3.GetTaskByID
通过任务ID查询任务详情，合约参数为id。

#### 4.StartTask
StartTask用于启动已确认的任务列表，合约参数taskId、signature。

#### 5.ListExecutorNodes
ListExecutorNodes用于查询区块链网络中的任务执行节点列表。

#### 6.GetExecutorNodeByID
通过任务执行节点公钥查询节点详情，合约参数为id。

### 任务执行节点
#### 1.下载预测结果
通过ListExecutorNodes查询到任务执行节点列表后，调用GRPC/HTTP API请求拥有标签方的任务执行节点下载预测结果：
``` go
service Task {
	// GetPredictResult is provided by Executor server for Executor client to get prediction result.
    rpc GetPredictResult(TaskRequest) returns (PredictResponse) {
        option (google.api.http) = {
            post : "/v1/task/predictres/get"
            body : "*"
        };
    }
}

// TaskRequest is message sent between Executors to request to start a task. 
message TaskRequest {
    bytes pubKey = 1;
    string taskID = 2;
    bytes signature = 4;
}

// PredictResponse is a message received from Executor 
message PredictResponse {
    string taskID = 1;
    bytes payload = 2; 
}

```
