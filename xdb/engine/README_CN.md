# Engine

Engine 模块是 XuperDB 的核心调度引擎，承担 Handler 和 Monitor 的大多数任务

## 模块划分
- slicer 数据切片模块
- copier 数据冗余模块
- encryptor 加密模块
- challenger 存在性证明模块
- monitor 监控任务模块
- handler 功能入口

### Handler

#### Write
数据写入口分别对用户上传的数据执行 分片、定位、加密、分发、上链 等操作。其中的各个步骤采用异步流式处理，步骤之间通过 chan 进行协调，主要是为了降低大文件对内存的需求。

#### Read
数据读入口采用滑动窗口的方式异步获取各个分片，主要也是为了降低大文件对内存的需求，同时提高时间利用率
