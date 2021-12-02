# 客户端工具

PaddleDTX各方使用的客户端工具详细使用说明如下：

* <a href="https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/requester/cmd/README.md">Distributed AI 计算需求方命令使用说明</a>
* <a href="https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/executor/cmd/README.md">Distributed AI 任务执行方命令使用说明</a>
* <a href="https://github.com/PaddlePaddle/PaddleDTX/blob/master/xdb/cmd/client/README.md">XuperDB 客户端命令使用说明</a>

本章重点介绍使用 PaddleDTX 时的几个常用命令。

## 操作XuperDB

### 创建命名空间

使用 XuperDB 的第一步是在每一个数据持有节点创建文件存储的命名空间，使用如下命令：

```
$ ./xdata-cli files addns  --host http://127.0.0.1:8122 -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 -n paddlempc  -r 2
```

可以通过替换 host 来实现请求不同的数据持有节点；-k 参数为对应数据持有节点的私钥，不同的数据持有节点有不同私钥，可以通过对应的配置 config.toml 来获取；-n 参数为命名空间的名称，这里命名空间会被配置到任务执行节点中，所以需要创建任务执行节点中使用的命名空间；-r 参数为副本数，一般取大于 1 的数。

如果您使用 docker-compose 来部署网络，需要进入 docker 后执行命令，也可以使用docker exec命令，例如

```
$ docker exec -it dataowner1.node.com sh -c "./xdata-cli files addns  --host http://dataowner1.node.com:80 -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 -n paddlempc  -r 1"
```

使用 listns 命令可以查看已有的命名空间：

```
$ ./xdata-cli files listns  --host http://127.0.0.1:8122 -o 4637ef79f14b036ced59b76408b0d88453ac9e5baa523a86890aa547eac3e3a0f4a3c005178f021c1b060d916f42082c18e1d57505cdaaeef106729e6442f4e5
```

-o 参数为数据持有节点的公钥

### 上传文件

执行训练任务和预测任务之前都需要上传对应的文件，文件上传需要请求对应的数据持有节点进行上传。

对应我们部署的环境，需要上传两方的样本文件和一方的预测文件。

```
$ ./xdata-cli --host http://127.0.0.1:8122 files upload -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 -n paddlempc -m train_dataA4.csv -i ./train_dataA.csv --ext '{"FileType":"csv","Features":"id,CRIM,ZN,INDUS,CHAS,NOX,RM", "TotalRows":457}'  -e '2021-12-10 12:00:00' -d 'train_dataA4'
# 命令返回
FileID: 01edba10-ef04-4096-a984-c81191262d03
```

通过修改 host 来指定不同的数据持有节点；-k 参数为数据持有节点使用的私钥；-n 为命名空间的名称; -i 指定了上传的文件；--ext指定了样本或者预测文件中的标签；-e 为文件在 XuperDB 中的过期时间；-d 为文件描述。

上传文件后，可以使用getbyid命令进行文件的查询：

```
$ ./xdata-cli files getbyid  -i 01edba10-ef04-4096-a984-c81191262d03 --host http://127.0.0.1:8122
```

相同的，如果使用 docker 的话，需要进入 docker 后执行命令，也可以使用docker exec命令。

## 操作Distributed AI

Distributed AI的操作方分为两个角色，计算需求方和任务执行方，分别通过 requester-cli 和 executor-cli 两个命令行客户端进行操作。

### 发布训练任务

训练任务由计算需求方发起：

```
$ ./requester-cli task publish -a "linear-vl" -l "MEDV" -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 -t "train" -n "房价预测任务v3" -d "hahahha" -p "id,id" --conf ./testdata/executor/node1/conf/config.toml -f "01edba10-ef04-4096-a984-c81191262d03,21e5b591-9126-4df8-8b84-72a682a46fc1"
# 命令行返回
TaskID: fdc5b7e1-fc87-4e4b-86ee-b139a7721391
```

命令行各参数说明如下：

* -a: 训练使用的算法, 可选线性回归 'linear-vl' 或逻辑回归 'logistic-vl'
* -l: 训练的目标特征
* -k: 计算需求方的私钥, 表明了计算需求方的身份
* -t: 任务类型, 可选训练任务'train' 或预测任务 'predict'
* -n: 任务名称
* -d: 任务描述
* -p: PSI求交时使用的标签
* --conf: 使用的配置文件
* -f: 训练使用的文件ID, 这里是一个列表, 指明了各个任务执行方需要使用的文件

与 XuperDB 的使用方法一致，当使用 docker-compose 部署时需要进入容器执行命令或者使用 docker exec 命令，后续命令将不再赘述。

### 确认训练任务

计算需求方发布任务之后，需要各个任务执行节点进行确认，确认时通过 executor-cli 客户端进行操作。

```
$ ./executor-cli task confirm  --id  fdc5b7e1-fc87-4e4b-86ee-b139a7721391 --host 127.0.0.1:8123 -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21
 
```

各参数说明如下：
* --id: 为训练任务的 id
* --host: 对应任务执行节点的地址
* -k: 任务执行节点的私钥

### 启动训练任务

当所有的任务执行节点对任务进行确认后，需要计算需求方触发启动命令的执行，训练任务的执行结果是产出一个预测模型。

```
$ ./requester-cli task start --id fdc5b7e1-fc87-4e4b-86ee-b139a7721391 -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 --conf ./testdata/executor/node1/conf/config.toml
```

各参数说明如下：
* --id: 任务 id
* -k: 需求方的私钥, 表明了需求方的身份
* --conf: 使用的配置文件

### 发布预测任务

训练任务执行完成后产出预测模型，计算需求方可以提交预测任务，为预测数据计算出预测结果。

```
$ ./requester-cli task publish -a "linear-vl" -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 -t "predict" -n "房价任务v3" -d "hahahha" -p "id,id" --conf ./testdata/executor/node1/conf/config.toml -f "01d3b812-4dd7-4deb-a48d-4437312a164a,e02b27a6-0057-4673-b7ec-408ad060c952" -i fdc5b7e1-fc87-4e4b-86ee-b139a7721391
TaskID: a7dfac43-fa51-423e-bd05-8e0965c708a8
```

参数说明与发布训练任务差别在一个参数：
* -i: 指定训练任务的ID, 使用训练任务的产出

### 确认预测任务

与训练任务一致，任务执行节点需对预测任务进行确认。

```
$ ./executor-cli task confirm  --id  a7dfac43-fa51-423e-bd05-8e0965c708a8 --host 127.0.0.1:8123 -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21
```

### 启动预测任务

任务被各任务执行节点确认后，由计算需求方启动预测任务。

```
$ ./requester-cli task start --id a7dfac43-fa51-423e-bd05-8e0965c708a8 -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 --conf ./testdata/executor/node1/conf/config.toml
```

### 获取预测结果

预测任务执行成功后，计算需求方可以获取到预测的结果。

```
$ requester-cli task result --id a7dfac43-fa51-423e-bd05-8e0965c708a8 -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 --conf ./testdata/executor/node1/conf/config.toml  -o ./output.csv
```

各参数说明如下：
* --id: 预测任务的 id
* -k: 需求方的私钥
* --conf: 指定使用的配置文件
* -o: 预测结果的导出文件







