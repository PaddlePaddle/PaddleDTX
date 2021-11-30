# TestData
基于docker-compose一键启动PaddleDTX分布式机器学习所需的节点配置文件及docker-compose.yml文件.

## 一、文件目录说明
- blockchain【区块链网络】: 三个节点的xchain网络
    - blockchain: 区块链节点配置目录，包含三个网络节点账户配置
    - user: 用户安装合约所使用的区块链账户地址，包含用户的助记词、公私钥等信息
    - docker-compose.yml: xchain网络启动所需的配置

- xdb【去中心化存储服务】：两个数据持有节点，三个存储节点
    - data1: 数据持有节点1配置目录
    - data2: 数据持有节点2配置目录
    - storage1: 存储节点1配置目录
    - storage2: 存储节点2配置目录
    - storage3: 存储节点3配置目录
    - docker-compose.yml: 去中心化存储服务启动所需的配置
    
- executor【多方安全计算网络】：两个任务执行节点，与两个数据持有节点分别对应
    - node1: 任务执行节点1配置目录，节点私钥配置同data1中privateKey配置
    - node2: 任务执行节点2配置目录，节点私钥配置同data2中privateKey配置
    - docker-compose.yml:多方安全计算网络启动所需的配置

## 二、准备工作
服务启动依赖docker-compose，请先确认本地是否安装docker-compose

## 三、服务启动及任务执行
参考[README](../scripts/README.md)
