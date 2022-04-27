# 源码安装

### 1.1 环境准备
PaddleDTX 使用 golang 进行开发，当您使用源码进行编译和安装时，首先需要准备源码以及编译的环境：

* 系统环境，推荐使用Linux或MacOS操作系统
* golang编译器，推荐版本1.13+
[点击下载安装golang](https://studygolang.com/dl)
* git源码下载工具
[点击下载安装git](https://git-scm.com/download)
* make编译工具
	- ubuntu:   `$ sudo apt-get install -y make`
	- centos:  `$ sudo yum install -y make`
	- macOS:  `通过iTunes下载安装xcode`

### 1.2 源码编译

1. 下载源码:  
   ```
   $ git clone https://github.com/PaddlePaddle/PaddleDTX.git
   ```

2. 编译XuperDB
	```
	$ cd PaddleDTX/xdb
	$ make
	```
	编译产出为 output 文件夹，内容为：
	```
	├── conf
	│   ├── config-dataowner.toml
	│   └── config-storage.toml
	├── xdb                   //数据存储服务启动二进制,根据配置文件的不同有数据持有节点或存储节点两种类型
	└── xdb-cli               //数据存储服务操作客户端
	```
3. 编译Distributed AI
	```
	$ cd ../dai
	$ make
	```
	编译产出为 output 文件夹，内容为：
	```
	├── conf
	│   ├── config-cli.toml     //客户端配置,也可以使用config.toml作为客户端配置
	│   └── config.toml         //服务配置
	├── executor                //DAI任务执行节点服务启动二进制
	├── executor-cli            //DAI任务执行节点操作客户端
	└── requester-cli           //计算需求节点操作客户端
	```
4. 编译区块链合约:
	```
	$ go build -o paddlempc ./blockchain/xchain/contract
	```
	编译产出为 paddlempc 合约文件，为安装在xchain 区块链上的合约文件。

### 1.3 网络部署

1. 部署区块链网络

	PaddleDTX 使用区块链网络支撑计算层和去中心化存储网络，底层依赖可以使用同一个区块链网络。
	<br>
	这里使用百度超级链 xchain v3.9 作为底层区块链网络，可参考 [XuperChain环境部署](https://xuper.baidu.com/n/xuperdoc/v3.9/quickstart.html) 来搭建区块链网络。
	<br>
	您需要了解如何创建合约账户、部署智能合约，详细参考 [部署 native 合约](https://xuper.baidu.com/n/xuperdoc/v3.9/advanced_usage/create_contracts.html?highlight=native#native) ，更多内容请参考[XuperChain官方文档](https://xuper.baidu.com/n/xuperdoc/index.html) 。
	<br>
	合约安装过程如下：
	``` shell linenums="1"
	# 定义合约账户和合约名称
	$ export contractAccount='1234567890123456'
	$ export contractName='paddlempc'

	# 创建区块链账户，并转 token
	$ ./xchain-cli account newkeys --strength 1 -o ukeys
	$ export address=`cat ./ukeys/address`
	$ ./xchain-cli transfer --to $address --amount 1000000000000

	# 创建区块链合约账户，并转 token
	$ ./xchain-cli account new --account $contractAccount --fee 1000 --keys ./ukeys
	
	# 给合约账户转 token
	$ ./xchain-cli transfer --to XC${contractAccount}@xuper --amount 100000000000 --keys ./ukeys
	
	# 安装合约
	$ ./xchain-cli native deploy --account XC${contractAccount}@xuper --runtime go -a '{"creator":"XC${contractAccount}@xuper"}' --cname $contractName ./$contractName --fee 19267894 --keys ./ukeys

	# 查询合约安装的状态
	$ ./xchain-cli contract query $contractName
	```

2. 部署 XuperDB

	每一个 XuperDB 的节点都有一对公私钥，用来标识节点的账户，公私钥可以通过如下XuperDB的客户端命令进行获取：
	```
	$ ./xdb-cli key genkey -o ./keys
	```
	请妥善保存您创建的公私钥对，在后续的配置及命令行使用时您将会频繁的用到它。
	<br><br>
	^^XuperDB 包含两种类型的节点服务，数据持有节点和存储节点，我们需要分别启动这两种服务^^。为了方便，我们这里两种类型的服务分别启动一个。
	首先进入到 xdb 的编译产出 output 文件中：

	1.  启动数据存储节点

		修改配置文件，修改内容如下：
		``` toml linenums="1"
		# vim conf/config-storage.toml
		#
		type = "storage"
		[storage]
		name = "storageNode1"
		# 修改服务监听端口及对外服务地址
		listenAddress = ":8122"
		publicAddress = "127.0.0.1:8122"

		# genkey创建的私钥
		keyPath = "./keys"

		[storage.blockchain]
		    type = "xchain"
		    [storage.blockchain.xchain]
		        # 助记词为用户部署区块链网络后，安装合约过程中创建的区块链账户，取值./ukeys/mnemonic
		        mnemonic = "充 雄 孔 坝 低 狠 争 短 摸 拜 晨 造"
		        contractName = "paddlempc"
		        contractAccount = "XC1234567890123456@xuper"
		        chainAddress = "127.0.0.1:37101"
		        chainName = "xuper"

		    [storage.blockchain.fabric]
		        configFile = "./config/fabric/config.yaml"
		        channelId = "mychannel"
		        chaincode = "mycc"
		        userName = "Admin"
		        orgName = "org1"

		[storage.mode]
		    type = "local"
		    [storage.mode.local]
		        rootPath = "./slices"

		[storage.monitor]
		    challengingSwitch = "on"
		    nodemaintainerSwitch = "on"
		    fileclearInterval = 24

		[log]
		level = "debug"
		path = "./logs"
		```
		其中，listenAddress和publicAddress 指定服务监听的地址及对外暴露的地址，blockchain配置中使用区块链网络部署时创建的账户助记词、合约账户及合约名，rootPath指定文件存储的本地路径。

		启动服务：
		```
		$ nohup ./xdb -c conf/config-storage.toml > storage.log &
		```

	2.  启动数据持有节点
   
		修改配置文件，修改内容如下：
		``` toml linenums="1"
		# vim conf/config-dataowner.toml
		#
		type = "dataOwner"
		[dataOwner]
		name = "dataOwnerNode1"
		# 修改服务监听端口及对外服务地址
		listenAddress = ":8121"
		publicAddress = "127.0.0.1:8121"

		# genkey创建的私钥
		keyPath = "./keys"

		[dataOwner.slicer]
		    type = "simpleSlicer"
		    [dataOwner.slicer.simpleSlicer]
		        blockSize = 4194304
		        queueSize = 4

		[dataOwner.encryptor]
		    type = "softEncryptor"
		    [dataOwner.encryptor.softEncryptor]
		        password = "abcdefg"

		[dataOwner.challenger]
		    type = "merkle"
		    [dataOwner.challenger.pairing]
		        maxIndexNum = 5
		        sk = "Fudm9gDXNlEdCkieMid1WHIHd9K/M/CctBPlF/4y+AU="
		        pk = "B60Vdoq4SVUpVRZf1FM7ImRXo/22q0ZjYMHlaB4HgUXMSsu+2iCrNkk3gROXOUDvB8zWMcBGCnBCAnb6N9WOaBPbKkYWnp/iodp0+GFWvW1DCnAYNV2+vdaFkHaezsqeDqRDsdqV2uG47PTE2xdkljSblWNgKhsHYp7LgCcbBWiMO3TcrzUdq+ETxfIu1Bi7AzSHHAj8oc7toGT0anrO9LPSDcde8rCdsptX5CLH7WvRF0AXrqhX4Mr7i+547qI3"
		        randu = "NA1xy6JCWWc8IB4x1CM4DCoxKTqEele6zqD8kCfuj5s="
		        randv = "TV0J8YFWEsybwFdwm3DJvUHXx88YXkzK97Zpvj/tyGc="

		    [dataOwner.challenger.merkle]
		        leveldbRoot = "./challenger"
		        shrinkSize = 500
		        segmentSize = 5

		[dataOwner.blockchain]
		    type = "xchain"
		    [dataOwner.blockchain.xchain]
		        # 助记词为用户部署区块链网络后，安装合约过程中创建的区块链账户，取值./ukeys/mnemonic
		        mnemonic = "充 雄 孔 坝 低 狠 争 短 摸 拜 晨 造"
		        contractName = "paddlempc"
		        contractAccount = "XC1234567890123456@xuper"
		        chainAddress = "127.0.0.1:37101"
		        chainName = "xuper"

		    [dataOwner.blockchain.fabric]
		        configFile = "./config/fabric/config.yaml"
		        channelId = "mychannel"
		        chaincode = "mycc"
		        userName = "Admin"
		        orgName = "org1"

		[dataOwner.copier]
		    type = "random-copier"

		[dataOwner.monitor]
		    challengingSwitch = "on"
		    filemaintainerSwitch = "on"
		    filemigrateInterval = 6

		[log]
		level = "debug"
		path = "./logs"
		```
		其中，listenAddress和publicAddress 指定服务监听的地址及对外暴露的地址，blockchain配置中使用区块链网络部署时创建的账户助记词、合约账户及合约名。

		启动服务：
		```
		$ nohup ./xdb -c conf/config-dataowner.toml > dataowner.log &
		```
		!!! note ""
			注意：一般构建 PaddleDTX 网络至少需要两方参与，对应两个计算任务执行节点，每个任务执行节点可以从一个或多个数据持有节点获取数据，这里为了说明方便启动一个数据持有节点，您也可以根据实际需求自行启动多个数据存储节点和数据持有节点；配置中的keyPath参数为节点的身份，不同keyPath即对应了不同的身份。

	3. 查看服务状态: 使用 xdb-cli 客户端执行如下命令，请求数据持有节点查看存储节点的在线状态：
		```
		$ ./xdb-cli nodes list --host http://127.0.0.1:8121
		```

3. 部署Distributed AI

	一般多方安全计算至少由两个任务执行节点，所以这里部署两个任务执行节点。
	每一个 任务执行节点 都有一对公私钥，用来标识节点的账户，公私钥可以通过如下executor-cli的客户端命令进行获取：
	```
	$ ./executor-cli key genkey -o ./keys
	```
	请妥善保存您创建的公私钥对，在后续的配置及命令行使用时您将会频繁的用到它。
	!!! note ""
		注意: 任务发布后时，任务执行节点会向数据持有节点发起文件授权申请，数据持有节点可通过或拒绝样本文件授权申请。

	1. 准备两个任务执行节点的配置
		```
		$ cd PaddleDTX/dai/
		$ cp -r output executor1
		$ cp -r output executor2
		```
		需要分别修改对应的conf/config.toml文件：

		=== "executor1/conf/config.toml"

			``` toml linenums="1"
			# executor1
			listenAddress = ":8184"
			publicAddress = "127.0.0.1:8184"
			# genkey创建的私钥
			keyPath = "./keys"
			
			[executor.storage]
				# 定义模型存储的路径
				localModelStoragePath = "./models"
				# 定义模型评估结果的存储路径
				localEvaluationStoragePath = "./evalus"
				# 定义预测结果存储的方式，默认本地存储，如果用户采取XuperDB方式存储，则需提前生成数据持有节点客户端./ukeys并授权，同时创建预测结果存储的命名空间
				type = 'Local'
				[executor.storage.XuperDB]
					host = "http://127.0.0.1:8121"
					keyPath = "./ukeys"
					namespace = "mpc"
				[executor.storage.Local]
					localPredictStoragePath = "./predictions"
			[executor.blockchain]
				[executor.blockchain.xchain]
					# 助记词为用户部署区块链网络后，安装合约过程中创建的区块链账户，取值./ukeys/mnemonic
					mnemonic = "充 雄 孔 坝 低 狠 争 短 摸 拜 晨 造"
					contractName = "paddlempc"
					contractAccount = "XC1234567890123456@xuper"
					chainAddress = "127.0.0.1:37101"
					chainName = "xuper"
			```

		=== "executor2/conf/config.toml"

			``` toml linenums="1"
			# executor2
			listenAddress = ":8185"
			publicAddress = "127.0.0.1:8185"
			# genkey创建的私钥
			keyPath = "./keys"

			[executor.storage]
				# 定义模型存储的路径
				localModelStoragePath = "./models"
				# 定义模型评估结果的存储路径
				localEvaluationStoragePath = "./evalus"
				# 定义预测结果存储的方式，默认本地存储，如果用户采取XuperDB方式存储，则需提前生成数据持有节点客户端./ukeys并授权，同时创建预测结果存储的命名空间
				type = 'Local'
				[executor.storage.XuperDB]
					host = "http://127.0.0.1:8121"
					keyPath = "./ukeys"
					namespace = "mpc"
				[executor.storage.Local]
					localPredictStoragePath = "./predictions"
			[executor.blockchain]
				[executor.blockchain.xchain]
					# 助记词为用户部署区块链网络后，安装合约过程中创建的区块链账户，取值./ukeys/mnemonic
					mnemonic = "充 雄 孔 坝 低 狠 争 短 摸 拜 晨 造"
					contractName = "paddlempc"
					contractAccount = "XC1234567890123456@xuper"
					chainAddress = "127.0.0.1:37101"
					chainName = "xuper"
			```

	2. 启动服务

		分别在对应文件夹下执行如下命令，启动任务执行节点：
		```
		$ nohup ./executor &
		```

		通过命令查看有两个 executor 的进程，则启动成功：
		```
		$ ps -aux | grep executor
		```

### 1.4 客户端操作

步骤3为用户演示了源码安装区块链网络、去中心化存储网络（1个数据持有节点、1个数据存储节点）、Distributed AI（2个任务执行节点）流程，成功启动之后，可参考 [客户端工具](./client.md) 操作您的网络。

<br>
