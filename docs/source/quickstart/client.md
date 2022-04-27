# 客户端工具

PaddleDTX各方使用的客户端工具详细使用说明如下：

* [Distributed AI 计算需求方命令使用说明](../tutorial/dai-cmd.md#计算需求节点)
* [Distributed AI 任务执行方命令使用说明](../tutorial/dai-cmd.md#_3)
* [XuperDB 客户端命令使用说明](../tutorial/xdb-cmd.md#数据持有节点)

本章重点介绍使用 PaddleDTX 时的几个**常用命令** 。

### 1. 操作XuperDB

#### 1.1 创建客户端帐号
XuperDB数据持有节点服务的访问依赖于授权的客户端公私钥，可通过如下命令创建客户端公私钥：

```
$ ./xdb-cli key genkey -o ./ukeys
```

#### 1.2 授权客户端帐号
生成数据持有节点客户端公私钥后，需要服务端对客户端进行授权：

``` shell
# $publicKey为1.1步骤生成的客户端端公钥，cat ./ukeys/public.key
$ ./xdb-cli key addukey -o ./authkeys -u $publicKey
```
该命令会将客户端公钥添加到服务端授权白名单，只有授权通过的客户端才允许请求数据持有节点进行文件上传下载。授权记录存储在服务端的 authkeys 文件夹中，-u 取值为客户端公钥。

#### 1.3 创建命名空间

使用 XuperDB 服务的第一步是在每一个数据持有节点创建文件存储的命名空间，使用如下命令：

```
$ ./xdb-cli files addns  --host http://127.0.0.1:8121 --keyPath './ukeys' -n paddlempc  -r 2
```

可以通过替换 host 来实现请求不同的数据持有节点，--keyPath默认取值'./ukeys'，从该文件夹中读取客户端的私钥，-n 参数为命名空间的名称，-r 参数为副本数，一般取大于 1 的数。

如果您使用 docker-compose 来部署网络，需要进入 docker 后执行命令，也可以使用docker exec命令，例如

``` shell
$ docker exec -it dataowner1.node.com sh -c "./xdb-cli files addns --host http://dataowner1.node.com:80 -n paddlempc -r 2 --keyPath ./ukeys"
```

使用 listns 命令可以查看已有的命名空间：

```
$ ./xdb-cli files listns  --host http://127.0.0.1:8122
```

--keyPath默认取值'./ukeys'，该命令从该文件夹中读取客户端的公钥。

#### 1.4 上传文件

执行训练任务和预测任务之前都需要上传对应的文件，文件上传需要请求对应的数据持有节点进行上传。

对于用户部署的环境，需要分别上传两方的样本文件和预测文件。

``` shell
$ ./xdb-cli --host http://127.0.0.1:8121 files upload -n paddlempc -m train_dataA4.csv -i ./train_dataA.csv --ext '{"FileType":"csv","Features":"id,CRIM,ZN,INDUS,CHAS,NOX,RM"，"TotalRows":457}'  -e '2021-12-10 12:00:00' -d 'train_dataA4' --keyPath ./ukeys

# 命令返回
FileID: 01edba10-ef04-4096-a984-c81191262d03
```
各参数说明如下：

* 通过修改 host 来指定不同的数据持有节点
* --keyPath默认取值'./ukeys'，从该文件夹中读取客户端的私钥
* -n 为命名空间的名称
* -m 为文件名称
* -i 指定了上传的文件
* --ext指定了样本或者预测文件中的标签
* -e 为文件在 XuperDB 中的过期时间
* -d 为文件描述

上传文件后，可以使用getbyid命令进行文件的查询：

```
$ ./xdb-cli files getbyid -i 01edba10-ef04-4096-a984-c81191262d03 --host http://127.0.0.1:8121
```

相同的，如果使用 docker 的话，需要进入 docker 后执行命令，也可以使用docker exec命令。

#### 1.5 授权查询与确认

计算需求方发布训练或预测任务后，任务执行节点可获取到需自身参与计算的任务，自动向样本文件持有方发起文件授权使用申请，此时数据持有节点可通过如下命令查询授权申请列表：

```
$ ./xdb-cli --host http://localhost:8121 files listauth -a 6a5ba56bccf843c591a3a32baa5aa76deebffe2695d48521799b77fb0a32e286ae493143560d1f548dd494bf266d4df39375f755f6008e8db7444cd8a96258c6 -o 71c516458ef075609be6a7ebaeca23dc42a3ff3aa0597d0abd3843253da09ee5bcdc292d517617bb7eb610a5351ae92240a803cc5769346d81def574adbfdd1d
```
通过修改 host 来指定不同的数据持有节点，-a 为任务执行节点公钥，-o 为文件持有方公钥。

查询文件授权申请列表后，可以通过confirmauth命令进行文件授权使用确认，-i 为待授权文件ID：

``` shell
$ ./xdb-cli files confirmauth --host http://127.0.0.1:8121 -e '2022-08-08 15:15:04' -i b87b588f-2e46-4ee5-8128-888592ada4fd --keyPath ./ukeys
```

### 2. 操作Distributed AI

Distributed AI的操作方分为两个角色，计算需求方和任务执行方，分别通过 requester-cli 和 executor-cli 两个命令行客户端进行操作。

#### 2.1 创建计算需求方账户

```
$ ./requester-cli key genkey -o ./keys
```

#### 2.2 查询任务执行节点列表
发布训练或预测任务时，计算需求方需指定任务执行节点，如下命令可以查询区块链网络上的任务执行节点：

```
$ ./requester-cli nodes list
```

#### 2.3 发布训练任务

训练任务由计算需求方发起：

``` shell
$ ./requester-cli task publish -a "linear-vl" -l "MEDV" --keyPath './keys' -t "train" -n "房价预测任务v3" -d "hahahha" -p "id,id" --conf ./testdata/executor/node1/conf/config.toml -f "01edba10-ef04-4096-a984-c81191262d03,21e5b591-9126-4df8-8b84-72a682a46fc1" -e "executor1,executor2"

# 命令行返回
TaskID: fdc5b7e1-fc87-4e4b-86ee-b139a7721391
```

如果希望执行模型评估，增加 '--ev'、'--evRule'等参数：

``` shell
$ ./requester-cli task publish -a "linear-vl" -l "MEDV" --keyPath './keys' -t "train" -n "房价预测任务v3" -d "hahahha" -p "id,id" --conf ./testdata/executor/node1/conf/config.toml -f "01edba10-ef04-4096-a984-c81191262d03,21e5b591-9126-4df8-8b84-72a682a46fc1" -e "executor1,executor2"  --ev

# 命令行返回
TaskID: fdc5b7e1-fc87-4e4b-86ee-b139a7721391
```

命令行各参数说明如下：

* -a: 训练使用的算法，可选线性回归 'linear-vl' 或逻辑回归 'logistic-vl'
* -l: 训练的目标特征
* --keyPath: 默认取值'./keys'，从该文件夹中读取私钥，计算需求方的私钥，表明了计算需求方的身份，可以用-k 参数直接指定私钥
* -t: 任务类型，可选训练任务'train' 或预测任务 'predict'
* -n: 任务名称
* -d: 任务描述
* -p: PSI求交时使用的标签
* --conf: 使用的配置文件
* -f: 训练使用的文件ID，这里是一个列表，指明了各个任务执行方需要使用的文件
* -e: 任务执行节点名称，和-f 一一对应，执行节点执行任务时，分别取对应位置的样本文件
* --ev: 训练结束后，执行模型评估
* --evRule: 模型评估方式，可选随机划分训练集 '0'、10折交叉验证 '1' 或者 Leave One Out '2'，默认取值'0'

与 XuperDB 的使用方法一致，当使用 docker-compose 部署时需要进入容器执行命令或者使用 docker exec 命令，后续命令将不再赘述。

#### 2.4 授权确认

计算需求方发布任务之后，各个任务执行节点会自动向数据持有节点发起文件授权使用申请，此时需要数据持有节点查询授权申请并进行确认，命令参考：./xdb-cli files confirmauth。

#### 2.5 启动训练任务

当所有的任务执行节点对任务进行确认后，需要计算需求方触发启动命令的执行，训练任务的执行结果是产出一个预测模型。如果在发布训练任务的时候指定执行模型评估，训练任务结束后，训练样本中含有目标特征的一方会生成模型评估结果，并保存于 ./evalus 路径下。

``` shell
$ ./requester-cli task start --id fdc5b7e1-fc87-4e4b-86ee-b139a7721391 --keyPath './keys' --conf ./testdata/executor/node1/conf/config.toml
```

各参数说明如下：

* --id: 任务 id
* --keyPath: 默认取值'./keys'，从该文件夹中读取私钥，计算需求方的私钥，表明了计算需求方的身份，可以用-k 参数直接指定私钥
* --conf: 使用的配置文件

#### 2.6 发布预测任务

训练任务执行完成后产出预测模型，计算需求方可以提交预测任务，为预测数据计算出预测结果。

``` shell
$ ./requester-cli task publish -a "linear-vl" --keyPath './keys' -t "predict" -n "房价任务v3" -d "hahahha" -p "id,id" --conf ./testdata/executor/node1/conf/config.toml -f "01d3b812-4dd7-4deb-a48d-4437312a164a,e02b27a6-0057-4673-b7ec-408ad060c952" -i fdc5b7e1-fc87-4e4b-86ee-b139a7721391  -e "executor1,executor2"
 
# 命令行返回
TaskID: a7dfac43-fa51-423e-bd05-8e0965c708a8
```

参数说明与发布训练任务差别在一个参数：

* -i: 指定训练任务的ID，使用训练任务的产出

#### 2.7 授权确认

计算需求方发布任务之后，各个任务执行节点会自动向数据持有节点发起文件授权使用申请，此时需要数据持有节点查询授权申请并进行确认，命令参考：./xdb-cli files confirmauth。

#### 2.8 启动预测任务

任务被各任务执行节点确认后，由计算需求方启动预测任务。

``` shell
$ ./requester-cli task start --id a7dfac43-fa51-423e-bd05-8e0965c708a8 --keyPath './keys' --conf ./testdata/executor/node1/conf/config.toml
```

#### 2.9 获取预测结果

预测任务执行成功后，计算需求方可以获取到预测的结果。

``` shell
$ ./requester-cli task result --id a7dfac43-fa51-423e-bd05-8e0965c708a8 --keyPath './keys' --conf ./testdata/executor/node1/conf/config.toml  -o ./output.csv
```

各参数说明如下：

* --id: 预测任务的 id
* --keyPath: 默认取值'./keys'，从该文件夹中读取私钥，计算需求方的私钥，表明了计算需求方的身份，可以用-k 参数直接指定私钥
* --conf: 指定使用的配置文件
* -o: 预测结果的导出文件

<br>
