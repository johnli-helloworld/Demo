## 目录
- filecoin源码协议层分析之hello握手协议
    -   1. 执行流程
    -   2. 目的
    -   3. 源码信息
    -   4. 源码分析

当我们执行go-filecoin daemon命令时，会执行go-filecoin/commands/daemon.go中的daemonRun（）函数

如下：

![](Z:\go\src\Demo\hello-protocal\pic\node-daemon1.png)

