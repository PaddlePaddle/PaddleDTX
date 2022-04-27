# Docker安装

### 1.1 环境准备
PaddleDTX也支持使用 docker 进行编译、安装和使用，您需要准备如下环境：

* docker，推荐版本18.03+ [点击下载安装 docker](https://docs.docker.com/get-docker/)
* docker-compose，推荐版本1.26.0+ [点击下载安装 docker-compose](https://github.com/docker/compose/releases)

### 1.2 制作镜像

1. 制作XuperDB镜像
	``` shell
	$ cd PaddleDTX/xdb
	$ sh build_image.sh
	```
	产出为数据存储节点和数据持有节点两个镜像，镜像名和版本号分别为：
	* registry.baidubce.com/paddledtx/xdb-storage:1.1
	* registry.baidubce.com/paddledtx/xdb-dataowner:1.1

	实际上他们是使用不同镜像名的同一个镜像，可以通过修改 *build_image.sh* 脚本来修改镜像名和版本号。

2. 制作 Distributed AI 镜像
	``` shell
	$ cd PaddleDTX/dai
	$ sh build_image.sh
	```
	产出镜像名和版本号为 *registry.baidubce.com/paddledtx/paddledtx-dai:1.1* ，可以通过修改 *build_image.sh* 脚本来修改镜像名和版本号。

3. 编译合约
	``` shell
	$ export contractName='paddlempc'
	$ docker run -it --rm \
      -v $(dirname ${PWD}):/workspace \
      -v ~/.ssh:/root/.ssh \
      -w /workspace \
      -e GONOSUMDB=* \
      -e GO111MODULE=on \
      golang:1.13.4 sh -c "cd dai && go build -o ../testdata/blockchain/contract/$contractName ./blockchain/xchain/contract"
	```

### 1.3 网络部署

1. 部署区块链网络
	``` shell
	$ cd PaddleDTX/testdata/blockchain
	$ docker-compose -f docker-compose.yml up -d
	```
	搭建了三个节点的区块链网络，对应的区块链相关配置在文件夹 blockchain/xchain1/conf、blockchain/xchain2/conf、blockchain/xchain3/conf 下，需要调整配置时在网络拉起前进行修改。

	可以通过容器中的 xchain-cli客户端进行区块链上的一些操作，例如创建合约账户及安装智能合约。
	

	``` shell linenums="1"
	# 定义合约账户和合约名称
	$ export contractAccount='1234567890123456'
	$ export contractName='paddlempc'

	# 创建区块链账户，并转 token
	# 这里也可以使用已有的区块链账户，文章后面的修改区块链配置可以不操作
	$ docker exec -it xchain1.node.com sh -c "./xchain-cli account newkeys --strength 1  -o user"
	$ export address=`cat user/address`
	$ docker exec -it xchain1.node.com sh -c "./xchain-cli transfer --to $address --amount 1000000000000"
	
	# 创建合约账户
	$ docker exec -it xchain1.node.com sh -c "./xchain-cli account new --account $contractAccount --fee 1000 --keys ./user"

	# 给合约账户转 token
	$ docker exec -it xchain1.node.com sh -c "./xchain-cli transfer --to XC${contractAccount}@xuper --amount 100000000000 --keys ./user"

	# 将合约拷贝到容器中
	$ docker cp ./contract/$contractName xchain1.node.com:/home/work/xchain/$contractName

	# 安装合约
	$ docker exec -it xchain1.node.com sh -c "./xchain-cli native deploy --account XC${contractAccount}@xuper --runtime go -a '{\"creator\":\"XC${contractAccount}@xuper\"}' --cname $contractName ./contract/$contractName --fee 19267894 --keys ./user --fee 19597986"

	# 查询合约安装的状态
	$ docker exec -it xchain1.node.com sh -c "./xchain-cli contract query $contractName"

	```

2. 部署 XuperDB

	数据持有节点将自己的隐私数据进行加密、切分、副本复制后分发到存储节点，存储节点是数据存储的物理节点。这里部署三个存储节点和两个数据持有节点，两个数据节点模拟分别提供部分数据的两方。
	
	修改配置文件：
	``` shell linenums="1"
	$ vim PaddleDTX/testdata/xdb/data1/conf/config.toml
	$ vim PaddleDTX/testdata/xdb/data2/conf/config.toml

	# 使用在区块链部署时创建的合约账户、合约以及助记词
	[dataOwner.blockchain]
    type = "xchain"
    [dataOwner.blockchain.xchain]
        mnemonic = "提 现 详 责 腐 贪 沉 回 涨 谓 献 即"
        contractName = "paddlempc"
        contractAccount = "XC1234567890123456@xuper"
        chainAddress = "xchain1.node.com:37101"
        chainName = "xuper"
	```

	```  shell linenums="1"
	$ vim xdb/storage1/conf/config.toml
	$ vim xdb/storage2/conf/config.toml
	$ vim xdb/storage3/conf/config.toml

	# 使用在区块链部署时创建的合约账户、合约以及助记词
	[storage.blockchain]
	type = "xchain"
	[storage.blockchain.xchain]
        mnemonic = "提 现 详 责 腐 贪 沉 回 涨 谓 献 即"
        contractName = "paddlempc"
        contractAccount = "XC1234567890123456@xuper"
        chainAddress = "xchain1.node.com:37101"
        chainName = "xuper"
	```

	启动服务：
	```
	$ cd PaddleDTX/testdata/xdb
	$ docker-compose -f docker-compose.yml up -d
	```

	查看存储节点列表：
	```
	$ docker exec -it dataowner1.node.com sh -c "./xdb-cli nodes list --host http://dataowner1.node.com:80"
	```
	!!! note ""
		注意：如果用户想启动基于Fabric的XuperDB服务，可参考[XuperDB服务启动和命令使用说明](https://github.com/PaddlePaddle/PaddleDTX/tree/master/xdb/scripts)。

3. 部署 Distributed AI

	部署两个任务执行节点，模拟由两方组成的多方安全计算网络，两个任务执行节点分别对应不同的数据持有节点。

	修改配置文件：
	```  shell linenums="1"
	$ vim PaddleDTX/testdata/executor/node1/conf/config.toml
	$ vim PaddleDTX/testdata/executor/node2/conf/config.toml
	# 使用在区块链部署时创建的合约账户、合约以及助记词
	[executor.blockchain]
	type = 'xchain'
	[executor.blockchain.xchain]
        mnemonic = "提 现 详 责 腐 贪 沉 回 涨 谓 献 即"
        contractName = "paddlempc"
        contractAccount = "XC1234567890123456@xuper"
        chainAddress = "xchain1.node.com:37101"
        chainName = "xuper"
	```

	启动服务：
	```
	$ cd PaddleDTX/testdata/executor
	$ docker-compose -f docker-compose.yml up -d
	```

### 1.4 客户端操作

上述为用户演示了docker-compose方式启动网络流程，在实际应用中，用户可以通过K8S跨主机集群部署PaddleDTX，网络操作参考 [客户端工具](./client.md)。

<br>
