# Command-line Tool: xdb-cli 
The `xdb-cli` is a command-line tool to use the decentralized storage network by a DataOwner node or a Storage node.
There are four major subcommands of `xdb-cli` as follows.

| command    |        explanation      | 
| :----------: |   :-----------:   |
| key      |   generate node private/public key pair |
| nodes    |  control actions related to storage nodes | 
| files    | file operations on the decentralized storage network | 
| challenge    | challenge operations used to check file integrity in the storage node |  

## Command Parsing:  `xdb-cli key`
The subcommand `xdb-cli key` used to generate the node private/public key pair or add client's  public key into the whitelist.

| command    |        explanation      | 
| :----------: |   :-----------:   | 
| genkey       | generate a pair of key |  
| addukey      | used for the dataOwner node to add client's public key into the whitelist | 
| genpdpkeys   | generate pairing based challenge parameters |

### genkey
`xdb-cli key genkey` used for node or node's client to generate a pair of key

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --output  |      -o    |   output path |    yes    |

```
DEMO:
$  ./xdb-cli key genkey -o ./keys
$  ./xdb-cli key genkey -o ./ukeys
```

### addukey

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --output  |      -o    |   output path |    yes    |
|   --user    |      -u    |   user public key |    yes    |

```
DEMO:
$  ./xdb-cli key addukey -o ./authkeys -u 339524f35fb86a85bc3f9eed2b6ffd976de08b2cd47953b6640912f16e6863f2123f057cfef1f7132072602255a5a39bf254569fa6f8591327255c97881bc112
```

### genpdpkeys

```
DEMO:
$  ./xdb-cli key genpdpkeys
```

## Command Parsing: `xdb-cli nodes`

| command    |        explanation      |
| ---------- |   -----------   |
| get        | get the storage node by id  |
| health     | get the storage node's health status by id  |
| list       | list storage nodes |
| mrecords   | get node slice migrate records  |
| heartbeat  | get storage node heart beat number of one day, example '2021-07-10 12:00:00' |   
| offline    | set a storage node offline |
| online     | set a storage node online |   

| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --host |      -h    |   the storage node's host | yes |

### get

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i   | storage node's id |    no, you can replace 'id' with 'keyPath'    |
|   --keyPath  |         |  the file path of the stroaga node's public key |    no, default './keys'    |

```
DEMO:
$ ./xdb-cli nodes get --host http://localhost:8122 --keyPath ./keys
```

### health

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i   |  storage node's id |    no, you can replace 'id' with 'keyPath'    |
|   --keyPath  |         |  the file path of the stroaga node's public key |    no, default './keys'    |
```
DEMO:
$ ./xdb-cli nodes health --host http://localhost:8122 --keyPath ./keys
```

### list

```
DEMO:
$ ./xdb-cli nodes list --host http://localhost:8122
```

### mrecords

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i   |   storage node's id |    no, you can replace 'id' with 'keyPath'    |
|   --keyPath  |         |  the file path of the stroaga node's public key |    no, default './keys'    |
|   --limit  |  -l   |   limit for list slice migrate records|    no    |
|   --start  |      -s   |   start time of the slice migrate' query |    no    |
|   --end  |      -e   |   end time of the slice migrate' query |    no    |

```
DEMO:
$ ./xdb-cli nodes mrecords --host http://localhost:8122 --keyPath ./keys
```

### heartbeat

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |    -i   | storage node's id |    no, you can replace 'id' with 'keyPath'    |
|   --keyPath  |         |  the file path of the stroaga node's public key |    no, default './keys'    |
|   --ctime  |   -c |  storage node heart beat number of one day |    yes    |


```
DEMO:
$ ./xdb-cli nodes heartbeat --host http://localhost:8122 --keyPath ./keys -c "2021-08-04 17:29:00"
```

### offline

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --privateKey  |      -k    |   private key |    no, you can replace 'privateKey' with 'keyPath'    |
|   --keyPath  |         |  the file path of the storage node's private key |    no, default './keys'    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8122 nodes offline --keyPath ./keys
```

### online

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --privateKey  |      -k    |   private key |    no, you can replace 'privateKey' with 'keyPath'    |
|   --keyPath  |         |  the file path of the storage node's private key |    no, default './keys'    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8122 nodes online --keyPath ./keys
```

