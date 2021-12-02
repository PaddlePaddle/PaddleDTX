# Command-line Tool: executor-cli
The `executor-cli` is the client of Executor. It was used to control executor's behavior on the task.

### Command Parsing [./bin/executor-cli task]
The `executor-cli` only has one subcommand `executor-cli task`, related to task's management.
The detailed explanation is shown as follows.

| command    |        explanation      |
| :----------: |   :-----------:   |
| confirm    | confirm task in Ready status |
| getbyid    | get a task by id |
| list       | list tasks of the executor node |
   
| global flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :------: | 
|   --host |      -h    |   the executor's host | yes |
   
### confirm

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   task's id |    yes    |
|   --privkey  |      -k    |   private key |    yes    |

```
DEMO:
$ ./bin/executor-cli --host localhost:8184 task confirm -i a109984d-d741-4aea-800e-a5d0cf2b1eaf -k 14a54c188d0071bc1b161a50fe7eacb74dcd016993bb7ad0d5449f72a8780e21
```

### getbyid

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --id  |      -i    |   task's id |    yes    |

```
DEMO:
$ ./bin/executor-cli --host localhost:8184 task getbyid -i 87d22f67-6b84-4266-aec5-581ac3df09f9 
```

### list

|  flag  | short flag | explanation | necessary |
| :------: | :----------: | :------------: | :---------: |
|   --pubkey  |      -p    |   public key |    yes    |
|   --st  |      -s    |   start of time ranges |    no    |
|   --et  |      -e    |   end of time ranges |    no, default 'now'    |
|   --limit  |      -l    |   maximum of tasks can be queried |    no, default is 100    |
|   --status  |          |   status of task, such as Confirming, Ready, ToProcess, Processing, Finished, Failed |    no, default query all    |


```shell
$ ./bin/executor-cli --host localhost:8184 task list -p 4637ef79f14b036ced59b76408b0d88453ac9e5baa523a86890aa547eac3e3a0f4a3c005178f021c1b060d916f42082c18e1d57505cdaaeef106729e6442f4e5 -l 10 -s "2021-09-30 15:00:00" -e "2021-11-30 16:00:00" 
```