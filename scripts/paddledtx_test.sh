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

# Script to test PaddleDTX service, start train or predict task with docker-compose.
# Usage: ./paddledtx_test.sh {upload_sample_files | start_vl_linear_train | start_vl_linear_predict | start_vl_logistic_train | start_vl_logistic_predict | tasklist | gettaskbyid}

set -e
# Requester private key 
# 计算需求方私钥
requesterKey="40816c779f624a8fbc4e37be1ef8bbddc6c07b5f91e704953a599f9080458f60"
# Requester public key 
# 计算需求方公钥
requesterPublicKey="6cb69efc0439032b0d0f52bae1c9aada3f8fb46a5f24fa99065910055b77a1174d4afbac3c0529c8927587bb0e2ad90a85eaa600cfddd6b99f1212112135ef2b"
# Executor1 private key, same as network_up.sh's executor1PrivateKey
# 任务执行节点1私钥，同network_up.sh的executor1PrivateKey
executor1PrivateKey="14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21"
executor1PublicKey="4637ef79f14b036ced59b76408b0d88453ac9e5baa523a86890aa547eac3e3a0f4a3c005178f021c1b060d916f42082c18e1d57505cdaaeef106729e6442f4e5"
# Executor2 private key, same as network_up.sh's executor2PrivateKey
# 任务执行节点2私钥，同network_up.sh的executor2PrivateKey
executor2PrivateKey="858843291fe4ed4bd2afc1120efd7315f3cae2d3f79e582f7df843ac6eb0543b"
executor2PublicKey="e4530d81ccddc478978070e8f9fcc9f101dfc3b5c3ca1519c522c5e9698f394a35aab9145f242765185689a64b7338e9929c6a32e09050ff15645bb121ce1754"
config="./conf/config.toml"
# The namespace of the sample file store, same as network_up.sh's namespaces
# 样本文件存储的命名空间，同network_up.sh的namespaces
namespaces=paddlempc

# Parameters required for task training or prediction
# 训练及预测任务所需的参数
psi="id,id"
vlLinAlgo="linear-vl"
vlLogAlgo="logistic-vl"
vlLinLabel="MEDV"
vlLogLabel="Label"
vlLogLabelName="Iris-setosa"
vlLinTaskTrainName="boston_housing_train"
vlLinTaskPredictName="boston_housing_predict"
vlLogTaskTrainName="iris_plants_train"
vlLogTaskPredictName="iris_plants_predict"
taskNum=1
#alpha=0.1
amplitude=0.0001
#batch=4

function uploadSampleFiles() {
  # 1. Create a namespace for the sample file store
  # 1. 创建文件存储的命名空间
  createNamespaces
  
  # 2. Upload linear training file
  # 2. 上传线性训练文件
  sleep 1
  uploadLinearTrainSampleFile

  # 3. Upload linear prediction file
  # 3. 上传线性预测文件
  sleep 1
  uploadLinearPredictSampleFile

  # 4. Upload logic training files
  # 4. 上传逻辑训练文件
  sleep 1
  uploadLogisticTrainSampleFile

  # 5. Upload logic prediction file
  # 5. 上传逻辑预测文件
  sleep 1
  uploadLogisticPredictSampleFile
}

function createNamespaces() {
  # 1. Create namespace for dataOwner1
  # 1. 数据持有节点1创建命名空间
  data1AddNsResult=`docker exec -it dataowner1.node.com sh -c "
    ./xdata-cli files addns  --host http://dataowner1.node.com:80 -k $executor1PrivateKey -n $namespaces -r 2"`
  echo "======> DataOwner1 create files storage namespaces result: $data1AddNsResult"
  isData1AddNsOk=$(echo $data1AddNsResult | sed -e 's/\r//g' | sed 's/ //g')
  if [ "$isData1AddNsOk" != "OK" ]; then
    exit 1
  fi
  # 2. Create namespace for dataOwner2
  # 2. 数据持有节点2创建命名空间
  data2AddNsResult=`docker exec -it dataowner2.node.com sh -c "
    ./xdata-cli files addns --host http://dataowner2.node.com:80 -k $executor2PrivateKey  -n $namespaces  -r 2 
  "`
  echo "======> DataOwner2 create files storage namespaces result: $data2AddNsResult"
  isData2AddNsOk=$(echo $data2AddNsResult | sed -e 's/\r//g' | sed 's/ //g')
  if [ "$isData2AddNsOk" != "OK" ]; then
    exit 1
  fi
}

