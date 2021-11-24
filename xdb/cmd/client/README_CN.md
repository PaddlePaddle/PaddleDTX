[English](./README.md) | 中文

# XuperDB 命令行工具

## 一、存储节点操作

### 存储节点操作命令说明 [./bin/client nodes]
| command    |        说明      |
| ---------- |   -----------   |
| add        | add node into xuper db  |
| genkey     | generate a pair of node key |
| genpdpkeys | generate pdp keys |
| get        | get node of xuper db by id  |
| health     | get node health status by id  |
| list       | list nodes of xuper db  |
| mrecords   | get node slice migrate records  |
| heartbeat  | heartbeatnum of given day, example '2021-07-10 12:00:00' |   
| offline    | get node of xuper db offline by privatekey |
| online     | get node of xuper db online by privatekey |    

### 为节点生成公私钥
```shell
$ ./bin/client nodes genkey
```

### 为租赁节点生成基于双线性对挑战的公私钥
```shell
$ ./bin/client nodes genpdpkeys
```

### 获取节点列表
```shell
$ ./bin/client --host http://localhost:8001 nodes list
```

### 获取节点信息
```shell
$ ./bin/client --host http://localhost:8001 nodes get --id 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

### 节点上线
```shell
$ ./bin/client --host http://localhost:8001 nodes online -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79
```

### 节点下线
```shell
$ ./bin/client --host http://localhost:8001 nodes offline -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79
```

### 节点健康度查询
```shell
$ ./bin/client --host http://localhost:8001 nodes health --id 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

### 节点迁移记录查询
```shell
$ ./bin/client --host http://localhost:8001 nodes mrecords --id 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

### 节点心跳检测数量
```shell
$./bin/client --host http://localhost:8001 nodes heartbeat --id 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -c "2021-08-04 17:29:00"
```

## 二、文件操作

### 文件操作命令说明 [./bin/client files]：
| command    |        说明      |
| ---------- |   -----------   |
| addns      | add file namespace into xuper db  |
| download   | download file from xuper db  |
| getbyid    | get file of xuper db by id  |
| getbyname  | get file of xuper db by name  |
| getns      | get file namespace detail of xuper db  |
| list       | list files of xuper db |
| listexp    | list expired but valid files of xuper db |
| listns     | list file namespaces of owner |
| syshealth  | get files sys health of xuper db  |
| upload     | upload file into xuper db  |
| ureplica   | update file replica of xuper db |
| utime      | update file expiretime by id  |
 

### 文件命名空间新增
```shell
$ ./bin/client --host http://localhost:8001 files addns -n py1  -r 1 -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79 
```

### 命名空间详情查询
```shell
$ ./bin/client --host http://localhost:8001 files getns -n py1  -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 
```

### 查看命名空间列表
```shell
$ ./bin/client --host http://localhost:8001 files listns -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 
```

### 修改命名空间副本数
```shell
$ ./bin/client --host http://localhost:8001 files ureplica -n py  -r 3 -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79
```

### 文件上传
```shell
$ ./bin/client --host http://localhost:8001 files upload -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79 -n test -m bigfile -i ./bin/client -e "2021-06-30 15:00:00" -d "this is a test file"
```

### 文件续期
```shell
$ ./bin/client --host http://localhost:8001 files utime -e '2021-08-08 15:15:04' -i b87b588f-2e46-4ee5-8128-888592ada4fd -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79
```

### 文件下载：依据文件名下载
```shell
$ ./bin/client --host http://localhost:8001 files download -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79 -n test -m bigfile -o ./testdata/bigfile
```

### 文件下载：依据文件ID下载
```shell
$ ./bin/client --host http://localhost:8001 files download -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79 --fileid 2695c0c5-5a30-4184-bf15-94df43857070 -o ./testdata/bigfile
```

### 查看文件列表
```shell
$ ./bin/client --host http://localhost:8001 files list -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n test -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00" --ext "{'FileType':'csv','Features':'room,price', 'TotalRows':400}"
```

### 查看过期(仍在保留期内)文件列表
```shell
$ ./bin/client --host http://localhost:8001 files listexp -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n test -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### 查看文件信息：依据文件ID
```shell
$ ./bin/client --host http://localhost:8001 files getbyid --id d86737bf-97ac-427f-a835-871d307c3589
```

### 查看文件信息：依据文件名称
```shell
$ ./bin/client --host http://localhost:8001 files getbyname -n test -m bigfile -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

### 查看文件系统健康度
```shell
$ ./bin/client --host http://localhost:8001 files syshealth -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

## 三、挑战操作

### 挑战操作命令说明[./bin/client challenge]
| command    |        说明      |
| ---------- |   -----------   |
| failed     | get failed challenges by filters  |
| get        | get pdp challenge by id   |
| proved     | get proved challenges by filters  |        
| toprove    | get ToProve challenges by filters  |        

### 获取挑战
```shell
$ ./bin/client --host http://localhost:8001 challenge get --id ff334622-1442-425c-8d57-3f1b915b84e8 
```

### 获取成功的挑战列表
```shell
$ ./bin/client --host http://localhost:8001 challenge proved -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### 获取失败的挑战列表
```shell
$ ./bin/client --host http://localhost:8001 challenge failed -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### 获取待挑战的列表
```shell
$ ./bin/client --host http://localhost:8001 challenge toprove -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00" --list 0
```