## Command Parsing: `xdb-cli files`

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

| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --host |      -h    |   the dataOwner node's host | yes |


### addns

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --privateKey  |      -k    |   private key |    no, you can replace 'privateKey' with 'keyPath'    |
|   --keyPath  |         |  the file path of the dataOwner node client's private key |    no, default './ukeys'    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --description  |      -d    |   description |    no    |
|   --replica  |      -r    |   replica |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files addns -n testns  -r 2 --keyPath ./ukeys
```

### download

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --fileid  |      -f    |  file's id in XuperDB |   no    |
|   --filename  |      -m    |  file's name in XuperDB |    no    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --output  |      -o    |   output file path |    yes    |
|   --privkey  |      -k    |   private key |    no, you can replace 'privkey' with 'keyPath'    |
|   --keyPath  |         |  the file path of the dataOwner node client's private key |    no, default './ukeys'    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files download --keyPath ./ukeys -n testns -m bigfile -o ./testdata/bigfile 
```

### getbyid

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i   |  file's id in XuperDB |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files getbyid --id d86737bf-97ac-427f-a835-871d307c3589
```

### getbyname
|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --filename  |      -m    |  file's name |    yes    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    no, default host node's public key    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files getbyname -n testns -m bigfile
```

### getns

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --namespace  |      -n    |   namespace |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    no, default host node's public key   |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files getns -n testns  
```

### list

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --owner  |      -o    |  DataOwner's public key |   no, default host node's public key   |
|   --namespace  |      -n    |   namespace |    yes    |
|   --limit  |  -l   |   limit for list, 0 for unlimited|    no    |
|   --start  |      -s   |   start time of the slice migrate' query |    no    |
|   --end  |      -e   |   end time of the slice migrate' query |    no    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files list -n testns -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```


### listexp

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --owner  |      -o    |  DataOwner's public key |   no, default host node's public key    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --limit  |  -l   |   limit for list, 0 for unlimited|    no    |
|   --start  |      -s   |   start time of the slice migrate' query |    no    |
|   --end  |      -e   |   end time of the slice migrate' query |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files listexp -n testns -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### listns

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --owner  |      -o    |  DataOwner's public key |   no, default host node's public key   |
|   --limit  |  -l   |   limit for list, 0 for unlimited|    no    |
|   --start  |      -s   |   start time of the slice migrate' query |    no    |
|   --end  |      -e   |   end time of the slice migrate' query |    no    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files listns -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### syshealth

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --owner  |      -o    |  DataOwner's public key |    no, default host node's public key    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files syshealth
```
### upload

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --description  |      -d    |   description |    yes    |
|   --privkey  |      -k    |   private key |   no, you can replace 'privkey' with 'keyPath'    |
|   --keyPath  |        |  the file path of the dataOwner node client's private key |    no, default './ukeys'    |
|   --expireTime  |      -e    |   expiretime of the file in XuperDB |    yes    |
|   --ext  |        |   file extra info |    yes    |
|   --filename  |      -m    |  file's name in XuperDB |    yes    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --input  |      -i    |  input file path |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files upload --keyPath ./ukeys -n testns -m bigfile -i ./bin/client -e "2021-06-30 15:00:00" -d "this is a test file"
```

### ureplica

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --namespace  |      -n    |   namespace |    yes    |
|   --privkey  |      -k    |   private key |   no, you can replace 'privkey' with 'keyPath'    |
|   --keyPath  |        |  the file path of the dataOwner node client's private key |    no, default './ukeys'    |
|   --replica  |      -r    |   replica |    yes    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files ureplica -n testns  -r 3 --keyPath ./ukeys
```

### utime

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |  file's id in XuperDB |   yes    |
|   --privkey  |      -k    |   private key |    no, you can replace 'privkey' with 'keyPath'    |
|   --keyPath  |        |  the file path of the dataOwner node client's private key |    no, default './ukeys'    |
|   --expireTime  |      -e    |   expiretime of the file in XuperDB |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files utime -e '2021-08-08 15:15:04' -i b87b588f-2e46-4ee5-8128-888592ada4fd --keyPath ./ukeys
```

