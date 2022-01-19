# 案例应用-线性回归算法测试

在本节中，我们使用 PaddleDTX 解决波士顿房价预测问题，帮助您更好的理解 PaddleDTX。

您可以参考 [快速安装](./quickstart.md) 来准备 PaddleDTX 的环境。

## 案例简介
本案例中我们使用了来自 UCI 机器学习数据库中的波士顿房屋信息数据。该数据集统计了波士顿郊区不动产税、城镇人均犯罪率等共计13个特征指标和平均房价，我们通过机器学习找到特征指标和房价之间的关系，进而预测该地区房价，这是一个典型线性回归计算案例。

以下是数据集中的字段含义, 特征变量为:

* CRIM: 城镇人均犯罪率
* ZN: 住宅用地超过 25000 sq.ft. 的比例
* INDUS: 城镇非零售商用土地的比例
* CHAS: 边界是河流为1，否则0
* NOX: 一氧化氮浓度
* RM: 住宅平均房间数
* AGE: 1940年之前建成的自用房屋比例
* DIS: 到波士顿5个中心区域的加权距离
* RAD: 辐射性公路的靠近指数
* TAX: 每10000美元的全值财产税率
* PTRATIO: 城镇师生比例
* B: 城镇中黑人比例
* LSTAT: 人口中地位低下者的比例

目标变量(也称为标签变量)为:
* MEDV: 房价中位数

我们从数据集中随机选取了部分数据作为测试集，其余为训练集，训练集为模型训练使用的样本数据，测试集用来验证我们的模型。
同时, 我们又将数据集纵向拆分为两部分，每部分包含不同的特征变量，分别由不同的数据持有方 A 和 B 进行持有，通过 id 来标识同一条样本。
本案例我们模拟分别持有部分特征变量数据的两方进行联合训练和预测。

具体样本文件说明如下：
1. 训练任务：任务执行节点 A 参与模型训练样本文件 [train_dataA.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/linear_boston_housing/train_dataA.csv) ，任务执行节点 B 参与模型训练样本文件 [train_dataB.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/linear_boston_housing/train_dataB.csv)
2. 预测任务：任务执行节点 A 参与模型预测样本文件 [predict_dataA.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/linear_boston_housing/predict_dataA.csv) ，任务执行节点 B 参与模型预测样本文件 [predict_dataB.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/linear_boston_housing/predict_dataB.csv)

## 测试脚本说明

