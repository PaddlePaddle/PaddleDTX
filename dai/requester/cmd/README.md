# Command-line Tool: requester-cli
The `requester-cli` is the client of Requester. It can help users begin work on the training and predicting.
There are four major subcommands of `requester-cli` as follows.

| command      |        explanation      | 
| :----------: |   :-----------:   | 
| files    | query sample files info used when the task is published | 
| key      | generate the requester client private/public key pair |
| nodes    | query executor nodes used when the task is published |
| task     | the subcommands related to task's management |

## Command Parsing:  `requester-cli files`
The subcommand `requester-cli files` used to query the sample file's authorization application info.

| command      |        explanation      | 
| :----------: |   :-----------:   | 
| getauthbyid  | get the file authorization application detail |  
| listauth     | list file authorization applications |

| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --conf |      -c    |   configuration file  | no, the default is "./conf/config.toml" |

### getauthbyid

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --authID  |     -i    |   id of file authorization application |    yes    |

```
DEMO:
$  ./requester-cli files getauthbyid  -i 82bd3362-a95e-4c8b-9512-f17fc4091309
```

### listauth

|     flag    |  short flag   | explanation | necessary |
| :---------: | :-----------: | :------------: | :---------: |
|   --applier |      -a       |   applier's public key, often known as executor's public key |    no, but owner or applier cannot both be empty   |
|   --owner   |      -o       |   file owner |    no, but owner or applier cannot both be empty    |
|   --end     |      -e       |   authorization applications publish before endTime, example '2022-07-10 12:00:00' |    no    |
|   --fileID  |      -f       |   sample file ID |    no    |
|   --start   |      -s       |   authorization applications publish after startTime, example '2022-06-10 12:00:00' |    no    |
|   --limit   |      -l       |   slimit for list file authorization applications |    no    |
|   --status  |               |   status of file authorization application |    no    |

```
DEMO:
$  ./requester-cli files listauth  -a  b02fe5f7d12bf63131bb98c339f312c53ddf126e04a9a8b85d29bc3d74f2e7c04009db13e9d48039d0738f86fd71693187d2ed6bdf193dc260b0d594728b9e09
```

## Command Parsing:  `requester-cli key`
The subcommand `requester-cli key` used to generate the Requester client private/public key pair.

| command    |        explanation      | 
| :----------: |   :-----------:   | 
| genkey       | generate a pair of key |  

### genkey

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --output  |      -o    |   output path |    yes    |

```
DEMO:
$  ./requester-cli key genkey -o ./keys
```

## Command Parsing:  `requester-cli nodes`
The subcommand `requester-cli nodes` used to query executor nodes when the Requester publishes tasks.

| command    |        explanation      | 
| :----------: |   :-----------:   | 
| getbyid    | get the executor node by id |  
| getbyname  | get the executor node by name |
| list       | list executor nodes |


| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --conf |      -c    |   configuration file  | no, the default is "./conf/config.toml" |

### getbyid

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   executor node id |    yes    |

```
DEMO:
$  ./requester-cli nodes getbyid  -i b02fe5f7d12bf63131bb98c339f312c53ddf126e04a9a8b85d29bc3d74f2e7c04009db13e9d48039d0738f86fd71693187d2ed6bdf193dc260b0d594728b9e09
```

### getbyname

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --name  |      -n    |   executor node name |    yes    |

```
DEMO:
$  ./requester-cli nodes getbyname  -n executor1
```

### list

```
DEMO:
$  ./requester-cli nodes list
```


## Command Parsing: `requester-cli task`
The subcommand `requester-cli task` related to task's management.
The detailed explanation is shown as follows.

| command    |        explanation      | 
| :----------: |   :-----------:   | 
| getbyid    | get the task by id |  
| list       | list all tasks |
| publish    | publish a training task or prediction task |
| start      | start the confirmed task |
| result     | get predict task result from executor node |


| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --conf |      -c    |   configuration file  | no, the default is "./conf/config.toml" |
   
### getbyid

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   task's id |    yes    |

```
DEMO:
$  ./requester-cli task getbyid  -i 87d22f67-6b84-4266-aec5-581ac3df09f9
```


### list


