## 网络集成
PaddleDTX提供了可信分布式AI网络的 **标准通信协议**，计算需求节点可以直接和区块链节点交互，进行任务发布、启动、查询，也可以通过HTTP/RPC API与任务执行节点交互，进行预测结果的下载。为方便用户深度使用系统的各项功能，这里以 **计算需求节点** 为例，介绍如何在区块链上发布任务、检索样本文件，以及如何和任务执行节点交互进行预测结果下载。


### 区块链网络
!!! note ""
    DAI使用的XuperChain网络，其提供了多语言版本的SDK（JS，Golang，C#，Java，Python），这里以Golang为例来介绍一下基于XuperChain的可信分布式AI合约调用流程。
	合约调用源码可参考 [计算需求节点](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/requester/client/client.go)。

#### 1. PublishTask
合约方法PublishTask用于发布计算任务：
``` go linenums="1"
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
#### 2. ListTask


### 任务执行节点

TODO...

<br>

