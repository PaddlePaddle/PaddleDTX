PaddleDTX的四类节点（数据持有节点、存储节点、任务执行节点、区块链节点）均提供了HTTP API接口，供用户调用。

## XuperDB
### 1. 数据持有节点
#### 1.1 文件操作
参数类型和详细说明参考 [input.go](https://github.com/PaddlePaddle/PaddleDTX/xdb/server/types/input.go) 文件：

| URL  | Method | Param | explanation |
| :--------:   | :----------: | :------------: | :------: | 
|   /v1/file/write   |      POST   |   WriteOptions：user、token、ns、name、expireTime、desc、ext  | upload file |
|   /v1/file/read    |      GET    |   ReadOptions：user、token、ns、name、file_id、timestamp  | download file |
|   /v1/file/list    |      GET    |   ListFileOptions：owner、ns、start、end、ctime、limit  | list the unexpired files |
|   /v1/file/listexp |      GET    |   ListFileOptions：owner、ns、start、end、ctime、limit  | list expired but valid files |
|   /v1/file/getbyid |      GET    |   id（file id）  | get file by id |
|   /v1/file/getbyname |      GET    |   owner、ns、name  | get file by file name and namespace |
|   /v1/file/updatexptime |      POST    |   UpdateFileEtimeOptions：id、expireTime、ctime、user、token  | update file's expired time |
|   /v1/file/addns |      POST    |   AddNsOptions：replica、owner、ns、desc、ctime、user、token  | add file namespace |
|   /v1/file/ureplica |      POST    |   UpdateNsOptions：ns、replica、ctime、user、token  | update file namespace's replica |
|   /v1/file/listns   |      GET     |   ListNsOptions：owner、start、end、limit  | list namespaces by owner |
|   /v1/file/getns    |      GET     |   name、 owner（dataOwner nodes's public key） | get namespace by name |
|   /v1/file/getsyshealth |      GET    |   owner（dataOwner nodes's public key）  | get file owner's system health status |
|   /v1/file/listauth     |      GET    |  ListFileAuthOptions：applierPubkey、authorizerPubkey、fileID、status、start、end、limit  | list file's authorization applications |
|   /v1/file/confirmauth |      POST    |   ConfirmAuthOptions：status、user、authID、expireTime、token、rejectReason  | no, the default is "./conf/config.toml" |
|   /v1/file/getauthbyid |      GET     |   authID              | query authorization application detail by authID |


#### 1.2 节点操作
| URL  | Method | Param | explanation |
| :--------:   | :----------: | :------------: | :------: | 
|   /v1/node/list     |      GET   |     | list storage nodes |
|   /v1/node/get      |      GET    |   id（storage nodes's public key）  | get storage node's detail |
|   /v1/node/health   |      GET    |   id（storage nodes's public key）  | get storage node's health status|
|   /v1/node/getmrecord     |      GET    |   NodeSliceMigrateOptions：id、start、end、limit  | get storage node migration records  |
|   /v1/node/gethbnum      |      GET    |   id、ctime  | get storage node heartbeat number |

#### 1.3 副本保持证明
| URL  | Method | Param | explanation |
| :--------:   | :----------: | :------------: | :------: | 
|   /v1/challenge/getbyid    |      GET    |   id（challenge id）  | get challenge by challenge id |
|   /v1/challenge/toprove    |      GET    |   ListChallengeOptions：owner、node、file、start、end、limit  | get challenges with status "ToProve" |
|   /v1/challenge/proved     |      GET    |   ListChallengeOptions：owner、node、file、start、end、limit  | get challenges with status "proved" |
|   /v1/challenge/failed     |      GET    |   ListChallengeOptions：owner、node、file、start、end、limit  | get challenges with status "Failed" |


### 2. 存储节点
参数类型和详细说明参考 [input.go](https://github.com/PaddlePaddle/PaddleDTX/xdb/server/types/input.go) 文件：
#### 2.1 切片操作
| URL  | Method | Param | explanation |
| :--------:   | :----------: | :------------: | :------: | 
|   /v1/slice/push    |      POST   |   PushOptions：slice_id、source_id  | push file's slice |
|   /v1/slice/pull    |      GET    |   PullOptions：slice_id、file_id、timestamp、signature、pubkey  | pull file's slice |


#### 2.2 节点操作
| URL  | Method | Param | explanation |
| :--------:   | :----------: | :------------: | :------: | 
|   /v1/node/list     |      GET   |     | list storage nodes |
|   /v1/node/get      |      GET    |   id（storage nodes's public key）  | get storage node's detail |
|   /v1/node/health   |      GET    |   id（storage nodes's public key）  | get storage node's health |
|   /v1/node/offline  |      POST   |   NodeOfflineOptions：node、nonce、token  | node online |
|   /v1/node/online   |      POST   |   NodeOnlineOptions：node、nonce、token   | node offline |
|   /v1/node/getmrecord     |      GET    |   NodeSliceMigrateOptions：id、start、end、limit  | get storage node migration records  |
|   /v1/node/gethbnum      |      GET    |   id、ctime  | get storage node heartbeat number |


## Distributed AI
### 1. 任务执行节点
如下为任务执行节点对外提供的API接口，接口参数说明参考 [input.go](https://github.com/PaddlePaddle/PaddleDTX/xdb/server/types/input.go)：
#### 1.1 任务操作

``` proto linenums="1"
// Cluster defines communication communication between client and server, and communication between cluster members.
service Task {
    // ListTask is provided by Executor server for Executor client to list tasks with filters.
    rpc ListTask(ListTaskRequest)  returns (FLTasks) {
        option (google.api.http) = {
            post : "/v1/task/list"
            body : "*"
        };
    }
    // GetTaskById is provided by Executor server for Executor client to query a task.
    rpc GetTaskById(GetTaskRequest) returns (FLTask) {
        option (google.api.http) = {
            post : "/v1/task/getbyid"
            body : "*"
        };
    }
    // GetPredictResult is provided by Executor server for Executor client to get prediction result.
    rpc GetPredictResult(TaskRequest) returns (PredictResponse) {
        option (google.api.http) = {
            post : "/v1/task/predictres/get"
            body : "*"
        };
    }
    // StartTask is for Executors to request remote ones to start a task.
    rpc StartTask(TaskRequest) returns (TaskResponse);
}
```


## 区块链节点
DAI底链使用的是的Xuperchain，其提供了http_gateway，用于转发用户的HTTP请求，启动说明参考 [http_gateway](https://github.com/xuperchain/xuperchain/tree/v3.9/core/gateway)，支持的API接口参考 [xchain.proto](https://github.com/xuperchain/xuperchain/blob/v3.9/core/pb/xchain.proto)。



<br>
