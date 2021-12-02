# 快速安装

我们为您提供了能够快速拉起 PaddleDTX 测试网络的脚本，在使用前需要您准备如下环境:

* docker, 推荐版本18.03+ [点击下载安装 docker](https://docs.docker.com/get-docker/)
* docker-compose, 推荐版本1.26.0+ [点击下载安装 docker-compose](https://github.com/docker/compose/releases)

环境准备好之后，可以通过执行脚本快速拉起网络：
```
$ cd PaddleDTX/scripts
$ sh network_up.sh start
```

使用脚本也可以快速销毁网络：
```
$ sh network_up.sh stop
```

对于网络的应用可以参考命令行工具及相关案例。