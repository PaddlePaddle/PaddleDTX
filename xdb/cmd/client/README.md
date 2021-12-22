# Command-line Tool: xdb-cli 
The `xdb-cli` is a command-line tool to use the decentralized storage network by a DataOwner node.
There are three major subcommands of `xdb-cli` as follows.

| command    |        explanation      | 
| :----------: |   :-----------:   | 
| nodes    |  control actions related to storage nodes | 
| files    | file operations on the decentralized storage network | 
| challenge    | challenge operations used to check file integrity in the storage node |  


## Command Parsing: `xdb-cli nodes`

| command    |        explanation      |
| ---------- |   -----------   |
| add        | add a storage node into XuperDB  |
| genkey     | generate a pair of key |
| genpdpkeys | generate pdp keys |
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

### add

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --address  |      -a   |   network address can be accessed |    yes    |
|   --name  |      -n    |   node's name |    yes    |
|   --privateKey  |      -k    |   private key |    yes    |
|   --description  |      -d    |   description |    yes    |

```
DEMO:
$ ./xdb-cli nodes add -a http://127.0.0.2:8123 -n storage1 -k 0e632dfe60f6a70ae5230e963780c581499beccf6d04133c2dd1e59e27cb6404 -d 'a storage node'  --host http://localhost:8122
```

### genkey

```
DEMO:
$ ./xdb-cli nodes genkey
```

### genpdpkeys

```
DEMO:
$ ./xdb-cli nodes genpdpkeys
```

### get

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i   | storage node's id |    yes    |

```
DEMO:
$ ./xdb-cli nodes get --host http://localhost:8122 -i 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

### health

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i   |  storage node's id |    yes    |

```
DEMO:
$ ./xdb-cli nodes health --host http://localhost:8122 -i 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

### list

```
DEMO:
$ ./xdb-cli nodes list --host http://localhost:8122
```

### mrecords

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i   |   storage node's id |    yes    |
|   --limit  |  -l   |   limit for list slice migrate records|    no    |
|   --start  |      -s   |   start time of the slice migrate' query |    no    |
|   --end  |      -e   |   end time of the slice migrate' query |    no    |

```
DEMO:
$ ./xdb-cli nodes mrecords --host http://localhost:8122 --id 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

### heartbeat

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |    -i   | storage node's id |    yes    |
|   --ctime  |   -c |  storage node heart beat number of one day |    yes    |


```
DEMO:
$ ./xdb-cli nodes heartbeat --host http://localhost:8122 --id 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -c "2021-08-04 17:29:00"
```

### offline

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --privateKey  |      -k    |   private key |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 nodes offline -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79
```

### online

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --privateKey  |      -k    |   private key |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 nodes online -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79
```

## Command Parsing: `xdb-cli files`

| command    |        explanation      |
| ---------- |   -----------   |
| addns      | add a file namespace into XuperDB  |
| download   | download the file from XuperDB  |
| getbyid    | get the file by id from XuperDB  |
| getbyname  | get the file by name from XuperDB |
| getns      | get the file namespace detail in XuperDB  |
| list       | list files in XuperDB |
| listexp    | list expired but valid files in XuperDB |
| listns     | list file namespaces of the DataOwner |
| syshealth  | get the DataOwner's health status  |
| upload     | save a file into XuperDB |
| ureplica   | update file replica of XuperDB |
| utime      | update file's expiretime by the id |  

| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --host |      -h    |   the storage node's host | yes |


### addns

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --privateKey  |      -k    |   private key |    yes    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --description  |      -d    |   description |    no    |
|   --replica  |      -r    |   replica |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files addns -n testns  -r 1 -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79 
```

### download

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --fileid  |      -f    |  file's id in XuperDB |   no    |
|   --filename  |      -m    |  file's name in XuperDB |    no    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --output  |      -o    |   output file path |    yes    |
|   --privkey  |      -k    |   private key |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files download -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79 -n test -m bigfile -o ./testdata/bigfile 
```

### getbyid

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i   |  file's id in XuperDB |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files getbyid  --id d86737bf-97ac-427f-a835-871d307c3589
```

### getbyname
|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --filename  |      -m    |  file's name |    yes    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files getbyname  -n testns -m bigfile -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```

### getns

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --namespace  |      -n    |   namespace |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files getns -n testns  -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 
```

### list

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --owner  |      -o    |  DataOwner's public key |    yes    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --limit  |  -l   |   limit for list, 0 for unlimited|    no    |
|   --start  |      -s   |   start time of the slice migrate' query |    no    |
|   --end  |      -e   |   end time of the slice migrate' query |    no    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files list -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n test -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00" --ext "{'FileType':'csv','Features':'room,price', 'TotalRows':400}"
```


