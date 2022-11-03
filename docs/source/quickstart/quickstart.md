# 快速安装

### 1.1 基础环境

我们为您提供了能够快速拉起 PaddleDTX 测试网络的脚本，在使用前需要您准备如下环境:

* docker，推荐版本18.03+ [点击下载安装 docker](https://docs.docker.com/get-docker/)
* docker-compose，推荐版本1.26.0+ [点击下载安装 docker-compose](https://github.com/docker/compose/releases)
* 如果使用Mac启动服务，Docker Desktop 至少设置为4GB 运行时内存，参考[Docker Desktop for Mac 用户手册](https://docs.docker.com/desktop/mac/)

### 1.2 网络启动

环境准备好之后，可以通过执行脚本快速拉起网络：
```
$ git clone git@github.com:PaddlePaddle/PaddleDTX.git
$ cd PaddleDTX/scripts
$ sh network_up.sh start

# 支持启动基于Fabric区块链网络的DAI，命令如下：
$ sh network_up.sh start -b fabric
# 支持启动基于ipfs存储网络的DAI，命令如下：
$ sh network_up.sh start -s ipfs
# 支持三方的DNN 算法，需要启动 PaddleFL 的节点，执行如下命令代替上述命令：
$ sh network_up.sh start -p true
# 支持界面可视化操作，启动paddledtx-visual时应指定可被浏览器访问的计算节点IP, 如在host为“106.13.169.234”的机器上启动PaddleDTX及其可视化服务，命令如下：
$ ./network_up.sh start -h 106.13.169.234
# 可视化服务启动之后，浏览器输入 http://106.13.169.234:8233/ 即可访问，用户使用前的节点配置参考网络启动说明。
```


使用脚本也可以快速销毁网络：
```
$ sh network_up.sh stop
```


销毁基于Fabric的DAI网络：
```
$ sh network_up.sh stop -b fabric
```
网络启动之后，当收到返回 **PaddleDTX starts successfully ! ** 即启动成功，用户可通过 ==./paddledtx_test.sh== 脚本，开启你的PaddleDTX初体验。

!!! info "说明"

    1、我们推荐用户采用Linux环境安装，若采用Mac启动，需修改docker运行资源限制，设置较高的Cpus（>4）、Memory（>4GB）、Swap（>4GB）。

    2、网络启动成功后，可通过docker ps查看脚本启动的服务，共包含3个区块链节点、3个数据持有节点、3个存储节点、3个可信计算节点，如果用户采用`sh network_up.sh start -s ipfs -p true -h 106.13.169.234`命令启动，则会再启动一个ipfs节点和3个paddlefl节点、1个可视化服务节点。

    3、如果用户无需进行模型训练，可以选择只启动去中心化存储网络（Xuperdb），参考 [XuperDB 服务启动和命令使用说明](https://github.com/PaddlePaddle/PaddleDTX/tree/master/xdb/scripts)：
    ``` shell

        # 启动基于Xchain的Xuperdb
        $ cd PaddleDTX/xdb/scripts
        $ sh network_up.sh start -b xchain

        # 启动基于Fabric网络的Xuperdb
        $ cd PaddleDTX/xdb/scripts
        $ sh network_up.sh start -b fabric

        # 启动采用ipfs存储网络的Xuperdb
        $ cd PaddleDTX/xdb/scripts
        $ sh network_up.sh start -b xchain -s ipfs 
    ```
    4、PaddleDTX支持用户通过可视化界面发布任务、授权样本文件，针对网络管理、节点权限管理、模型评估结果等可视化操作正在开发中，后续版本会开源。用户在使用可视化操作之前，需要导入如下配置paddledtx_setting.json：
    ```shell
        {
            "users": [
                {
                    "publicKey": "e790393685a359e37a73457b3eef55c87264a61c968e5c136b70b8b5e6941f3605a67561af41633035239f6393b949584470da7a67b5b8fe284bd69cfb0d3d59",
                    "privateKey": "f0f6ad5422b37bdf18f3ef6464ce682d7412f25b5f5f5e800454f195055bffb1",
                    "mnemonic": "提 现 详 责 腐 贪 沉 回 涨 谓 献 即",
                    "address": "eFHH6ovPcG6eMszLB4DxFWeY3EBPZ9Hrb",
                    "default": true
                }
            ],
            "contractName": "paddlempc",
            "node": "106.13.169.234:8908",
            "dataOwners": [
                {
                    "address": "106.13.169.234:8441",
                    "default": false
                },
                {
                    "address": "106.13.169.234:8442",
                    "default": true
                },
                {
                    "address": "106.13.169.234:8443",
                    "default": false
                }
            ]
        }
    # users: 计算需求节点账户设置，助记词默认配置为“提 现 详 责 腐 贪 沉 回 涨 谓 献 即”；
    # node：超级链节点，配置为浏览器可访问的区块链节点地址，可通过'curl http://106.13.169.234:8098/v1/get_bcchains'验证；
    # contractName：合约名称“paddlempc”；
    # dataOwners：配置数据持有节点的地址
    ```

### 1.3 任务发布和执行
./paddledtx_test.sh脚本提供了多种快捷操作，方便用户文件上传、下载、发布训练和预测任务等，快捷命令如下：
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
!!! info "说明"

    用户可通过cat ./paddledtx_test.sh查看脚本默认创建的文件存储命名空间、上传文件列表等，如有额外需求，可自定义配置；

    脚本执行的 start_vl_linear_train、start_vl_linear_predict、start_vl_logistic_train、start_vl_logistic_predict、start_vl_dnn_train、start_vl_dnn_predic 命令，本质为用户展示了多元线性回归、多元逻辑回归和神经网络算法的项目案例，参考 [项目案例](../projectcases/linear.md)

1. 上传训练及预测样本文件
   ```shell
   # upload_sample_files会为数据持有节点A/B/C创建数据存储的命名空间，并上传任务训练和预测所需的样本文件
   # 该命令共上传了14个文件，包括数据持有方A/B发布纵向线性回归、纵向逻辑回归训练和预测任务所需的8个样本文件，数据持有方A/B/C发布纵向深度神经网络训练和预测任务所需的6个样本文件
   ./paddledtx_test.sh upload_sample_files

   # 执行后，命令返回：
   # Vertical linear train sample files：纵向线性训练任务所需样本ID
   # Vertical linear prediction sample files：纵向线性预测任务所需样本ID
   # Vertical logistic train sample files：纵向逻辑回归训练任务所需样本ID
   # Vertical logistic prediction sample files：纵向逻辑回归预测任务所需样本ID
   # PaddleFL train sample files：纵向深度神经网络训练任务所需样本ID
   # PaddleFL prediction sample files：纵向深度神经网络预测任务所需样本ID
   ```

2. 启动纵向线性回归训练任务，$vlLinTrainfiles 取值为 **步骤1** 获取到的 Vertical linear train sample files
    ``` shell
    # 发布纵向线性回归训练任务，不启动模型评估
    $./paddledtx_test.sh start_vl_linear_train -f $vlLinTrainfiles

    # 发布纵向线性回归训练任务，启动模型评估
    $./paddledtx_test.sh start_vl_linear_train -f $vlLinTrainfiles -e true

    # 发布纵向线性回归训练任务，启动动态模型评估
    $./paddledtx_test.sh start_vl_linear_train -f $vlLinTrainfiles -l true

    # 发布成功后，会返回训练任务ID
    ```

3. 启动纵向线性回归预测任务，$vlLinPredictfiles 取值为 **步骤1** 获取到的 Vertical linear prediction sample files，$linearModelTaskId 取值为 **步骤2** 返回的模型训练任务ID，发布预测任务前，请确保 **步骤2** 已经训练完成
    ``` shell
    # 用户可通过 ./paddledtx_test.sh gettaskbyid -i $taskID 查看任务状态

    # 发布预测任务
    $./paddledtx_test.sh start_vl_linear_predict -f $vlLinPredictfiles -m $linearModelTaskId
    ```

4. 启动纵向逻辑回归训练任务，$vlLogTrainfiles 取值为 **步骤1** 获取到的 Vertical logistic train sample files
    ``` shell
    # 发布纵向逻辑回归训练任务，不启动模型评估
    $./paddledtx_test.sh start_vl_logistic_train -f $vlLogTrainfiles

    # 发布纵向逻辑回归训练任务，启动模型评估
    $./paddledtx_test.sh start_vl_logistic_train -f $vlLogTrainfiles -e true

    # 发布纵向逻辑回归训练任务，启动动态模型评估
    $./paddledtx_test.sh start_vl_logistic_train -f $vlLogTrainfiles -l true

    # 发布成功后，会返回训练任务ID
    ```

5. 启动纵向逻辑回归预测任务，$vlLogPredictfiles 取值为 **步骤1** 获取到的 Vertical logistic prediction sample files，$logisticModelTaskId 取值为 **步骤4** 返回的模型训练任务ID，发布预测任务前，请确保 **步骤4** 已经训练完成
    ``` shell
    # 用户可通过 ./paddledtx_test.sh gettaskbyid -i $taskID 查看任务状态

    # 发布预测任务
    $./paddledtx_test.sh start_vl_logistic_predict -f $vlLogPredictfiles -m $logisticModelTaskId
    ```

6. 任务列表查询
    ``` shell
    # 脚本方式
    $ ./paddledtx_test.sh tasklist

    # 命令行方式：
    $ docker exec -it executor1.node.com sh -c " 
    ./executor-cli  task list  --host 127.0.0.1:80 -p 6cb69efc0439032b0d0f52bae1c9aada3f8fb46a5f24fa99065910055b77a1174d4afbac3c0529c8927587bb0e2ad90a85eaa600cfddd6b99f1212112135ef2b
    "
    ```

7. 根据任务ID查询任务
    ``` shell
    # 脚本方式
    # taskID 为目标任务ID
    $ ./paddledtx_test.sh gettaskbyid -i $taskID

    # 命令行方式：
    $ docker exec -it executor1.node.com sh -c "./executor-cli task getbyid --host 127.0.0.1:80 -i $taskID"
    ```
