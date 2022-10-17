# PaddleDTX 服务启动和命令使用说明

## 一、服务启动和停止 [./network_up.sh]
Usage: ./network_up.sh { start | stop | restart }

### 1.1 服务启动
- -b $blockchainType，网络类型，取值xchain或fabric，默认xchain网络
- -s $storageModeType，底层存储引擎，本地存储为 local，IPFS 为 ipfs，默认 local
```shell
./network_up.sh start -b $blockchainType -s $storageModeType
```

- -p 如果需要使用 PaddleFL 的能力, 需要启动 PaddleFL 的容器, 默认为不启动
```shell
./network_up.sh start -p true
```

- -h 如果需要使用PaddleDTX可视化能力，需启动可视化服务paddledtx-visual，启动时指定可被浏览器访问的计算节点IP地址，默认为不启动。在host为“106.13.169.234”的机器上启动PaddleDTX及其可视化服务，命令如下：
```shell
./network_up.sh start -h 106.13.169.234
```
paddledtx-visual启动之后，浏览器输入 http://106.13.169.234:8233/ 即可访问，使用之前需要输入区块链节点、数据持有节点相关信息，可直接导入如下配置paddledtx_setting.json：
```
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
```

### 1.2 服务停止
```shell
./network_up.sh stop -b $blockchainType
```
### 1.3 服务重启
```shell
./network_up.sh restart -b $blockchainType -s $storageModeType
```

## 二、任务发布和执行 [./paddledtx_test.sh]
Usage: ./paddledtx_test.sh {upload_sample_files | start_vl_linear_train | start_vl_linear_predict | start_vl_logistic_train | start_vl_logistic_predict | tasklist | gettaskbyid}

### 2.1 上传训练及预测样本文件
```shell
./paddledtx_test.sh upload_sample_files
```

### 2.2 启动纵向线性回归训练任务
- vlLinTrainfiles 取值为步骤2.1获取到的 vertical linear train sample files
#### 2.2.1 发布纵向线性回归训练任务，不启动模型评估
```shell
./paddledtx_test.sh start_vl_linear_train -f $vlLinTrainfiles
```
#### 2.2.2 发布纵向线性回归训练任务，启动模型评估
```shell
./paddledtx_test.sh start_vl_linear_train -f $vlLinTrainfiles -e true
```
#### 2.2.3 发布纵向线性回归训练任务，启动动态模型评估
```shell
./paddledtx_test.sh start_vl_linear_train -f $vlLinTrainfiles -l true
```

### 2.3 启动纵向线性回归预测任务
- vlLinPredictfiles 取值为步骤2.1获取到的 vertical linear predict sample files
- linearModelTaskId 取值为步骤2.2的模型训练任务ID
- *请确保2.2训练任务已经完成*
```shell
./paddledtx_test.sh start_vl_linear_predict -f $vlLinPredictfiles -m $linearModelTaskId
```

### 2.4 启动纵向逻辑回归训练任务
- vlLogTrainfiles 取值为步骤2.1获取到的 vertical logistic train sample files
#### 2.4.1 发布纵向逻辑回归训练任务，不启动模型评估
```shell
./paddledtx_test.sh start_vl_logistic_train -f $vlLogTrainfiles
```
#### 2.4.2 发布纵向逻辑回归训练任务，启动模型评估
```shell
./paddledtx_test.sh start_vl_logistic_train -f $vlLogTrainfiles -e true
```
#### 2.4.3 发布纵向逻辑回归训练任务，启动动态模型评估
```shell
./paddledtx_test.sh start_vl_logistic_train -f $vlLogTrainfiles -l true
```

### 2.5 启动纵向逻辑回归预测任务
- vlLogPredictfiles 取值为步骤2.1获取到的 vertical logistic predict sample files
- logisticModelTaskId 取值为步骤2.4的模型任务ID
- *请确保2.4训练任务已经完成*
```shell
./paddledtx_test.sh start_vl_logistic_predict -f $vlLogPredictfiles -m $logisticModelTaskId
```

## 三、任务列表查询
### 3.1 脚本方式
```shell
./paddledtx_test.sh tasklist
```

### 3.2 命令行方式
```shell
docker exec -it executor1.node.com sh -c " 
./executor-cli  task list  --host 127.0.0.1:80 -p 6cb69efc0439032b0d0f52bae1c9aada3f8fb46a5f24fa99065910055b77a1174d4afbac3c0529c8927587bb0e2ad90a85eaa600cfddd6b99f1212112135ef2b
"
```

### 四、单个任务查询
### 4.1 脚本方式
- taskID 为目标任务ID
```shell
./paddledtx_test.sh gettaskbyid -i $taskID
```

### 4.2 命令行方式
- taskID 为目标任务ID
```shell
docker exec -it executor1.node.com sh -c "./executor-cli task getbyid --host 127.0.0.1:80 -i $taskID"
```