### getauthbyid

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --authID  |      -i    |  id of file authorization application |   yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files getauthbyid  --id 933b347a-a207-46ed-bcd7-8fdde94596d0
```

### confirmauth

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --authID   |      -i    |  id for file authorization application |   yes    |
|   --privkey  |      -k    |   private key |    no, you can replace 'privkey' with 'keyPath'    |
|   --keyPath  |            |  the file path of the dataOwner node client's private key |    no, default './ukeys'    |
|   --expireTime  |      -e    |  file authorization expiration time, example '2022-07-10 12:00:00' |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files confirmauth -e '2022-08-08 15:15:04' -i b87b588f-2e46-4ee5-8128-888592ada4fd --keyPath ./ukeys
```

### rejectauth

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --authID   |      -i    |  id for file authorization application |   yes    |
|   --privkey  |      -k    |   private key |    no, you can replace 'privkey' with 'keyPath'    |
|   --keyPath  |            |  the file path of the dataOwner node client's private key |    no, default './ukeys'    |
|   --rejectReason  |      -e    |  reason for reject the authorization |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files rejectauth -r '拒绝授权申请' -i b87b588f-2e46-4ee5-8128-888592ada4fd --keyPath ./ukeys
```

### listauth

|     flag    |  short flag   | explanation | necessary |
| :---------: | :-----------: | :------------: | :---------: |
|   --applier |      -a       |   applier's public key |    no   |
|   --owner   |      -o       |   file owner |    no, default host node's public key    |
|   --end     |      -e       |   authorization applications publish before endTime, example '2022-07-10 12:00:00' |    no    |
|   --fileID  |      -f       |   sample file ID |    no    |
|   --start   |      -s       |   authorization applications publish after startTime, example '2022-06-10 12:00:00' |    no    |
|   --limit   |      -l       |   slimit for list file authorization applications |    no    |
|   --status  |               |   status of file authorization application, example 'Unapproved, Approved or Rejected' |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8121 files listauth -s '2022-01-08 15:15:04'
```

## Command Parsing: `xdb-cli challenge`

| command    |        explanation      |
| ---------- |   -----------   |
| failed     | get failed challenges by filters  |
| get        | get pdp challenge by id   |
| proved     | get proved challenges by filters  |        
| toprove    | get ToProve challenges by filters  |        

| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --host |      -h    |   the dataOwner node's host | yes |


### get
|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |  challenge's id |   no    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8121 challenge get --id ff334622-1442-425c-8d57-3f1b915b84e8 
```

### proved

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --file  |      -f    |  file's id in XuperDB |   no    |
|   --limit  |  -l   |   limit for list num|    no    |
|   --list  |   |  whether or not show challenges list, 0:not show |  default 1  |
|   --node  |  -n  |  storage node's id |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    no, default host node's public key    |
|   --start  |      -s   |   start time of the query |    no    |
|   --end  |      -e   |   end time of the query |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8121 challenge proved -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### failed

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --file  |      -f    |  file's id in XuperDB |   no    |
|   --limit  |  -l   |   limit for list num|    no    |
|   --list  |   |  whether or not show challenges list, 0:not show |  default 1  |
|   --node  |  -n  |  storage node's id |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    no, default host node's public key   |
|   --start  |      -s   |   start time of the query |    no    |
|   --end  |      -e   |   end time of the query |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8121 challenge failed -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### toprove

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --file  |      -f    |  file's id in XuperDB |   no    |
|   --limit  |  -l   |   limit for list num|    no    |
|   --list  |   |  whether or not show challenges list, 0:not show |  default 1  |
|   --node  |  -n  |  storage node's id |    yes    |
|   --owner  |      -o    |  DataOwner's public key |   no, default host node's public key   |
|   --start  |      -s   |   start time of the query |    no    |
|   --end  |      -e   |   end time of the query |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8121 challenge toprove -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```
