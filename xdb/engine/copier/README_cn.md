# copier

copier 是数据冗余模块，为文件切片提供多副本选择。

## 模块划分
- random: 提供切片的随机多副本选择功能，优先从健康存储节点中选择指定数量的节点来存储文件切片。