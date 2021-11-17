[English](./README.md) | 中文

# XuperDB
敬请期待

## 一、项目简介
敬请期待

### 1.1 架构概览
敬请期待

![Image text](./images/architecture_overview.png)

### 1.2 核心技术
敬请期待

## 二、安装
XuperDB包含：
* xdata DataOwner或者Storage节点服务程序
* client DataOwner或者Storage节点命令行工具

我们提供两种方式安装XuperDB，您可以根据自己的实际情况进行选择：

### 2.1 通过Docker安装
**强烈建议**您通过docker安装XuperDB。
您可以参考 [XuperDB镜像制作脚本](./build_image.sh) 制作docker镜像，也可以使用我们提供的镜像构建系统，请参考 [docker-compose部署XuperDB](./testdata/docker-compose.yml)。

### 2.2 源码安装
编译依赖

* go 1.13.x 及以上

```sh
# In xdb directory
make
```
您可以在 `./output` 中获取安装包，然后手动安装。

## 三、测试
请参考 [命令行工具说明](./cmd/client/README.md) 了解和测试XuperDB。
