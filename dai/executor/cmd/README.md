# Command-line Tool: executor-cli
The `executor-cli` is the client of Executor. It was used to control executor's behavior on the task.
There are two major subcommands of `executor-cli` as follows.

| command      |        explanation      | 
| :----------: |   :-----------:   | 
| key      | generate the executor node private/public key pair |
| task     | A command helps to executor manage tasks |


## Command Parsing:  `executor-cli key`
The subcommand `executor-cli key` used to generate the Executor node private/public key pair.

| command    |        explanation      | 
| :----------: |   :-----------:   | 
| genkey       | generate a pair of key |  

### genkey

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --output  |      -o    |   output path |    yes    |

```
DEMO:
$  ./executor-cli key genkey -o ./keys
```

### Command Parsing: `executor-cli task`
The subcommand `executor-cli task` related to task's management.
The detailed explanation is shown as follows.

| command    |        explanation      |
| :----------: |   :-----------:   |
| getbyid    | get a task by id |
| list       | list tasks of the executor node |
   
| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --host |      -h    |   the executor's host | yes |

### getbyid

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   task's id |    yes    |

```
DEMO:
$ ./executor-cli --host localhost:8184 task getbyid -i 87d22f67-6b84-4266-aec5-581ac3df09f9 
```

### list

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --pubkey  |      -p    |   public key |  requester or executor public key hex string, you can replace 'pubkey' with 'keyPath'    |
|   --keyPath  |        |  the file path of the node's public key |    no, default './keys'    |
|   --start  |      -s    |   start of time ranges |    no    |
|   --end  |      -e    |   end of time ranges |    no, default 'now'    |
|   --limit  |      -l    |   maximum of tasks can be queried |    no, default is 100    |
|   --status  |          |   status of task, such as Confirming, Ready, ToProcess, Processing, Finished, Failed |    no, default query all    |

```shell
$ ./executor-cli --host localhost:8184 task list --keyPath ./keys -l 10 -s "2021-09-30 15:00:00" -e "2021-11-30 16:00:00" 
```