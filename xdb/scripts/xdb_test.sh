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

# Script to test xdb service, create namespace and upload file with docker-compose.
# Usage: ./xdb_test.sh {create_namespace | upload_file | download_file | filelist | getfilebyid

# Expiration time of file storage
# 文件存储的到期时间
ARCH=$(uname -s | grep Darwin)
if [ "$ARCH" == "Darwin" ]; then
  FILE_EXPIRETIME=`date -v +6m +"%Y-%m-%d %H:%M:%S"`
else
  FILE_EXPIRETIME=`date -d "+6 month" +"%Y-%m-%d %H:%M:%S"`
fi

# createNamespace create namespace for dataOwner
# 数据持有节点创建命名空间
function createNamespace() {
  paramCheck "$NAMESPACE" "Namespace of file cannot be empty"
  dataownerAddnsResult=`docker exec -it dataowner.node.com sh -c "
    ./xdb-cli files addns --keyPath ./ukeys --host http://dataowner.node.com:80 -n $NAMESPACE -r 2"`
  echo "======> DataOwner create files storage namespace of $NAMESPACE: $dataownerAddnsResult"
  isAddNsRes=$(echo $dataownerAddnsResult | awk 'BEGIN{RS="\r";ORS="";}{print $0}' | awk '$1=$1')
  if [ "$isAddNsRes" != "OK" ]; then
    exit 1
  fi
}

# uploadFile dataOwner uploads file into xdb network
# 数据持有节点上传文件到去中心化存储网络
function uploadFile() {
  paramCheck "$NAMESPACE" "Namespace of file cannot be empty"
  paramCheck "$FILE_PATH" "FilePath cannot be empty"
  paramCheck "$FILE_DES" "File description cannot be empty"

  fileName=`echo ${FILE_PATH##*/}`
  docker exec -it dataowner.node.com sh -c "rm -rf uploadFiles && mkdir uploadFiles"
  # Copy file to the dataowner container
  # 将上传的文件拷贝到dataowner.node.com容器
  docker cp $FILE_PATH dataowner.node.com:/home/uploadFiles/$fileName
  uploadResult=`docker exec -it dataowner.node.com sh -c "
    ./xdb-cli files upload --keyPath ./ukeys --host http://dataowner.node.com:80  -e '$FILE_EXPIRETIME' -n $NAMESPACE -m $fileName \
    -i /home/uploadFiles/$fileName -d '$FILE_DES'"`
  echo "======> DataOwner upload file: $uploadResult"
}

# downloadFile dataOwner downloads file from xdb network
# 数据持有节点从去中心化存储网络下载文件
function downloadFile() {
  paramCheck "$FILEID" "FileId cannot be empty"
  # Get file name by fileID
  # 通过文件ID获取文件名称
  fileName=`docker exec -it dataowner.node.com sh -c " 
  ./xdb-cli files getbyid --host http://dataowner.node.com:80 \
  -i $FILEID" | grep "FileName" | awk -F": " '{print $2}' | awk 'BEGIN{RS="\r";ORS="";}{print $0}' | awk '$1=$1'`
  if [ "$fileName" = "" ];then
    paramCheck "$fileName" "FileId is invalid"
  fi

  docker exec -it dataowner.node.com sh -c "rm -rf downloadFiles && mkdir downloadFiles"
  downloadResult=`docker exec -it dataowner.node.com sh -c "
  ./xdb-cli files download --keyPath ./ukeys --host http://dataowner.node.com:80 -f $FILEID -o /home/downloadFiles/$fileName"`
  # Copy file to the current directory
  # 将下载的文件拷贝到当前目录
  docker cp dataowner.node.com:/home/downloadFiles/$fileName ./
  echo "======> DataOwner download file path: ./$fileName"
}

# fileList get file list with specified namespace from xdb network
# 获取指定命名空间的文件列表
function fileList() {
  paramCheck "$NAMESPACE" "Namespace of file cannot be empty"
  docker exec -it dataowner.node.com sh -c "
  ./xdb-cli files list --host http://dataowner.node.com:80 -n $NAMESPACE"
}

# getFileByID get file info by file id from xdb network
# 根据文件ID获取文件详情
function getFileByID() {
  paramCheck "$FILEID" "FileId cannot be empty"
  docker exec -it dataowner.node.com sh -c " 
  ./xdb-cli files getbyid --host http://dataowner.node.com:80 -i $FILEID"
}

function paramCheck() {
  if [ "$1" = "" ];then
    printf "\033[0;31m======> ERROR !!!! %s\033[0m\n" "$2"
    exit 1
  fi
}

# Print the usage message
function printHelp() {
  echo "Usage: "
  echo "  ./xdb_test.sh <mode> [-n <file namespace>] [-i <file path>] [-d <file description>] [-d <file id>]"
  echo "    <mode> - one of 'create_namespace', 'upload_file', 'download_file', 'filelist' or 'getfilebyid'"
  echo "      - 'create_namespace' - add a file namespace into XuperDB"
  echo "      - 'upload_file' - save a file into XuperDB"
  echo "      - 'download_file' - download the file from XuperDB"
  echo "      - 'filelist' - list files in XuperDB"
  echo "      - 'getfilebyid' - get the file by id from XuperDB"
  echo "    -n <file namespace> - namespace for file"
  echo "    -i <file path> - input file path"
  echo "    -d <file description> - file description"
  echo "    -f <file id> - file id"
  echo
  echo "  ./xdb_test.sh -h (print this message), e.g.:"
  echo
  echo "  ./xdb_test.sh create_namespace -n mynamespace"
  echo "  ./xdb_test.sh upload_file -n mynamespace -i ./README.md -d 'readme of xdb test'"
  echo "  ./xdb_test.sh filelist -n mynamespace"
  echo "  ./xdb_test.sh getfilebyid -f bfba4b3a-5b60-43e2-af79-b3be7ae6ef3b"
  echo "  ./xdb_test.sh download_file -f bfba4b3a-5b60-43e2-af79-b3be7ae6ef3b"
  echo
}

NAMESPACE=""
FILE_PATH=""
FILE_DES=""
FILEID=""

action=$1
shift
while getopts "h?n:i:d:f:" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  n)
    NAMESPACE=$OPTARG
    ;;
  i)
    FILE_PATH=$OPTARG
    ;;
  d)
    FILE_DES=$OPTARG
    ;;
  f)
    FILEID=$OPTARG
    ;;
  esac
done

case $action in
create_namespace)
  createNamespace $@
  ;;
upload_file)
  uploadFile $@
  ;;
download_file)
  downloadFile $@
  ;;
filelist)
  fileList $@
  ;;
getfilebyid)
  getFileByID $@
  ;;
*)
  printHelp
  exit 1
  ;;
esac
