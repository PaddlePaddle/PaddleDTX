业务系统可以直接引用PaddleDTX密码学库 [Crypto](https://github.com/PaddlePaddle/PaddleDTX/tree/master/crypto)（当前仅支持Go语言，JS未开源）进行账户创建、签名生成等操作。

### 账户创建
GenerateKeyPair用于各类节点生成自己的身份账户：
``` go linenums="1"
import (
    "github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
)
prikey, pubkey, err := ecdsa.GenerateKeyPair()
```


### 签名生成
计算需求节点与区块链网络、任务执行节点交互，数据持有节点客户端请求数据持有节点上传下载文件等操作均需要生成签名，签名生成规则如下：
``` go linenums="1"
import (
    "github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
)
// message待签名信息，不同请求的签名message不同
sig, err := ecdsa.Sign(privkey, hash.HashUsingSha256(message))
```


<br>










