# XuperDB 服务启动和命令使用说明

## 一、服务启动和停止 [./network_up.sh]
Usage: ./network_up.sh { start | stop | restart }

### 1.1 服务启动
- blockchainType, 网络类型, 取值xchain或fabric，默认xchain网络
```shell
./network_up.sh start $blockchainType
```

### 1.2 服务停止
```shell
./network_up.sh stop $blockchainType
```

### 1.3 服务重启
```shell
./network_up.sh restart $blockchainType
```

## 二、文件上传和下载 [./xdb_test.sh]
Usage: ./xdb_test.sh {create_namespace | upload_file | download_file | filelist | getfilebyid

### 2.1 命名空间创建
- nameSpace, 文件存储所需的命名空间
```shell
./xdb_test.sh create_namespace -n $nameSpace
```

### 2.2 文件上传
- nameSpace, 取值2.1步骤创建的nameSpace
- filePath, 所需上传的文件路径
- fileDescription, 文件的描述信息
```shell
./xdb_test.sh upload_file -n $nameSpace -i $filePath -d $fileDescription
```

### 2.3、文件下载
- fileID，文件ID
```shell
./xdb_test.sh download_file -f $fileID
```

## 三、文件列表查询
### 3.1 脚本方式
- nameSpace, 文件所在的命名空间
```shell
./xdb_test.sh filelist -n $nameSpace
```

### 3.2 命令行方式
- nameSpace, 文件所在的命名空间
- dataOwnerPublicKey, 数据拥有者的公钥
```shell
docker exec -it dataowner.node.com sh -c "
  ./xdb-cli files list --host http://dataowner.node.com:80 -n $nameSpace -o  $dataOwnerPublicKey"
```

### 四、单个文件查询
### 4.1 脚本方式
- fileID，文件ID
```shell
./xdb_test.sh getfilebyid -f $fileID
```

### 4.2 命令行方式
- fileID，文件ID
```shell
docker exec -it dataowner.node.com sh -c "./xdb-cli files getbyid --host http://dataowner.node.com:80 -i $fileID"
```
