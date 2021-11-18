#!/bin/bash

# Copyright (c) 2021 PaddlePaddle Authors. All Rights Reserved.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Script to start paddleDTX and its dependent services with docker-compose.
# Usage: ./network_up.sh {start|stop|restart}

# PaddleDTX contract name
# PaddleDTX服务启动需要安装的智能合约名称
contractName=paddlempc
contractAccount=1111111111111112
# User's blockchain account address, used invoke contract
# 用户安装合约所使用的区块链账户地址
addressPath=../testdata/blockchain/user
transferAmount=110009887797
# Executor1 private key, same as paddledtx_test.sh's executor1PrivateKey
# 任务执行节点1私钥，同paddledtx_test.sh的executor1PrivateKey
executor1PrivateKey=14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21
# Executor2 private key, same as paddledtx_test.sh's executor2PrivateKey
# 任务执行节点2私钥，同paddledtx_test.sh的executor2PrivateKey
executor2PrivateKey=858843291fe4ed4bd2afc1120efd7315f3cae2d3f79e582f7df843ac6eb0543b
# The namespace of the sample files and prediction files store, same as paddledtx_test.sh's namespaces
# 样本文件存储的命名空间，同paddledtx_test.sh的namespaces
namespaces=paddlempc

function start() {
  # 1. Standardize Conf
  # 1. 标准化配置文件
  standardizeConf

  # 2. Start xchain network
  # 2. 启动区块链网络
  startXchain

  # 3. Compile and install paddleDTX contract
  # 3. 编译和安装智能合约
  installContract
  sleep 5

  # 4. Start decentralized storage network
  # 4. 启动去中心化存储网络
  startXdb
  sleep 5

  # 5. Start PaddleDTX
  # 5. 启动多方安全计算网络
  startPaddleDTX
}

function stop() {
  # Stop PaddleDTX
  # 停止多方安全计算网络
  print_blue "==========> Stop executor network ..."
  docker-compose -f ../testdata/executor/docker-compose.yml down
  # Stop decentralized storage network
  # 停止去中心化存储网络
  print_blue "==========> Stop decentralized storage network ..."
  docker-compose -f ../testdata/xdb/docker-compose.yml down
  # Stop xchain network
  # 停止区块链网络
  print_blue "==========> Stop xchain network ..."
  docker-compose -f ../testdata/blockchain/docker-compose.yml down
  print_green "==========> PaddleDTX stopped !"
}


function standardizeConf() {
  # Generate standard config.toml file
  # 生成标准配置文件
  sampleConfigFiles=`find $(dirname ${PWD})/testdata -name "config.toml.sample"`
  mnemonic=`cat $addressPath/mnemonic`
  for file in $sampleConfigFiles
  do
    conf=${file%.sample}
    eval "cat <<EOF
$(< $file)
EOF
"  > $conf
  done
}

# startXchain start xchain network with docker compose
# 通过docker-compose启动区块链网络
function startXchain() {
  print_blue "==========> Xchain network start ..."
  docker-compose -f ../testdata/blockchain/docker-compose.yml up -d
  sleep 6

  xchainContainers="xchain1.node.com xchain2.node.com xchain3.node.com"
  checkContainerStatus "$xchainContainers" "Xchain network"
  print_green "==========> Xchain network starts successfully !"
}

# startXdb start Decentralized storage network with docker compose
# 通过docker-compose启动去中心化存储网络
function startXdb() {
  print_blue "==========> Decentralized storage network start ..."
  docker-compose -f ../testdata/xdb/docker-compose.yml up -d
  sleep 6

  xdbContainers="dataowner1.node.com dataowner2.node.com storage1.node.com storage2.node.com storage3.node.com"
  checkContainerStatus "$xdbContainers" "Decentralized storage network"
  print_green "==========> Decentralized storage network starts successfully !"
}

# startPaddleDTX start paddleDTX with docker compose
# 通过docker-compose启动多网安全计算网络
function startPaddleDTX() {
  print_blue "==========> Executor network start ..."
  docker-compose -f ../testdata/executor/docker-compose.yml up -d
  sleep 6

  executorContainers="executor1.node.com executor2.node.com"
  checkContainerStatus "$executorContainers" "PaddleDTX"
  print_green "========================================================="
  print_green "          PaddleDTX starts successfully !                "
  print_green "========================================================="
}

# checkContainerStatus check container status
# 判断容器启动状态
function checkContainerStatus() {
  for containerName in $1
  do
    exist=`docker inspect --format '{{.State.Running}}' ${containerName}`
    if [ "${exist}" != "true" ]; then
      print_red "==========> $2 start error ..."
      exit 1
    fi
  done
}