|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --pubkey  |      -p    |   public key |    you can replace 'pubkey' with 'keyPath'    |
|   --keyPath  |        |  the file path of the requester client's public key |    no, default './keys'    |
|   --st  |      -s    |   start of time ranges |    no    |
|   --et  |      -e    |   end of time ranges |    no, default 'now'    |
|   --limit  |      -l    |   maximum of tasks can be queried |    no, default is 100    |
|   --status  |          |   status of task, such as Confirming, Ready, ToProcess, Processing, Finished, Failed |    no, default query all    |

```
Demo:
$  ./requester-cli task list  --keyPath ./keys
```

### publish

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --name  |      -n    |   task's name |    yes    |
|   --privkey  |      -k    |   private key |    no, can be replaced by 'keyPath'    |
|   --keyPath  |        |  the file path of the requester client's private key |    no, default './keys'    |
|   --type  |      -t    |   task type, 'train' or 'predict' |   yes    |
|   --algorithm  |      -a    |   algorithm assigned to task, 'linear-vl' or 'logistic-vl' |    yes    |
|   --files  |    -f      |  files IDs with ',' as delimiter |   yes   |
|   --executors  |    -e      |  executor node names with ',' as delimiter, like 'executor1,executor2' |   yes   |
|   --label  |      -l    |   training task's target feature  |    yes in training task, no in prediction task   |
|   --labelName  |          |   target variable required in logistic-vl training task | yes in logistic-vl training task, no in others    |
|   --PSILabel  |      -p    |  labels used by PSI process |   yes    |
|   --taskId  |      -i   |   algorithm assigned to task, 'linear-vl' or 'logistic-vl' |    yes    |
|   --regMode  |          | regularization mode of training task, can be l1(L1-norm) or l2(L2-norm)  |   no, default no regularization   |
|   --regParam  |          | regularization parameter |   no, default is 0.1   |
|   --alpha  |          |   learning rate alpha |    no, default is 0.1    |
|   --amplitude  |    amplitude      |   |   no, default is 0.0001   |
|   --accuracy  |      accuracy    |    |    no, default is 10    |
|   --description  |    -d      | task  description  |   no   |
|   --batchSize  |    -b      |  size of samples for one round of training loop, |   no, default is 4   |
|   --ev  |          | perform model evaluation |   no   |
|   --evRule  |          | the way to evaluate model, 0 means 'Random Split', 1 means 'Cross Validation', 2 means 'Leave One Out' |   no, default is 0   |
|   --folds  |          | number of folds, 5 or 10 supported, a optional parameter when perform model evaluation in the way of 'Cross Validation' |   no, default is 10   |
|   --shuffle  |          | shuffle the samples before division when perform model evaluation in the way of 'Cross Validation' |   no   |
|   --plo  |          | percentage to leave out as validation set when perform model evaluation in the way of 'Random Split' |   no, default is 30   |
|   --le  |          | perform live model evaluation |   no   |
|   --lplo  |          | percentage to leave out as validation set when perform live model evaluation |   no, default is 30   |

```shell
$  ./requester-cli task publish -a "linear-vl" -l "MEDV" -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 -t "train" -n "房价预测任务" -d "it's a test" -p "id,id" -f "52357151-de44-445a-a137-9c79a33c12ed,21e44577-c57f-4c92-b97e-7213222062da" -e "executor1,executor2"
```

### start
|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   task's id |    yes    |
|   --privkey  |      -k    |   private key |    you can replace 'privkey' with 'keyPath'    |
|   --keyPath  |        |  the file path of the requester client's private key |    no, default './keys'    |


```
DEMO:
$  ./requester-cli task start -i a109984d-d741-4aea-800e-a5d0cf2b1eaf --keyPath ./keys
```

### result
|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   task's id |    yes    |
|   --privkey  |      -k    |   private key |    you can replace 'privkey' with 'keyPath'    |
|   --keyPath  |        |  the file path of the requester client's private key |    no, default './keys'    |
|   --output  |      -o    |  file to store prediction outcomes  |    yes    |

```
DEMO:
$  ./requester-cli task result -i a109984d-d741-4aea-800e-a5d0cf2b1eaf --keyPath ./keys -o ./output.csv --config ./conf/config.toml
```
