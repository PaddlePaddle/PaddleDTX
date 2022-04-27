# 贡献指南

## 团队成员

团队的主要成员来自 **百度超级链研发团队** 和 **联通链研发团队（孙林、贾晓芸、马宇峰、庄媛等）**，具备区块链、隐私计算、大数据、分布式机器学习等多领域的技术背景。

如果您对我们的研究方向或者技术团队感兴趣，欢迎加入我们。 [点击投递简历](https://talent.baidu.com/external/baidu/index.html#/social/2/%E5%8C%BA%E5%9D%97%E9%93%BE%E7%B3%BB%E7%BB%9F%E9%83%A8)


## 参与开发&测试

### 1. 代码提交指南

PaddleDTX 欢迎任何形式的贡献（包括但不限于贡献新feature，完善文档，参加线下活动，提Issue)。对于想给 PaddleDTX 贡献代码的开发者，在这里我们以给 https://github.com/PaddlePaddle/PaddleDTX 这个仓库提交代码为例来详细解释代码提交流程。

#### 1.1 Fork 代码
首先访问 https://github.com/PaddlePaddle/PaddleDTX ，fork代码到自己的代码仓库：

<img src='../../_static/fork.png' width = "100%" height = "100%" align="middle"/>

<br>

#### 1.2 Clone代码到本地
假设fork完之后的代码仓库路径为 https://github.com/MyYuan/PaddleDTX：

```
git clone git@github.com:MyYuan/PaddleDTX.git
```
之后再设置一个upstream的remote地址，方便我们同步原始仓库地址的更新
```
git remote add upstream git@github.com:PaddlePaddle/PaddleDTX.git
```
#### 1.3 同步代码并建立新分支
我们每次要提交PR的时候都要新建一个分支，这样可以同时开发多个feature，分支基于upstream的master建立：

```
# 拉取上游的最新代码
git fetch upstream

# 建立新分支
git checkout -b new_feature upstream/master

```
之后我们就可以在这个分支上开发我们的代码了

#### 1.4 提交代码
当我们的代码写完之后就可以提交了，注意我们这里提交的remote是origin，也就是自己的代码仓库 https://github.com/MyYuan/PaddleDTX：

```
$ git push origin new_feature
```
提交之后返回：
```
[new_feature 294a22e] update config.toml
 1 file changed, 1 insertion(+)
MacBook-Pro:dai panyuan$ git push origin new_feature
Enumerating objects: 9, done.
Counting objects: 100% (9/9), done.
Delta compression using up to 8 threads
Compressing objects: 100% (5/5), done.
Writing objects: 100% (5/5), 405 bytes | 405.00 KiB/s, done.
Total 5 (delta 4), reused 0 (delta 0)
remote: Resolving deltas: 100% (4/4), completed with 4 local objects.
remote: 
remote: Create a pull request for 'new_feature' on GitHub by visiting:
remote:      https://github.com/MyYuan/PaddleDTX/pull/new/new_feature
remote: 
To github.com:MyYuan/PaddleDTX.git
 * [new branch]      new_feature -> new_feature
```
#### 1.5 创建PR
提交完之后，一般有个类似 https://github.com/MyYuan/PaddleDTX/pull/new/new_feature 这样的地址，在浏览器打开这个地址就跳转到创建PR的页面：

<img src='../../_static/submit_pr.png' width = "100%" height = "100%" align="middle"/>

<br>

#### 1.6 持续提交修改补丁
在review的过程中，会有人提出修改意见，继续在new_feature分支上添加commit，再push，就会在当前的PR上进行更新。

```
git add -u
git commit -m 'some fix'
git push origin new_feature
```
#### 1.7 合入代码
如果代码的CI过了，reviewer也没有意见就会合入代码，代码就进入了master分支，之后就可以删除本地和远端的new_feature分支：
```
git branch -D new_feature
```

### 2. 提交Issue

打开 https://github.com/PaddlePaddle/PaddleDTX/issues，点击`New issue`，即可提交用户在使用中的各类问题。

<br>