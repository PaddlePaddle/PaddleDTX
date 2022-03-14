[English](./README.md) | 中文

# XuperDB 命令行工具

## 一、节点操作

### 节点公私钥对生成命令说明 [./xdb-cli key]
| command    |        explanation      | 
| :----------: |   :-----------:   | 
| genkey       | generate a pair of key |  
| addukey      | used for the dataOwner node to add client's public key into the whitelist | 
| genpdpkeys   | generate pairing based challenge parameters |

### 为数据持有节点或存储节点生成公私钥
```shell
$ ./xdb-cli key genkey -o ./keys
```

### 为数据持有节点客户端生成公私钥
```shell
$ ./xdb-cli key genkey -o ./ukeys
```

### 将数据持有节点的客户端公钥加入白名单
```shell
$ ./xdb-cli key addukey -o ./authkeys -u 339524f35fb86a85bc3f9eed2b6ffd976de08b2cd47953b6640912f16e6863f2123f057cfef1f7132072602255a5a39bf254569fa6f8591327255c97881bc112
```

### 为数据持有节点生成基于双线性对挑战的公私钥
```shell
$ ./xdb-cli nodes genpdpkeys
```

## 二、存储节点操作

### 存储节点操作命令说明 [./xdb-cli nodes]
| command    |        explanation      |
| ---------- |   -----------   |
| get        | get the storage node by id  |
| health     | get the storage node's health status by id  |
| list       | list storage nodes |
| mrecords   | get node slice migrate records  |
| heartbeat  | get storage node heart beat number of one day, example '2021-07-10 12:00:00' |   
| offline    | set a storage node offline |
| online     | set a storage node online |   

### 获取节点列表
```shell
$ ./xdb-cli nodes list --host http://localhost:8122
```

### 获取节点信息
```shell
$ ./xdb-cli nodes get --host http://localhost:8122 --keyPath ./keys
```

### 节点上线
```shell
$ ./xdb-cli --host http://localhost:8122 nodes online --keyPath ./keys
```

### 节点下线
```shell
$ ./xdb-cli --host http://localhost:8122 nodes offline --keyPath ./keys
```

### 节点健康度查询
```shell
$ ./xdb-cli nodes health --host http://localhost:8122 --keyPath ./keys
```

### 节点迁移记录查询
```shell
$ ./xdb-cli nodes mrecords --host http://localhost:8122 --keyPath ./keys
```

### 节点心跳检测数量
```shell
$ ./xdb-cli nodes heartbeat --host http://localhost:8122 --keyPath ./keys -c "2021-08-04 17:29:00"
```

## 三、文件操作

### 文件操作命令说明 [./bin/xdb-cli files]：
| command     |        explanation      |
| ----------  |   -----------   |
| addns       | add a file namespace into XuperDB  |
| download    | download the file from XuperDB  |
| getbyid     | get the file by id from XuperDB  |
| getbyname   | get the file by name from XuperDB |
| getns       | get the file namespace detail in XuperDB  |
| list        | list files in XuperDB |
| listexp     | list expired but valid files in XuperDB |
| listns      | list file namespaces of the DataOwner |
| syshealth   | get the DataOwner's health status  |
| upload      | save a file into XuperDB |
| ureplica    | update file replica of XuperDB |
| utime       | update file's expiretime by the id |  
| getauthbyid | get the file authorization application detail | 
| confirmauth | confirm the applier's file authorization application | 
| rejectauth  | reject the applier's file authorization application |
| listauth    | list file authorization applications | 
 

### 文件命名空间新增
```shell
$ ./xdb-cli --host http://localhost:8121 files addns -n testns  -r 2 --keyPath ./ukeys
```

### 命名空间详情查询
```shell
$ ./xdb-cli --host http://localhost:8121 files getns -n testns
```

### 查看命名空间列表
```shell
$ ./xdb-cli --host http://localhost:8121 files listns -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### 修改命名空间副本数
```shell
$ ./xdb-cli --host http://localhost:8121 files ureplica -n testns  -r 3 --keyPath ./ukeys
```

### 文件上传
```shell
$ ./xdb-cli --host http://localhost:8121 files upload --keyPath ./ukeys -n testns -m bigfile -i ./bin/client -e "2021-06-30 15:00:00" -d "this is a test file"
```

### 文件续期
```shell
$ ./xdb-cli --host http://localhost:8121 files utime -e '2021-08-08 15:15:04' -i b87b588f-2e46-4ee5-8128-888592ada4fd --keyPath ./ukeys
```

### 文件下载：依据文件名下载
```shell
$ ./bin/xdb-cli --host http://localhost:8001 files download --keyPath ./ukeys -n testns -m bigfile -o ./testdata/bigfile
```

### 文件下载：依据文件ID下载
```shell
$ ./bin/xdb-cli --host http://localhost:8001 files download --keyPath ./ukeys --fileid 2695c0c5-5a30-4184-bf15-94df43857070 -o ./testdata/bigfile
```

### 查看文件列表
```shell
$ ./xdb-cli --host http://localhost:8121 files list -n testns -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### 查看过期(仍在保留期内)文件列表
```shell
$ ./xdb-cli --host http://localhost:8121 files listexp -n testns -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### 查看文件信息：依据文件ID
```shell
$ ./xdb-cli --host http://localhost:8121 files getbyid --id d86737bf-97ac-427f-a835-871d307c3589
```

### 查看文件信息：依据文件名称
```shell
$ ./xdb-cli --host http://localhost:8121 files getbyname -n testns -m bigfile
```

### 查看文件系统健康度
```shell
$ ./xdb-cli --host http://localhost:8121 files syshealth
```

### 查看文件授权申请列表
```shell
$ ./xdb-cli --host http://localhost:8121 files listauth -s '2022-01-08 15:15:04'
```

### 查看文件授权申请详情
```shell
$ ./xdb-cli --host http://localhost:8121 files getauthbyid  --id 933b347a-a207-46ed-bcd7-8fdde94596d0
```

### 确认文件授权申请
```shell
$ ./xdb-cli --host http://localhost:8121 files confirmauth -e '2022-08-08 15:15:04' -i b87b588f-2e46-4ee5-8128-888592ada4fd --keyPath ./ukeys
```

### 拒绝文件授权申请
```shell
$ ./xdb-cli --host http://localhost:8121 files rejectauth -r '拒绝授权申请' -i b87b588f-2e46-4ee5-8128-888592ada4fd --keyPath ./ukeys
```

## 四、挑战操作

### 挑战操作命令说明[./bin/xdb-cli challenge]
| command    |        说明      |
| ---------- |   -----------   |
| failed     | get failed challenges by filters  |
| get        | get pdp challenge by id   |
| proved     | get proved challenges by filters  |        
| toprove    | get ToProve challenges by filters  |        

### 获取挑战
```shell
$ ./xdb-cli --host http://localhost:8121 challenge get --id ff334622-1442-425c-8d57-3f1b915b84e8 
```

### 获取成功的挑战列表
```shell
$ ./xdb-cli --host http://localhost:8121 challenge proved -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### 获取失败的挑战列表
```shell
$ ./xdb-cli --host http://localhost:8121 challenge failed -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### 获取待挑战的列表
```shell
$ ./xdb-cli --host http://localhost:8121 challenge toprove -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```