本案例采用 [paddledtx_test.sh](https://github.com/PaddlePaddle/PaddleDTX/tree/master/scripts) 演示：
```
Usage:
  ./paddledtx_test.sh <mode> [-f <sample files>] [-m <model task id>] [-i <task id>]
    <mode> - one of 'upload_sample_files', 'start_vl_linear_train', 'start_vl_linear_predict', 'start_vl_logistic_train'
         'start_vl_logistic_predict', 'tasklist', 'gettaskbyid'
      - 'upload_sample_files' - save linear and logistic sample files into XuperDB
      - 'start_vl_linear_train' - start vertical linear training task
      - 'start_vl_linear_predict' - start vertical linear prediction task
      - 'start_vl_logistic_train' - start vertical logistic training task
      - 'start_vl_logistic_predict' - start vertical logistic prediction task
      - 'tasklist' - list task in PaddleDTX
      - 'gettaskbyid' - get task by id from PaddleDTX
    -f <sample files> - linear or logistic sample files
    -m <model task id> - finished train task ID from which obtain the model, required for predict task
    -i <task id> - training or prediction task id

  ./paddledtx_test.sh -h (print this message), e.g.:

  ./paddledtx_test.sh upload_sample_files
  ./paddledtx_test.sh start_vl_linear_train -f 1ffc4504-6a62-45be-a7e3-191c708b901f,f8439128-bebb-47c2-a04d-1121dbc087a4
  ./paddledtx_test.sh start_vl_linear_predict -f cb40b8ad-db08-447f-a9d9-628b69d01660,2a8a45ab-3c5d-482e-b945-bc45b7e28bf9 -m 9b3ff4be-bfcd-4520-a23b-4aa6ea4d59f1
  ./paddledtx_test.sh start_vl_logistic_train -f b31f53a5-0f8b-4f57-a7ea-956f1c7f7991,f3dddade-1f52-4b9e-9253-835e9fc81901
  ./paddledtx_test.sh start_vl_logistic_predict -f 1e97d684-722f-4798-aaf0-dffe955a94ba,b51a927c-f73e-4b8f-a81c-491b9e938b4d -m d8c8865c-a837-41fd-802b-8bd754b648eb
  ./paddledtx_test.sh gettaskbyid -i 9b3ff4be-bfcd-4520-a23b-4aa6ea4d59f1
  ./paddledtx_test.sh tasklist
```

## 上传样本文件

任务的发布与执行离不开样本文件，故在计算需求方发布任务之前，需确保数据持有方已上传各自所拥有的样本文件。

```
# 上传样本文件
$ sh paddledtx_test.sh upload_sample_files

# 命令返回
Vertical linear train sample files: 688e4a1b-e9bf-4bfe-a13c-23ebb1d82850,19d4d284-6b1e-4a62-b421-40fdb6b7e787
Vertical linear prediction sample files: 9196f040-0743-4ae6-a1aa-b37f08c9bd3b,6d34fa49-5aac-409f-8973-a648d9309378
Vertical logistic train sample files: 9fb28896-eb6c-48f2-b356-2ab342a2aa6d,8b79fddd-3370-402c-ba9b-1f239156ec51
Vertical logistic prediction sample files: 96140537-8c7a-46cb-b2d3-0540e8cadc0e,abacaded-afdd-419d-bc52-0d90b5641aa2
```

命令执行说明：

* **upload_sample_files** 命令会自动化执行如下2个步骤：
    1. 为数据持有方A与B分别创建文件存储所需的命名空间
    2. 上传数据持有方A与B拥有的波士顿房价预测和鸢尾花数据分类所需的训练及预测样本文件

每个步骤对应的客户端命令详情参考 [操作 XuperDB](./client.md) 。

样本上传执行结果说明：

* Vertical linear train sample files值为数据持有方A上传train_dataA.csv和数据持有方B上传train_dataB.csv后所得的波士顿房价训练样本文件ID
* Vertical linear prediction sample files值为数据持有方A上传predict_dataA.csv和数据持有方B上传predict_dataB.csv所得的波士顿房价预测样本文件ID
* upload_files.csv 存储了波士顿房价训练和预测所需的样本文件ID

查看 XuperDB 中的样本文件：

```
# 数据持有方A查询train_dataA.csv文件：
$ docker exec -it dataowner1.node.com sh -c './xdb-cli files getbyid -i 688e4a1b-e9bf-4bfe-a13c-23ebb1d82850 --host http://127.0.0.1:80'

# 数据持有方B查询train_dataB.csv文件：
$ docker exec -it dataowner2.node.com sh -c './xdb-cli files getbyid -i 19d4d284-6b1e-4a62-b421-40fdb6b7e787 --host http://127.0.0.1:80'

# 数据持有方A查询predict_dataA.csv文件：
$ docker exec -it dataowner1.node.com sh -c './xdb-cli files getbyid -i 9196f040-0743-4ae6-a1aa-b37f08c9bd3b --host http://127.0.0.1:80'

# 数据持有方B查询predict_dataB.csv文件：
$ docker exec -it dataowner2.node.com sh -c './xdb-cli files getbyid -i 6d34fa49-5aac-409f-8973-a648d9309378 --host http://127.0.0.1:80'
```

## 训练任务

发布和启动训练任务：

```
# -f 取值样本上传upload_sample_files命令返回的Vertical linear train sample files
$ sh paddledtx_test.sh start_vl_linear_train -f 688e4a1b-e9bf-4bfe-a13c-23ebb1d82850,19d4d284-6b1e-4a62-b421-40fdb6b7e787

# 命令返回
Requester published linear train task: TaskID: 91d9c0b7-996b-4954-86e8-95048e91a3b8
```
命令执行说明：
* **start_vl_linear_train** 命令会自动化执行如下3个步骤：
    1. 计算需求方发布波士顿房价预测训练任务
    2. 数据持有方A/B授权任务执行节点A/B确认任务
    3. 计算需求方启动任务

每个步骤对应的客户端命令详情参考 [操作 Distributed AI](./client.md) 。

查看训练任务：

```
$ sh paddledtx_test.sh gettaskbyid -i 91d9c0b7-996b-4954-86e8-95048e91a3b8
```

## 预测任务

发布和启动波士顿房价预测任务：

```
# -f 取值样本上传upload_sample_files命令返回的Vertical linear prediction sample files，-m 取值为训练任务返回的TaskID
$ sh paddledtx_test.sh start_vl_linear_predict -f 9196f040-0743-4ae6-a1aa-b37f08c9bd3b,6d34fa49-5aac-409f-8973-a648d9309378 -m 91d9c0b7-996b-4954-86e8-95048e91a3b8

# 命令返回
Requester published linear prediction task: TaskID: 44e1cc47-4cb4-4d9c-a27e-81949182d2a4
Root mean square error of Boston house price prediction is: 4.568173732971698
```
命令执行说明：
* **start_vl_linear_predict** 自动化执行如下5个步骤：
    1. 计算需求方发布波士顿房价预测任务
    2. 数据持有方A/B授权任务执行节点A/B确认任务
    3. 计算需求方启动任务
    4. 为计算需求方下载预测结果
    5. 计算模型的均方根误差

每个步骤对应的客户端命令详情参考 [操作 Distributed AI](./client.md) 。

## 模型评估

通过预测任务，获得了模型的预测结果，我们通过计算预测值与真实值的均方根误差来评估模型优劣。

预测任务执行完成后，同时输出了波士顿房价预测模型的均方根误差，为 4.568173732971698。