### listexp

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --owner  |      -o    |  DataOwner's public key |    yes    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --limit  |  -l   |   limit for list, 0 for unlimited|    no    |
|   --start  |      -s   |   start time of the slice migrate' query |    no    |
|   --end  |      -e   |   end time of the slice migrate' query |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files listexp -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n test -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### listns

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --owner  |      -o    |  DataOwner's public key |    yes    |
|   --limit  |  -l   |   limit for list, 0 for unlimited|    no    |
|   --start  |      -s   |   start time of the slice migrate' query |    no    |
|   --end  |      -e   |   end time of the slice migrate' query |    no    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files listns -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 
```

### syshealth

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --owner  |      -o    |  DataOwner's public key |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files syshealth -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6
```
### upload

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --description  |      -d    |   description |    yes    |
|   --privkey  |      -k    |   private key |    yes    |
|   --expireTime  |      -e    |   expiretime of the file in XuperDB |    yes    |
|   --ext  |        |   file extra info |    yes    |
|   --filename  |      -m    |  file's name in XuperDB |    yes    |
|   --namespace  |      -n    |   namespace |    yes    |
|   --input  |      -i    |  input file path |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files upload -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79 -n test -m bigfile -i ./bin/client -e "2021-06-30 15:00:00" -d "this is a test file"
```

### ureplica

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --namespace  |      -n    |   namespace |    yes    |
|   --privkey  |      -k    |   private key |    yes    |
|   --replica  |      -r    |   replica |    yes    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files ureplica -n py  -r 3 -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79
```

### utime

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |  file's id in XuperDB |   no    |
|   --privkey  |      -k    |   private key |    yes    |
|   --expireTime  |      -e    |   expiretime of the file in XuperDB |    yes    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 files utime -e '2021-08-08 15:15:04' -i b87b588f-2e46-4ee5-8128-888592ada4fd -k 5572e2fa0c259fe798e5580884359a4a6ac938cfff62d027b90f2bac3eceef79
```

## Command Parsing: `xdb-cli challenge`

| command    |        explanation      |
| ---------- |   -----------   |
| failed     | get failed challenges by filters  |
| get        | get pdp challenge by id   |
| proved     | get proved challenges by filters  |        
| toprove    | get ToProve challenges by filters  |        

### get
|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |  challenge's id |   no    |

```
DEMO:
$ ./xdb-cli --host http://localhost:8122 challenge get --id ff334622-1442-425c-8d57-3f1b915b84e8 
```

### proved

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --file  |      -f    |  file's id in XuperDB |   no    |
|   --limit  |  -l   |   limit for list num|    no    |
|   --list  |   |  whether or not show challenges list, 0:not show |  default 1  |
|   --node  |  -n  |  storage node's id |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    yes    |
|   --start  |      -s   |   start time of the query |    no    |
|   --end  |      -e   |   end time of the query |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8122 challenge proved -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### failed

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --file  |      -f    |  file's id in XuperDB |   no    |
|   --limit  |  -l   |   limit for list num|    no    |
|   --list  |   |  whether or not show challenges list, 0:not show |  default 1  |
|   --node  |  -n  |  storage node's id |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    yes    |
|   --start  |      -s   |   start time of the query |    no    |
|   --end  |      -e   |   end time of the query |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:8122 challenge failed -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00"
```

### toprove

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --file  |      -f    |  file's id in XuperDB |   no    |
|   --limit  |  -l   |   limit for list num|    no    |
|   --list  |   |  whether or not show challenges list, 0:not show |  default 1  |
|   --node  |  -n  |  storage node's id |    yes    |
|   --owner  |      -o    |  DataOwner's public key |    yes    |
|   --start  |      -s   |   start time of the query |    no    |
|   --end  |      -e   |   end time of the query |    no    |


```
DEMO:
$ ./xdb-cli --host http://localhost:81221 challenge toprove -o 363c4c996a0a6d83f3d8b3180019702be1b7bb7a5e2a61ce1ef9503a5ad55c4beb1c78d616355a58556010a3518c66526c6dc17b0bea3fe965042ad3adcfe3e6 -n 58c4fe74988b3bd62a99f143bd07eb1b1e27f77a0c2d90d1c76f84d1adbcb240c652c81f005e4a0a0b3f43c9ebfab713e0e68d74695701f5564478ee59354f58 -l 10 -s "2021-06-30 15:00:00" -e "2021-06-30 16:00:00" --list 0
```
