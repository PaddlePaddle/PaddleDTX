# Command-line Tool: requester-cli
The `requester-cli` is the client of Requester. It can help users begin work on the 
training and predicting.

## Command Parsing
The `requester-cli` only has one subcommand `requester-cli task`, related to task's management.
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
$  ./bin/requester-cli task getbyid  -i 87d22f67-6b84-4266-aec5-581ac3df09f9
```


### list


|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --pubkey  |      -p    |   public key |    yes    |
|   --st  |      -s    |   start of time ranges |    no    |
|   --et  |      -e    |   end of time ranges |    no, default 'now'    |
|   --limit  |      -l    |   maximum of tasks can be queried |    no, default is 100    |
|   --status  |          |   status of task, such as Confirming, Ready, ToProcess, Processing, Finished, Failed |    no, default query all    |

```
Demo:
$  ./bin/requester-cli task list -p 4637ef79f14b036ced59b76408b0d88453ac9e5baa523a86890aa547eac3e3a0f4a3c005178f021c1b060d916f42082c18e1d57505cdaaeef106729e6442f4e5
```

### publish

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --name  |      -n    |   task's name |    yes    |
|   --privkey  |      -k    |   private key |    yes    |
|   --type  |      -t    |   task type, 'train' or 'predict' |   yes    |
|   --algorithm  |      -a    |   algorithm assigned to task, 'linear-vl' or 'logistic-vl' |    yes    |
|   --files  |    -f      |  files IDs with ',' as delimiter |   yes   |
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

```shell
$  ./bin/requester-cli task publish -a "linear-vl" -l "MEDV" -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 -t "train" -n "房价预测任务" -d "it's a test" -p "id,id" -f "52357151-de44-445a-a137-9c79a33c12ed,21e44577-c57f-4c92-b97e-7213222062da"
```

### start
|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   task's id |    yes    |
|   --privkey  |      -k    |   private key |    yes    |

```
DEMO:
$  ./bin/requester-cli task start -i a109984d-d741-4aea-800e-a5d0cf2b1eaf -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 
```

### result
|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   task's id |    yes    |
|   --privkey  |      -k    |   private key |    yes    |
|   --output  |      -o    |  file to store prediction outcomes  |    yes    |

```
DEMO:
$  ./bin/requester-cli task result -i a109984d-d741-4aea-800e-a5d0cf2b1eaf -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21 -o ./output.csv --config ./conf/config.toml
```
