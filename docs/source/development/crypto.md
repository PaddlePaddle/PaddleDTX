业务系统可以直接引用PaddleDTX密码学库 [Crypto](https://github.com/PaddlePaddle/PaddleDTX/tree/master/crypto)（当前仅支持Go语言，JS未开源）进行账户创建、签名生成等操作。

### 账户创建
GenerateKeyPair用于各类节点生成自己的身份账户：
``` go
import (
    "github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
)
prikey, pubkey, err := ecdsa.GenerateKeyPair()
```


### 签名生成
计算需求节点与区块链网络、任务执行节点交互，数据持有节点客户端请求数据持有节点上传下载文件等操作均需要生成签名，签名生成规则如下：
``` go
import (
    "github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
)
// message为待签名信息，不同请求的签名message不同
// privkey为节点身份的账户私钥
signature, err := ecdsa.Sign(privkey, hash.HashUsingSha256(message))
```

### 签名规则
签名是通过标准SHA2-256算法对数据计算哈希，待签名数据为字段按字母a-z排序处理后拼接的结果，如下载预测结果参数：
```
{
    "taskID": "e6ebf3c9-d1af-4381-8656-844405fc9c18", 
    "pubKey":"RjfvefFLA2ztWbdkCLDYhFOsnluqUjqGiQqlR+rD46D0o8AFF48CHBsGDZFvQggsGOHVdQXNqu7xBnKeZEL05Q=="
}
```

则生成签名的字符串：
```
message = "pubKey=RjfvefFLA2ztWbdkCLDYhFOsnluqUjqGiQqlR+rD46D0o8AFF48CHBsGDZFvQggsGOHVdQXNqu7xBnKeZEL05Q==&taskID=e6ebf3c9-d1af-4381-8656-844405fc9c18"
signature = hash.HashUsingSha256([]byte(message))
```
	
DAI合约调用参数参考 [任务操作](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/blockchain/blockchain.go)，
XuperDB&DAI HTTP/GRPC API接口调用参数参考 [接口说明](./api.md)