function compileContract() {
  docker run -it --rm \
      -v $(dirname ${PWD}):/workspace \
      -v ~/.ssh:/root/.ssh \
      -w /workspace \
      -e GONOPROXY=**.baidu.com** \
      -e GONOSUMDB=* \
      -e GOPROXY=https://goproxy.baidu-int.com \
      -e GO111MODULE=on \
      golang:1.13.4 sh -c "cd dai && go build -o ../testdata/blockchain/contract/$contractName ./blockchain/xchain/contract"

  # Copy contract file to xchain1 container
  # 将本地合约编译结果拷贝到区块链节点1容器中
  docker cp ../testdata/blockchain/contract/$contractName xchain1.node.com:/home/work/xchain/$contractName
  rm -f ../testdata/blockchain/contract/$contractName
}

function installContract() {
  print_blue "==========> Install $contractName contract start ..."
  compileContract

  address=`cat $addressPath/address`
  echo "user address $address ..."

  # Transfer token from miner's account to user's account
  # 从矿工账户给用户账户转移token
  transferAddressResult=`docker exec -it xchain1.node.com sh -c "
    ./xchain-cli transfer --to $address --amount $transferAmount --keys ./data/keys --host xchain1.node.com:37101
  "`
  checkOperateResult "$transferAddressResult"
  # Get required fee to create the contract account
  # 获取创建合约账户所需要的fee
  accountFee=`docker exec -it xchain1.node.com sh -c "
    ./xchain-cli account new --account $contractAccount --host xchain1.node.com:37101 --keys ./user
  " | grep "The gas you cousume is" | awk -F: '{print $2}' | awk 'BEGIN{RS="\r";ORS="";}{print $0}' | awk '$1=$1'`

  # Create xchain contract account
  # 创建合约账户
  contractAccountResult=`docker exec -it xchain1.node.com sh -c "
    ./xchain-cli account new --account $contractAccount --host xchain1.node.com:37101 --fee $accountFee --keys ./user
  "`
  checkOperateResult "$contractAccountResult"

  print_blue "success create contract account, $contractAccountResult ..."

  # Transfer token from miner's account to contract account
  # 从矿工账户给用户创建的合约账户转移token
  transferAccountResult=`docker exec -it xchain1.node.com sh -c "
    ./xchain-cli transfer --to XC${contractAccount}@xuper --amount $transferAmount --keys ./data/keys  --host xchain1.node.com:37101
  "`
  checkOperateResult "$transferAccountResult"

  print_blue "success transfer amount to contract account, $transferAccountResult ..."

  # Ensure that the contract account is created successfully, and then install the contract
  # 确保合约账户创建成功后，再进行合约安装
  sleep 5
  # Get required fee for deploy contract
  # 获取安装合约所需要的fee
  contractFee=`docker exec -it xchain1.node.com sh -c "
    ./xchain-cli native deploy --account XC${contractAccount}@xuper --host xchain1.node.com:37101 --runtime go -a '{\"creator\":\"XC${contractAccount}@xuper\"}' --cname $contractName ./$contractName --keys ./user
  " | grep "The gas you cousume is" | awk -F: '{print $2}' | awk 'BEGIN{RS="\r";ORS="";}{print $0}' | awk '$1=$1'`

  # Install contract
  # 安装智能合约
  installResult=`docker exec -it xchain1.node.com sh -c "
    ./xchain-cli native deploy --account XC${contractAccount}@xuper --fee $contractFee  --host xchain1.node.com:37101 --runtime go -a '{\"creator\":\"XC${contractAccount}@xuper\"}' --cname $contractName ./$contractName --keys ./user
  "`
  checkOperateResult "$installResult"
  print_green "==========> Install $contractName contract successfully ! "
}

function checkOperateResult() {
  errMessage=`echo "$1" | grep "Error\|Fail\|err"`
  if [ "$errMessage" ]; then
    print_red "start paddleDTX error: $1"
    exit 1
  fi
}

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function print_green() {
  printf "${GREEN}%s${NC}\n" "$1"
}

function print_red() {
  printf "${RED}%s${NC}\n" "$1"
}

action=$1
case $action in
start)
  start $@
  ;;
stop)
  stop $@
  ;;
restart)
  stop $@
  start $@
  ;;
*)
  echo "Usage: $0 {start|stop|restart}"
  exit 1
  ;;
esac

