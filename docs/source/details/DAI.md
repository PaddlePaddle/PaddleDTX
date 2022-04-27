# Distributed AI
Distributed AI 是PaddleDTX的计算层。一方面，它实现了可信的多方安全计算网络（SMPC），支持多个学习过程并行运行。另一方面，作为一个可扩展框架，它可以持续集成多种联邦学习算法。

## 1. 服务组件
Distributed AI 包含以下服务：

- Requester，计算需求节点，向区块链网络发布计算任务，从任务执行节点获取任务执行结果，并做可信性验证。
- Executor，任务执行节点，从区块链上获取要参与执行的任务，确认数据的使用权限后执行任务。做权限确认的过程也是数据持有节点对数据的可信性做背书的过程。

## 2. 多方安全计算框架
PaddleDTX实现的多方安全计算框架，具备以下特征：

- 任务维度动态构建计算网络
- 支持多个学习过程并行执行
- 可扩展，方便集成各种联邦学习算法
- 可执行模型评估和动态模型评估
- 以区块链、隐私计算、ACL技术为支撑，保证数据、模型的隐私性和可信性

<img src='../../_static/smpc.png' width = "100%" height = "100%" align="middle"/>

## 3. 可信联邦学习
PaddleDTX中，联邦学习分为训练过程和预测过程。计算需求方通过发布训练任务，任务执行节点会向数据持有节点做数据可信性背书，继而触发训练过程，最终得到满足条件的模型。如果有预测需求，计算需求方发布预测任务，任务执行节点会向数据持有节点做数据可信性背书，继而触发预测过程，最终得到预测结果。目前已集成的算法及其原理和实现，在 [crypto](./crypto.md#id2) 部分有更多体现。

## 4. 模型评估
一个训练任务的输入有两个，一个是算法，一个是训练集。计算需求方需要判断采用的算法是否能在训练集上训练出好的模型，模型评估可为判断提供依据。在商业应用中，模型训练往往以试验的方式开始，根据评估的指标，不断优化超参数，最终获取比较理想的超参数。

目前，PaddleDTX实现的模型评估，针对分布式、有监督的机器学习算法，可应用于任意已经实现的二分类算法、回归算法。如果计算需求方指定执行模型评估，在正常的训练任务结束后，参与训练的多方节点自动启动模型评估流程。

### 4.1 评估方式
PaddleDTX提供 3 中评估方式：Random Split（随机划分）、Cross Validation（交叉验证）、Leave One Out（留一法）。三者的区别在于划分训练集的方式不同，进而执行模型训练的次数不同，评估的耗时、成本也会随之不同。对于纵向联邦学习算法，在进行划分之前，会先对训练集做样本对齐，以保证多个计算节点在训练和预测过程中用到的数据集是一致的，并且不会浪费数据。

#### Random Split
随机打乱经过样本对齐的训练集，按照计算需求方发布任务时指定的比例（默认30%）选取数据集作为验证集，其余作为训练集用于模型训练。随机种子是对任务ID经过哈希计算得来，保证各个节点上的随机种子是一致的，这样最终得到的 2 个子集合也是一致的，在训练和预测过程中不会因为样本对齐而浪费数据。评估过程只进行 1 次分布式模型训练，1 次分布式预测验证。
#### Cross Validation
K 折交叉验证高效利用数据，计算成本适度，是最基本最常用的模型评估方式。训练集被划分为 K 个小的子集，每次训练的时候取其中的 K - 1 作为训练集，剩余的作为验证集。评估过程进行 K 次分布式模型训练，K 次分布式预测验证。PaddleDTX支持 5 折交叉验证和 10 折交叉验证。

如果计算需求方发布任务时指定打乱训练集，各方节点上运行的Evaluator会在样本对齐后随机打乱训练集。随机打乱训练集的算法和Random Split相同，保证多个计算节点最终得到的 K 个子集合是一致的。
#### Leave One Out
在Leave One Out中，每次用于预测验证的样本只有 1 条数据，其余用于模型训练。如果训练集中有 N 个样本，评估过程进行 N 次分布式模型训练，N 次分布式预测验证。这种方式虽然更加充分利用数据，但是计算成本更高，用时更长，并且模型也高度相似，最终计算的各类评估指标的偏差也会比较大。

### 4.2 评估指标
分类问题相关的指标：

``` proto linenums="1"
message BinaryClassCaseMetricScores {
    CaseType    caseType                                = 1;
    double      avgAccuracy                             = 2; // Accuracy 的平均值
    double      avgPrecision                            = 3; // Precision 的平均值
    double      avgRecall                               = 4; // Recall 的平均值
    double      avgF1Score                              = 5; // F1Score 的平均值
    double      avgAUC                                  = 6; // AUC 的平均值

    message Point {
        // ROC曲线上的一个点，表示为 [FPR, TPR, threshold]([x,y,threshold])
        repeated double p = 1; 
    } 

    // 每次 “训练-预测” 完成后，计算的各种指标值
    message MetricsPerFold {
        double accuracy     = 1;
        double precision    = 2;
        double recall       = 3;
        double F1Score      = 4;
        double AUC          = 5;
        repeated Point ROC  = 6;
    }
    map<int32, MetricsPerFold> metricsPerFold   = 7;
}
```

回归问题相关的指标：

``` proto linenums="1"
message RegressionCaseMetricScores {
    CaseType caseType           = 1;
    map<int32, double> RMSEs    = 2; // 每次 “训练-预测” 完成后，计算得到的 RMSE (Root Mean Squard Error) 
    double meanRMSE             = 3; // RMSE 的平均值
    double stdDevRMSE           = 4; // RMSE 的标准差
}
```

训练任务结束后，训练样本中含有目标特征的**任务执行节点**会生成模型评估结果，包含上述指标，保存于 ./evalus 路径下。

### 4.3 Evaluator
模型评估的步骤可以简述为：

1. 初始化
2. 划分数据集
3. 模型训练
4. 预测和验证
5. 计算评估指标
6. 结束并清理资源

Evaluator 接口定义：

``` go linenums="1"
type Evaluator interface {
    // Start 启动模型评估
    // 1、划分数据集, 采用 Random Split、Cross Validation、Leave One Out 中的一种方式
    // 2、实例化 Learners, 启动模型训练
    // fileRows 是经过样本对齐后的训练集
    Start(fileRows [][]string) error

    // Stop 关闭模型评估, 清理数据
    Stop()

    // SaveModel 当 Learner 训练结束后, 调用该方法保存模型
    // 如果成功训练出模型，触发实例化 Model, 启动预测过程
    SaveModel(*pbCom.TrainTaskResult) error

    // SavePredictOut 当 Model 预测结束后，调用该方法保存预测结果
    // 1、保存预测结果
    // 2、如果全部预测结束, 计算评估指标
    // 3、通知模型评估结束，并返回评估结果
    SavePredictOut(*pbCom.PredictTaskResult) error
}
```

## 5. 动态模型评估
如果算法实现对接了动态模型评估的接口，在模型训练的过程中，可以持续获得模型的阶段性评估结果。训练任务执行过程中，可依据每个阶段模型的评估结果，判断是否提前终止训练。当训练任务结束时，可获得一系列评估指标，展示训练效果变化趋势。

### 5.1 评估方式
采用 Random Split 方式。
### 5.2 评估指标
分类问题采用 Accuracy, 回归问题采用 RMSE 。

训练样本中含有目标特征的**任务执行节点**会生成阶段性模型评估结果，目前会在服务日志中体现，后续会写入可视化模块，训练过程中实时展示模型效果。

### 5.3 LiveEvaluator
动态模型评估的步骤可以简述为：

1. 初始化
2. 划分数据集
3. 初始化 1 个Learner
4. 模型训练，到达暂停轮数（pause round）停止
5. 预测和验证
6. 计算评估指标
7. 重复 4 - 6 直至接收到训练任务结束的消息
8. 结束并清理资源

LiveEvaluator 接口定义：

``` go linenums="1"
type LiveEvaluator interface {
    // Trigger 触发模型评估
    // 参数包含的消息分为两类：
    // 1 启动评估流程，参数中会携带训练集、暂停轮数（pause round），该消息触发初始化 1 个 Learner 并启动训练
    // 2 继续评估流程，该消息会触发继续训练直至到达 pause round，参数中会携带暂停轮数（pause round）
    Trigger(*pb.LiveEvaluationTriggerMsg) error

    // Stop 关闭模型评估, 清理数据
    Stop()

    // SaveModel 当 Learner 训练暂停后, 调用该方法保存模型
    // 实例化 Model, 启动预测过程
    SaveModel(*pbCom.TrainTaskResult) error

    // SavePredictOut 当 Model 预测结束后，调用该方法保存预测结果
    // 1、保存预测结果
    // 2、计算评估指标
    // 3、向可视化模块提交指标值
    SavePredictOut(*pbCom.PredictTaskResult) error
}
```

## 6. 接口与消息定义
[任务与相关接口定义](https://github.com/PaddlePaddle/PaddleDTX/tree/master/dai/protos/task)

[联邦学习过程接口定义](https://github.com/PaddlePaddle/PaddleDTX/tree/master/dai/protos/mpc/learners)

## 7. 配置说明
[Requester配置](../tutorial/dai-config.md#_2)

[Executor配置](../tutorial/dai-config.md#_3)

## 8. 命令行工具

[Requester命令使用说明](../tutorial/dai-cmd.md#_2)

[Executor命令使用说明](../tutorial/dai-cmd.md#_3)

<br>
