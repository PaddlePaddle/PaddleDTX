# 案例应用-逻辑回归算法测试

在本节中，我们使用 PaddleDTX 解决鸢尾花的分类问题，帮助您更好的理解 PaddleDTX。

您可以参考 [快速安装](../quickstart/quickstart.md) 来准备 PaddleDTX 的环境。

## 案例简介

鸢尾花卉数据集是常用的分类实验数据集，也来源于UCI 机器学习数据库，其中鸢尾花的种类分为三种：

* 山鸢尾 (Iris-setosa)
* 变色鸢尾 (Iris-versicolor)
* 维吉尼亚鸢尾 (Iris-virginica)

在每一条样本数据中包含了四项特征值和一个标签值，标签值为鸢尾花的种类，特征值包括：

* 花瓣长度 (Petal length)
* 花瓣宽度 (Petal width)
* 花萼长度 (Sepal length)
* 花萼宽度 (Sepal width)

鸢尾花的分类问题是二分类逻辑回归的经典案例。我们利用逻辑回归的算法构建模型，根据鸢尾花的花萼和花瓣大小来区分鸢尾花的品种。

我们从数据集中随机选取了部分数据作为测试集，其余为训练集，训练集为模型训练使用的样本数据，测试集用来验证我们的模型。
同时, 我们又将数据集纵向拆分为两部分，每部分包含不同的特征变量，分别由不同的数据持有方 A 和 B 进行持有，通过 id 来标识同一条样本。
本案例我们模拟分别持有部分特征变量数据的两方进行联合训练和预测。

具体样本文件说明如下：

1. 训练任务：任务执行节点 A 参与模型训练样本文件 [train_dataA.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/logic_iris_plants/train_dataA.csv) ，任务执行节点 B 参与模型训练样本文件 [train_dataB.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/logic_iris_plants/train_dataB.csv)
2. 预测任务：任务执行节点 A 参与模型预测样本文件 [predict_dataA.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/logic_iris_plants/predict_dataA.csv) ，任务执行节点 B 参与模型预测样本文件 [predict_dataB.csv](https://github.com/PaddlePaddle/PaddleDTX/blob/master/dai/mpc/testdata/vl/logic_iris_plants/predict_dataB.csv)

## 测试脚本说明

参考 [案例应用-线性回归算法测试-测试脚本说明](./linear.md)

## 上传样本文件

参考 [案例应用-线性回归算法测试-上传样本文件](./linear.md)

样本上传执行结果说明：

* Vertical logistic train sample files 值为数据持有方 A 上传 train_dataA.csv 和数据持有方 B 上传 train_dataB.csv 所得的鸢尾花品种分类样本文件ID
* Vertical logistic prediction sample files 值为数据持有方 A 上传predict_dataA.csv和数据持有方 B 上传 predict_dataB.csv 所得的鸢尾花品种分类的样本文件ID
* upload_files.csv 存储了鸢尾花训练和预测所需的样本文件ID

样本文件查看：

``` shell linenums="1"
# 数据持有方A查询train_dataA.csv文件：
$ docker exec -it dataowner1.node.com sh -c './xdb-cli files getbyid -i 9fb28896-eb6c-48f2-b356-2ab342a2aa6d --host http://127.0.0.1:80'

# 数据持有方B查询train_dataB.csv文件：
$ docker exec -it dataowner2.node.com sh -c './xdb-cli files getbyid -i 8b79fddd-3370-402c-ba9b-1f239156ec51 --host http://127.0.0.1:80'

# 数据持有方A查询predict_dataA.csv文件：
$ docker exec -it dataowner1.node.com sh -c './xdb-cli files getbyid -i 96140537-8c7a-46cb-b2d3-0540e8cadc0e --host http://127.0.0.1:80'

# 数据持有方B查询predict_dataB.csv文件：
$ docker exec -it dataowner2.node.com sh -c './xdb-cli files getbyid -i abacaded-afdd-419d-bc52-0d90b5641aa2 --host http://127.0.0.1:80'
```

## 训练任务

发布和启动训练任务：

``` shell linenums="1"
# -f 取值样本上传upload_sample_files命令返回的Vertical logistic train sample files
$ sh paddledtx_test.sh start_vl_logistic_train -f 9fb28896-eb6c-48f2-b356-2ab342a2aa6d,8b79fddd-3370-402c-ba9b-1f239156ec51

# 命令返回
Requester published logistic train task: TaskID: 95104913-c6cc-4520-bab1-2be814f0d81e
```

命令执行说明

* **start_vl_logistic_train** 命令会自动化执行如下3个步骤:
    1. 计算需求方发布鸢尾花品种分类训练任务
    2. 数据持有方A/B授权任务执行节点A/B确认任务
    3. 计算需求方启动任务

每个步骤对应的客户端命令详情参考 [操作 Distributed AI](../quickstart/client.md) 。

查看训练任务：

```
$ sh paddledtx_test.sh gettaskbyid -i 95104913-c6cc-4520-bab1-2be814f0d81e
```

## 预测任务

发布和启动预测任务：

``` shell linenums="1"
# -f 取值样本上传upload_sample_files命令返回的Vertical logistic prediction sample files，-m 取值为训练任务返回的TaskID
$ sh paddledtx_test.sh start_vl_logistic_predict -f 96140537-8c7a-46cb-b2d3-0540e8cadc0e,abacaded-afdd-419d-bc52-0d90b5641aa2 -m 95104913-c6cc-4520-bab1-2be814f0d81e

# 命令返回
Requester published logistic prediction task: TaskID: f86ff72b-fb94-4f8a-9797-9b42ae3ade84
Accuracy of Iris plants prediction is: 1.00
```

命令执行说明：

* **start_vl_logistic_predict** 自动化执行如下6个步骤:
    1. 计算需求方发布鸢尾花品种分类预测任务
    2. 数据持有方A/B授权任务执行节点A/B确认任务
    3. 计算需求方启动任务
    4. 计算需求方下载预测结果
    5. 计算模型的均方根误差
    6. 计算预测的准确率

每个步骤对应的客户端命令详情参考 [操作 Distributed AI](../quickstart/client.md) 。

## 模型评估

通过预测任务，获得了模型的预测结果，我们将置信度设置为0.5，通过计算预测值与真实值的分类准确率来评估模型优劣。

预测任务执行完成后，同时输出了鸢尾花品种分类模型的预测结果，在测试集上准确率为 100%。

<br>
