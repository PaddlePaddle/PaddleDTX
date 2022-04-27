# 节点配置

## 数据持有节点
conf/config-dataowner.toml 文件配置说明如下：
``` toml
# the type of the running node, can be set with 'dataOwner' or 'storage'.
type = "dataOwner"

#########################################################################
#
#  [dataOwner] defines features of a dataOwner node.
#
#########################################################################
[dataOwner]
# Define a name of the node, for readability
name = "node1"

# The Address this server will listen on
listenAddress = ":8121"

# The private key of the node.
# Different key express different identity.
# Only need to choose one from 'privateKey' and 'keyPath', and if both exist, 'keyPath' takes precedence over 'privateKey'
# privateKey = "5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79"
keyPath = "./keys"

publicAddress = "10.144.94.17:8121"

[dataOwner.slicer]
    type = "simpleSlicer"
    [dataOwner.slicer.simpleSlicer]
        blockSize = 4194304
        queueSize = 4

[dataOwner.encryptor]
    type = "softEncryptor"
    [dataOwner.encryptor.softEncryptor]
        password = "abcdefg"

# The generator of the challenge requests, to check if the file exists on the storage node.
[dataOwner.challenger]
    # Generator's type, can be 'pairing' or 'merkle'.
    type = "pairing"

    # Proof of data Possession based on Bilinear Pairing.
    [dataOwner.challenger.pairing]
        # Max file slices number when generating pairing based challenge
        maxIndexNum = 5
        sk = "Fudm9gDXNlEdCkieMid1WHIHd9K/M/CctBPlF/4y+AU="
        pk = "B60Vdoq4SVUpVRZf1FM7ImRXo/22q0ZjYMHlaB4HgUXMSsu+2iCrNkk3gROXOUDvB8zWMcBGCnBCAnb6N9WOaBPbKkYWnp/iodp0+GFWvW1DCnAYNV2+vdaFkHaezsqeDqRDsdqV2uG47PTE2xdkljSblWNgKhsHYp7LgCcbBWiMO3TcrzUdq+ETxfIu1Bi7AzSHHAj8oc7toGT0anrO9LPSDcde8rCdsptX5CLH7WvRF0AXrqhX4Mr7i+547qI3"
        # Random Number used by pdb.
        randu = "NA1xy6JCWWc8IB4x1CM4DCoxKTqEele6zqD8kCfuj5s="
        randv = "TV0J8YFWEsybwFdwm3DJvUHXx88YXkzK97Zpvj/tyGc="

    # Proof of data Possession based on Merkle Tree.
    [dataOwner.challenger.merkle]
        leveldbRoot = "/home/data/challenger"
        shrinkSize = 500
        segmentSize = 5

# Blockchain used by the dataOwner node.
[dataOwner.blockchain]
    # blockchain type, 'xchain' or 'fabric'
    type = "xchain"

    # The configuration of how to invoke contracts using xchain. It is necessary when type is 'xchain'.
    [dataOwner.blockchain.xchain]
        mnemonic = "臂 袁 饿 莫 新 棉 骗 矩 巨 愿 稍 造"
        contractName = "dstorage1"
        contractAccount = "XC7142093261616521@dstorage"
        chainAddress = "106.12.139.7:15022"
        chainName = "dstorage"

    # The configuration of how to invoke contracts using fabric. It is necessary when type is 'fabric'.
    [dataOwner.blockchain.fabric]
        configFile = "./conf/fabric/config.yaml"
        channelId = "mychannel"
        chaincode = "mycc"
        userName = "Admin"
        orgName = "org1"

# The copier makes backups of files, currently only supports 'random-copier'.
[dataOwner.copier]
    type = "random-copier"

# The monitor will query new tasks in blockchain regularly, and trigger the task handler's operations
[dataOwner.monitor]
    # Whether to monitor the challenge answer of the storage node.
    challengingSwitch = "on"

    # Whether to monitor the file migration.
    filemaintainerSwitch = "on"
    # unit: hour
    filemigrateInterval = 6

#########################################################################
#
#   [log] sets the log related options
#
#########################################################################
[log]
level = "debug"
path = "./logs"

```
!!! note "配置说明"

    1. dataOwner.slicer 定义切片大小、文件切分时并行队列数；
    2. dataOwner.encryptor 配置文件及切片加密的初始密钥，系统采取一次一密方式，后续密钥均基于该密钥衍生；
    3. dataOwner.challenger 定义了副本保持证明的算法，支持 'pairing' or 'merkle'；
    4. dataOwner.blockchain 定义了节点操作区块链网络所需的配置，当前支持Xchain、Fabric网络；

## 数据存储节点
conf/config-storage.toml 文件配置说明如下：
``` toml
# The type of the running node, can be set with 'dataOwner' or 'storage'.
type = "storage"

#########################################################################
#
#  [storage] defines features of a storage node.
#
#########################################################################
[storage]
# Define a name of the node, for readability
name = "node1"

# The Address this server will listen on
listenAddress = ":8122"

# The private key of the node.
# Different key express different identity.
# Only need to choose one from 'privateKey' and 'keyPath', and if both exist, 'keyPath' takes precedence over 'privateKey'
# privateKey = "5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79"
keyPath = "./keys"

# The endpoint can be connected by other node, and showed in blockchain.
# If your network mode is 'host', it is the machine's ip and the port in publicAddress in before section.
publicAddress = "10.144.94.17:8122"

# Blockchain used by the storage node.
[storage.blockchain]
    # blockchain type, 'xchain' or 'fabric'
    type = "xchain"

    # The configuration of how to invoke contracts using xchain. It is necessary when type is 'xchain'.
    [storage.blockchain.xchain]
        mnemonic = "臂 袁 饿 莫 新 棉 骗 矩 巨 愿 稍 造"
        contractName = "dstorage1"
        contractAccount = "XC7142093261616521@dstorage"
        chainAddress = "106.12.139.7:15022"
        chainName = "dstorage"

    # The configuration of how to invoke contracts using fabric. It is necessary when type is 'fabric'.
    [storage.blockchain.fabric]
        configFile = "./conf/fabric/config.yaml"
        channelId = "mychannel"
        chaincode = "mycc"
        userName = "Admin"
        orgName = "org1"

# The storage mode used by the storage node, currently only supports local file system.
[storage.mode]
    type = "local"
    [storage.mode.local]
        # Location of file fragments
        rootPath = "/root/xdb/data/slices"

# The monitor will query new tasks in blockchain regularly, and trigger the task handler's operations
[storage.monitor]
    # Whether to monitor the challenge requests from the dataOwner node.
    challengingSwitch = "on"

    # Whether to monitor the node's change， such as  HeartBeat etc.
    nodemaintainerSwitch = "on"
    # Interval time of the node maintainer to clear file slice
    fileclearInterval = 24

#########################################################################
#
#   [log] sets the log related options
#
#########################################################################
[log]
level = "debug"
path = "./logs"

```

!!! note "配置说明"

    1. storage.blockchain 定义了节点操作区块链网络所需的配置，当前支持Xchain、Fabric网络；
    2. storage.mode 用于指定存储节点的存储方式，当前仅支持本地文件系统方式存储，后续持续支持Ipfs、Nas等；
    3. storage.monitor 用于存储节点开启心跳检测、配置文件清理时间间隔等；


<br>