# uploadLinearTrainSampleFile dataOwner1 and dataOwner2 upload vertical linear train sample files
# 数据持有节点1和数据持有节点2上传纵向线性训练样本文件
function uploadLinearTrainSampleFile() {
  sampleFileAName=train_dataA.csv
  sampleFileBName=train_dataB.csv
  fileAName="linear_"$sampleFileAName
  fileBName="linear_"$sampleFileBName

  # DataOwner1 uploads linear train sample files
  # 数据持有节点1上传纵向线性训练样本文件
  data1Samples=`docker exec -it dataowner1.node.com sh -c "
    ./xdata-cli files upload --host http://dataowner1.node.com:80  -e '2022-06-21 15:59:05' -n $namespaces -m $fileAName -k $executor1PrivateKey \
    --ext '{\"FileType\":\"csv\",\"Features\":\"id,CRIM,ZN,INDUS,CHAS,NOX,RM\", \"TotalRows\":456}' -i /home/mpc-data/linear_boston_housing/$sampleFileAName
  "`
  echo "======> DataOwner1 upload vertical_linear_train sample file: $data1Samples"

  data1FileUploadRes=$(echo $data1Samples | sed -e 's/\r//g' | sed 's/ //g')
  data1FileId=${data1FileUploadRes##*:}

  sleep 3
  # DataOwner2 uploads linear train sample files
  # 数据持有节点2上传纵向线性训练样本文件
  data2Samples=`docker exec -it dataowner2.node.com sh -c "
    ./xdata-cli files upload --host http://dataowner2.node.com:80  -e '2022-06-21 15:59:05' -n $namespaces -m $fileBName -k $executor2PrivateKey \
    --ext '{\"FileType\":\"csv\",\"Features\":\"id,AGE,DIS,RAD,TAX,PTRATIO,B,LSTAT,MEDV\",\"TotalRows\":456}' -i /home/mpc-data/linear_boston_housing/$sampleFileBName
  "`
  echo "======> DataOwner2 upload vertical_linear_train sample file: $data2Samples"

  data2FileUploadRes=$(echo $data2Samples | sed -e 's/\r//g' | sed 's/ //g')
  data2FileId=${data2FileUploadRes##*:}

  files="$data1FileId,$data2FileId"

  printf "\033[0;32m%s\033[0m\n" "======> Vertical linear train sample files: $files"
}

# uploadLinearPredictSampleFile dataOwner1 and dataOwner2 upload vertical linear predict sample files
# 数据持有节点1和数据持有节点2上传纵向线性预测样本文件
function uploadLinearPredictSampleFile() {
  sampleFileAName=predict_dataA.csv
  sampleFileBName=predict_dataB.csv
  fileAName="linear_"$sampleFileAName
  fileBName="linear_"$sampleFileBName

  # DataOwner1 uploads linear predict sample files
  # 数据持有节点1上传纵向线性预测样本文件
  data1Samples=`docker exec -it dataowner1.node.com sh -c "
    ./xdata-cli files upload --host http://dataowner1.node.com:80  -e '2022-06-21 15:59:05' -n $namespaces -m $fileAName -k $executor1PrivateKey \
    --ext '{\"FileType\":\"csv\",\"Features\":\"id,CRIM,ZN,INDUS,CHAS,NOX,RM\", \"TotalRows\":50}' -i /home/mpc-data/linear_boston_housing/$sampleFileAName
  "`
  echo "======> DataOwner1 upload vertical_linear_predict sample file: $data1Samples"

  data1FileUploadRes=$(echo $data1Samples | sed -e 's/\r//g' | sed 's/ //g')
  data1FileId=${data1FileUploadRes##*:}

  # DataOwner2 uploads linear predict sample files
  # 数据持有节点2上传纵向线性预测样本文件
  data2Samples=`docker exec -it dataowner2.node.com sh -c "
    ./xdata-cli files upload --host http://dataowner2.node.com:80  -e '2022-06-21 15:59:05' -n $namespaces -m $fileBName -k $executor2PrivateKey \
    --ext '{\"FileType\":\"csv\",\"Features\":\"id,AGE,DIS,RAD,TAX,PTRATIO,B,LSTAT\",\"TotalRows\":50}' -i /home/mpc-data/linear_boston_housing/$sampleFileBName
  "`
  echo "======> DataOwner2 upload vertical_linear_predict sample file: $data2Samples"

  data2FileUploadRes=$(echo $data2Samples | sed -e 's/\r//g' | sed 's/ //g')
  data2FileId=${data2FileUploadRes##*:}

  files="$data1FileId,$data2FileId"
  printf "\033[0;32m%s\033[0m\n" "======> Vertical linear predict sample files: $files"
}

# uploadLogisticTrainSampleFile dataOwner1 and dataOwner2 upload vertical logistic train sample files
# 数据持有节点1和数据持有节点2上传纵向逻辑训练样本文件
function uploadLogisticTrainSampleFile() {
  sampleFileAName=train_dataA.csv
  sampleFileBName=train_dataB.csv
  fileAName="logistic_"$sampleFileAName
  fileBName="logistic_"$sampleFileBName

  # DataOwner1 uploads logistic train sample files
  # 数据持有节点1上传纵向逻辑训练样本文件
  data1Samples=`docker exec -it dataowner1.node.com sh -c "
    ./xdata-cli files upload --host http://dataowner1.node.com:80  -e '2022-06-21 15:59:05' -n $namespaces -m $fileAName -k $executor1PrivateKey \
    --ext '{\"FileType\":\"csv\",\"Features\":\"id,Sepal Length,Sepal Width\", \"TotalRows\":135}' -i /home/mpc-data/logic_iris_plants/$sampleFileAName
  "`
  echo "======> DataOwner1 upload vertical_logistic_train sample file: $data1Samples"

  data1FileUploadRes=$(echo $data1Samples | sed -e 's/\r//g' | sed 's/ //g')
  data1FileId=${data1FileUploadRes##*:}

  # DataOwner2 uploads logistic train sample files
  # 数据持有节点2上传纵向逻辑训练样本文件
  data2Samples=`docker exec -it dataowner2.node.com sh -c "
    ./xdata-cli files upload --host http://dataowner2.node.com:80  -e '2022-06-21 15:59:05' -n $namespaces -m $fileBName -k $executor2PrivateKey \
    --ext '{\"FileType\":\"csv\",\"Features\":\"id,Petal Length,Petal Width,Label\", \"TotalRows\":135}' -i /home/mpc-data/logic_iris_plants/$sampleFileBName
  "`
  echo "======> DataOwner2 upload vertical_logistic_train sample file: $data2Samples"

  data2FileUploadRes=$(echo $data2Samples | sed -e 's/\r//g' | sed 's/ //g')
  data2FileId=${data2FileUploadRes##*:}

  files="$data1FileId,$data2FileId"

  printf "\033[0;32m%s\033[0m\n" "======> Vertical logistic train sample files: $files"
}

# uploadLogisticPredictSampleFile dataOwner1 and dataOwner2 upload vertical logistic predict sample files
# 数据持有节点1和数据持有节点2上传纵向逻辑预测样本文件
function uploadLogisticPredictSampleFile() {
  sampleFileAName=predict_dataA.csv
  sampleFileBName=predict_dataB.csv
  fileAName="logistic_"$sampleFileAName
  fileBName="logistic_"$sampleFileBName

  # DataOwner1 uploads logistic predict sample files
  # 数据持有节点1上传纵向逻辑预测样本文件
  data1Samples=`docker exec -it dataowner1.node.com sh -c "
    ./xdata-cli files upload --host http://dataowner1.node.com:80  -e '2022-06-21 15:59:05' -n $namespaces -m $fileAName -k $executor1PrivateKey \
    --ext '{\"FileType\":\"csv\",\"Features\":\"id,Petal Length,Petal Width\", \"TotalRows\":15}' -i /home/mpc-data/logic_iris_plants/$sampleFileAName
  "`
  echo "======> DataOwner1 upload vertical_logistic_predict sample file: $data1Samples"

  data1FileUploadRes=$(echo $data1Samples | sed -e 's/\r//g' | sed 's/ //g')
  data1FileId=${data1FileUploadRes##*:}

  # DataOwner2 uploads logistic predict sample files
  # 数据持有节点2上传纵向逻辑预测样本文件
  data2Samples=`docker exec -it dataowner2.node.com sh -c "
    ./xdata-cli files upload --host http://dataowner2.node.com:80  -e '2022-06-21 15:59:05' -n $namespaces -m $fileBName -k $executor2PrivateKey \
    --ext '{\"FileType\":\"csv\",\"Features\":\"id,Petal Length,Petal Width\", \"TotalRows\":15}' -i /home/mpc-data/logic_iris_plants/$sampleFileBName
  "`
  echo "======> DataOwner2 upload vertical_logistic_predict sample file: $data2Samples"

  data2FileUploadRes=$(echo $data2Samples | sed -e 's/\r//g' | sed 's/ //g')
  data2FileId=${data2FileUploadRes##*:}

  files="$data1FileId,$data2FileId"

  printf "\033[0;32m%s\033[0m\n" "======> Vertical logistic predict sample files: $files"
}

# taskConfirmAndStart used Executor1 and Executor2 confirm task, then requester start ready task
# 任务执行节点分别确认任务后，计算需求方启动任务
function taskConfirmAndStart() {
  sleep 4
  # Executor1 confirms the task published by the requester
  # 任务执行节点1确认任务
  executor1ConfirmResult=`docker exec -it executor1.node.com sh -c "
    ./executor-cli task --host executor1.node.com:80 confirm -k $executor1PrivateKey -i $1"`
  echo "======> executor1 confirm task result: $executor1ConfirmResult"
  sleep 4

  # Executor2 confirms the task published by the requester
  # 任务执行节点2确认任务
  executor2ConfirmResult=`docker exec -it executor2.node.com sh -c "
    ./executor-cli task --host executor2.node.com:80 confirm  -k $executor2PrivateKey -i $1
    "`
  echo "======> executor2 confirm task result: $executor1ConfirmResult"
  sleep 4

  # Requester starts the task when train or predict task is confirmed
  # 计算方需求方启动任务
  requesterStartResult=`docker exec -it executor1.node.com sh -c "
  ./requester-cli task start -k $requesterKey -c ./conf/config.toml -i $1
  "`
  echo "======> requester start task result: $executor1ConfirmResult"
}

function linearVlTrain() {
  # List of sample files involved in linear train
  # 纵向线性训练任务所需的样本文件
  vlLinTrainfiles=$2
  for ((i=1; i<=$taskNum; i++))
  do
  # Requester publish linear training task
  # 计算需求方发布纵向线性训练任务
  result=`docker exec -it executor1.node.com sh -c "./requester-cli task publish -p $psi -a $vlLinAlgo -f $vlLinTrainfiles \
  -l $vlLinLabel -k $requesterKey -t train -n $vlLinTaskTrainName -c $config --amplitude $amplitude" | sed -e 's/\r//g' | sed 's/ //g'`

  echo "======> Requester publish linear train task result: $result "
  taskid=${result##*:};

  taskConfirmAndStart $taskid;
  done
}

function linearVlPredict() {
  # List of sample files involved in linear prediction
  # 纵向线性预测所需的预测样本文件
  vlLinPredictfiles=$2
  # Training task model ID required for linear prediction
  # 纵向线性预测所需的模型ID
  linearModelTaskId=$3
  # Requester publish linear prediction task
  # 计算需求方发布纵向线性预测任务
  result=`docker exec -it executor1.node.com sh -c " 
    ./requester-cli task publish -p $psi -a $vlLinAlgo -f $vlLinPredictfiles -k $requesterKey -t predict -n $vlLinTaskPredictName -c $config -i $linearModelTaskId
    " | sed -e 's/\r//g' | sed 's/ //g'`
  echo "======> Requester publish linear predict task result: $result "
  taskid=${result##*:}

  taskConfirmAndStart $taskid

  sleep 30
  # Get linear prediction results
  # 获取线性预测任务的预测结果
  linearVlPredictRes=`docker exec -it executor1.node.com sh -c "
  ./requester-cli task result -k $requesterKey -o ./linear_output.csv \
  --conf ./conf/config.toml  -i $taskid
  "`
  echo "======> Requester get linear predict task result: $linearVlPredictRes "
  # Copy linear prediction results to the current directory
  # 将线性预测结果拷贝到当前目录
  docker cp executor1.node.com:/home/linear_output.csv ./
  echo "======> LinearVlPredict file path: ./linear_output.csv"
}

function logisticVlTrain() {
  # List of sample files involved in logistic train
  # 纵向逻辑训练任务所需的样本文件
  vlLogTrainfiles=$2
  for ((i=1; i<=$taskNum; i++))
  do
  # Requester publish logistic training task
  # 计算需求方发布纵向逻辑训练任务
  result=`docker exec -it executor1.node.com sh -c "./requester-cli task publish -p $psi -a $vlLogAlgo -f $vlLogTrainfiles \
  -l $vlLogLabel -k $requesterKey -t train -n $vlLogTaskTrainName -c $config --labelName $vlLogLabelName" | sed -e 's/\r//g' | sed 's/ //g'`

  echo "======> Requester publish logistic train task result: $result "
  taskid=${result##*:};

  taskConfirmAndStart $taskid;
  done
}

function logisticVlPredict() {
  # List of sample files involved in logistic prediction
  # 纵向逻辑预测所需的预测样本文件
  vlLogPredictfiles=$2
  # Training task model ID required for logistic prediction
  # 纵向逻辑预测所需的模型ID
  logisticModelTaskId=$3
  # Requester publish linear training task
  # 计算需求方发布纵向逻辑预测任务
  result=`docker exec -it executor1.node.com sh -c " 
    ./requester-cli task publish -p $psi -a $vlLogAlgo -f $vlLogPredictfiles -k $requesterKey -t predict -n $vlLogTaskPredictName -c $config -i $logisticModelTaskId
    " | sed -e 's/\r//g' | sed 's/ //g'`
  echo "======> Requester publish logistic predict task result: $result "
  taskid=${result##*:}

  taskConfirmAndStart $taskid

  sleep 10
  # Get logistic prediction results
  # 获取逻辑预测任务的预测结果
  logisticVlPredictRes=`docker exec -it executor1.node.com sh -c "
  ./requester-cli task result -k $requesterKey -o ./logistic_output.csv \
  --conf ./conf/config.toml  -i $taskid
  "`
  echo "======> Requester get logistic predict task result: $logisticVlPredictRes "
  # Copy logistic prediction results to the current directory
  # 将逻辑预测结果拷贝到当前目录
  docker cp executor1.node.com:/home/logistic_output.csv ./
  echo "======> LogisticVlPredict file path: ./logistic_output.csv"
}

function taskList() {
  docker exec -it executor1.node.com sh -c "
  ./requester-cli task list --conf ./conf/config.toml  -p $requesterPublicKey"
}

function getTaskById() {
  taskID=$2
  docker exec -it executor1.node.com sh -c " 
  ./executor-cli task getbyid --host 127.0.0.1:80 -i $taskID"
}

action=$1
case $action in
upload_sample_files)
  uploadSampleFiles $@
  ;;
start_vl_linear_train)
  linearVlTrain $@
  ;;
start_vl_linear_predict)
  linearVlPredict $@
  ;;
start_vl_logistic_train)
  logisticVlTrain $@
  ;;
start_vl_logistic_predict)
  logisticVlPredict $@
  ;;
tasklist)
  taskList $@
  ;;
gettaskbyid)
  getTaskById $@
  ;;
*)
  echo "Usage: $0 {upload_sample_files | start_vl_linear_train | start_vl_linear_predict \
  | start_vl_logistic_train | start_vl_logistic_predict | tasklist | gettaskbyid}"
  exit 1
  ;;
esac



