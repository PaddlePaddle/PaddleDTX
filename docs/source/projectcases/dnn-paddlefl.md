# 案例应用-神经网络算法测试

在本节中，我们使用 PaddleDTX 中的深度神经网络算法来解决波士顿房价预测问题。

您可以参考 [快速安装](../quickstart/quickstart.md) 来准备 PaddleDTX 的环境。

## 案例简介

与 [案例应用-线性回归算法测试](./linear.md) 中一致，本应用也是采用 UCI 机器学习数据库中的波士顿房屋信息数据进行训练和预测。为了帮助您理解 PaddleDTX 中使用的 PaddleFL DNN 算法，我们使用 [PaddlePaddle 框架中处理好的训练集和预测集](https://www.paddlepaddle.org.cn/documentation/docs/zh/1.8/api_cn/data_cn/dataset_cn/uci_housing_cn.html)。

数据集统计了波士顿郊区不动产税、城镇人均犯罪率等共计13个特征指标和平均房价，我们纵向拆分了数据，每部分包含不同的数据特征，分别由不同的数据持有方 A、B、C 进行持有，并通过 id 来标识同一条样本。

本案例我们通过 PaddleDTX 来进行三方的联合训练和预测。

具体样本文件说明如下：

1. 训练任务：任务执行节点 A 参与模型训练样本文件 [train_dataA.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/dnn_paddlefl/train_dataA.csv) ，任务执行节点 B 参与模型训练样本文件 [train_dataB.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/dnn_paddlefl/train_dataB.csv)，任务执行节点 C 参与模型训练样本文件 [train_dataB.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/dnn_paddlefl/train_dataC.csv)
2. 预测任务：任务执行节点 A 参与模型预测样本文件 [predict_dataA.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/dnn_paddlefl/predict_dataA.csv) ，任务执行节点 B 参与模型预测样本文件 [predict_dataB.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/dnn_paddlefl/predict_dataB.csv)，任务执行节点 C 参与模型预测样本文件 [predict_dataC.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/dnn_paddlefl/predict_dataC.csv)

## 测试脚本说明

本案例采用 [paddledtx_test.sh](https://github.com/PaddlePaddle/PaddleDTX/tree/master/scripts) 演示：
``` shell
Usage:
  ./paddledtx_test.sh <mode> [-f <sample files>] [-m <model task id>] [-i <task id>]
    <mode> - one of 'upload_sample_files', 'start_vl_linear_train', 'start_vl_linear_predict', 'start_vl_logistic_train'
         'start_vl_logistic_predict','start_vl_dnn_train', 'start_vl_dnn_predict', 'tasklist', 'gettaskbyid'
      - 'upload_sample_files' - save linear and logistic sample files into XuperDB
      - 'start_vl_linear_train' - start vertical linear training task
      - 'start_vl_linear_predict' - start vertical linear prediction task
      - 'start_vl_logistic_train' - start vertical logistic training task
      - 'start_vl_logistic_predict' - start vertical logistic prediction task
      - 'start_vl_dnn_train' - start vertical paddlefl-dnn training task
      - 'start_vl_dnn_predict' - start vertical paddlefl-dnn prediction task
      - 'tasklist' - list task in PaddleDTX
      - 'gettaskbyid' - get task by id from PaddleDTX
    -f <sample files> - linear or logistic sample files
    -e <model evaluation> - whether to perform model evaluation on the training task, default false, if select true, the evaluate rule is 'Cross Validation'
    -l <live model evaluation> - whether to perform live model evaluation, default false
    -m <model task id> - finished train task ID from which obtain the model, required for predict task
    -i <task id> - training or prediction task id

  ./paddledtx_test.sh -h (print this message), e.g.:

  ./paddledtx_test.sh upload_sample_files
  ./paddledtx_test.sh start_vl_linear_train -f 1ffc4504-6a62-45be-a7e3-191c708b901f,f8439128-bebb-47c2-a04d-1121dbc087a4
  ./paddledtx_test.sh start_vl_linear_predict -f cb40b8ad-db08-447f-a9d9-628b69d01660,2a8a45ab-3c5d-482e-b945-bc45b7e28bf9 -m 9b3ff4be-bfcd-4520-a23b-4aa6ea4d59f1
  ./paddledtx_test.sh start_vl_logistic_train -f b31f53a5-0f8b-4f57-a7ea-956f1c7f7991,f3dddade-1f52-4b9e-9253-835e9fc81901
  ./paddledtx_test.sh start_vl_logistic_predict -f 1e97d684-722f-4798-aaf0-dffe955a94ba,b51a927c-f73e-4b8f-a81c-491b9e938b4d -m d8c8865c-a837-41fd-802b-8bd754b648eb
  ./paddledtx_test.sh start_vl_dnn_train -f 34cf2ee3-81b2-4865-907d-a9eab3c5b384,9dc7e0b7-18dd-4d5a-a3a1-6dace6d04fc8,3eaee2ea-4680-4b0b-bde3-ab4a4949159e
  ./paddledtx_test.sh start_vl_dnn_predict -f 25ec6fd0-904e-4737-9bcc-c1cc11df1170,4442acae-90a2-4b92-b05f-cf1503c9d55e,73176b51-07f1-4f50-82c8-2d9d8908849b -m d8c8865c-a837-41fd-802b-8bd754b648eb
  ./paddledtx_test.sh gettaskbyid -i 9b3ff4be-bfcd-4520-a23b-4aa6ea4d59f1
  ./paddledtx_test.sh tasklist
```

## 上传样本文件

任务的发布与执行离不开样本文件，故在计算需求方发布任务之前，需确保数据持有方已上传各自所拥有的样本文件。

``` shell
# 上传样本文件
$ sh paddledtx_test.sh upload_sample_files

# 命令返回
Vertical linear train sample files: 688e4a1b-e9bf-4bfe-a13c-23ebb1d82850,19d4d284-6b1e-4a62-b421-40fdb6b7e787
Vertical linear prediction sample files: 9196f040-0743-4ae6-a1aa-b37f08c9bd3b,6d34fa49-5aac-409f-8973-a648d9309378
Vertical logistic train sample files: 9fb28896-eb6c-48f2-b356-2ab342a2aa6d,8b79fddd-3370-402c-ba9b-1f239156ec51
Vertical logistic prediction sample files: 96140537-8c7a-46cb-b2d3-0540e8cadc0e,abacaded-afdd-419d-bc52-0d90b5641aa2
PaddleFL train sample files: 34cf2ee3-81b2-4865-907d-a9eab3c5b384,9dc7e0b7-18dd-4d5a-a3a1-6dace6d04fc8,3eaee2ea-4680-4b0b-bde3-ab4a4949159e
PaddleFL prediction sample files: 25ec6fd0-904e-4737-9bcc-c1cc11df1170,4442acae-90a2-4b92-b05f-cf1503c9d55e,73176b51-07f1-4f50-82c8-2d9d8908849b
```

命令执行说明：

* **upload_sample_files** 命令会自动化执行如下2个步骤：
    1. 为数据持有方分别创建文件存储所需的命名空间
    2. 上传数据持有方拥有的训练和预测数据

每个步骤对应的客户端命令详情参考 [操作 XuperDB](../quickstart/client.md) 。

样本上传执行结果说明：

* PaddleFL train sample files值为A、B、C 三个数据持有方分别持有的波士顿房价预测样本文件
* PaddleFL prediction sample files值为A、B、C 三个数据持有方分别持有的波士顿房价训练样本文件
* upload_files.csv 存储了所需的样本文件ID

查看 XuperDB 中的样本文件：

``` shell
# 数据持有方A查询train_dataA.csv文件：
$ docker exec -it dataowner1.node.com sh -c './xdb-cli files getbyid -i 34cf2ee3-81b2-4865-907d-a9eab3c5b384 --host http://127.0.0.1:80'

# 数据持有方B查询train_dataB.csv文件：
$ docker exec -it dataowner2.node.com sh -c './xdb-cli files getbyid -i 9dc7e0b7-18dd-4d5a-a3a1-6dace6d04fc8 --host http://127.0.0.1:80'

# 数据持有方C查询train_dataC.csv文件：
$ docker exec -it dataowner3.node.com sh -c './xdb-cli files getbyid -i 3eaee2ea-4680-4b0b-bde3-ab4a4949159e --host http://127.0.0.1:80'

# 数据持有方A查询predict_dataA.csv文件：
$ docker exec -it dataowner1.node.com sh -c './xdb-cli files getbyid -i 25ec6fd0-904e-4737-9bcc-c1cc11df1170 --host http://127.0.0.1:80'

# 数据持有方B查询predict_dataB.csv文件：
$ docker exec -it dataowner2.node.com sh -c './xdb-cli files getbyid -i 4442acae-90a2-4b92-b05f-cf1503c9d55e --host http://127.0.0.1:80'

# 数据持有方B查询predict_dataC.csv文件：
$ docker exec -it dataowner3.node.com sh -c './xdb-cli files getbyid -i 73176b51-07f1-4f50-82c8-2d9d8908849b --host http://127.0.0.1:80'
```

## 训练任务

发布和启动训练任务：

``` shell
# -f 取值样本上传upload_sample_files命令返回的PaddleFL train sample files
$ sh paddledtx_test.sh start_vl_dnn_train -f 34cf2ee3-81b2-4865-907d-a9eab3c5b384,9dc7e0b7-18dd-4d5a-a3a1-6dace6d04fc8,3eaee2ea-4680-4b0b-bde3-ab4a4949159e

# 命令返回
Requester published dnn train task: TaskID: 96aaa3a1-5532-4d00-be6a-721540518be3
```
命令执行说明：

* **start_vl_dnn_train** 命令会自动化执行如下3个步骤：
    1. 计算需求方发布波士顿房价预测训练任务
    2. 数据持有方A/B/C授权任务执行节点A/B/C确认任务
    3. 计算需求方启动任务

每个步骤对应的客户端命令详情参考 [操作 Distributed AI](../quickstart/client.md) 。

查看训练任务：

```
$ sh paddledtx_test.sh gettaskbyid -i 96aaa3a1-5532-4d00-be6a-721540518be3
```

## 预测任务

发布和启动波士顿房价预测任务：

``` shell
# -f 取值样本上传upload_sample_files命令返回的Vertical linear prediction sample files，-m 取值为训练任务返回的TaskID
$ ./paddledtx_test.sh start_vl_dnn_predict -f c21b367f-2cb8-4859-87d8-18c52d397b13,043b9f55-68f6-4587-be8b-2340ea4432c2,b36442b6-ea3d-4530-910a-ec44291cd66c -m 96aaa3a1-5532-4d00-be6a-721540518be3

# 命令返回
Requester published dnn prediction task: TaskID: e83c2295-8913-4603-9150-ccf4498e76ce
Root mean square error of Boston house price prediction is: 5.513432645452959
```
命令执行说明：

* **start_vl_dnn_predict** 自动化执行如下5个步骤：
    1. 计算需求方发布波士顿房价预测任务
    2. 数据持有方A/B/C授权任务执行节点A/B/C确认任务
    3. 计算需求方启动任务
    4. 为计算需求方下载预测结果
    5. 计算模型的均方根误差

每个步骤对应的客户端命令详情参考 [操作 Distributed AI](../quickstart/client.md) 。

## 模型评估

通过预测任务，获得了模型的预测结果，我们通过计算预测值与真实值的均方根误差来评估模型优劣。

预测任务执行完成后，同时输出了波士顿房价预测模型的均方根误差，为 5.513432645452959 。

