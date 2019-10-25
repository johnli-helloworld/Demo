## 目录

- [1.lotus](#1lotus)
  - [1.1 启动守护进程](#11-%E5%90%AF%E5%8A%A8%E5%AE%88%E6%8A%A4%E8%BF%9B%E7%A8%8B)
  - [1.2 Auth](#12-auth)
    - [1.2.1 创建admin令牌](#121-%E5%88%9B%E5%BB%BAadmin%E4%BB%A4%E7%89%8C)
  - [1.3 chain](#13-chain)
    - [1.3.1 获取Chainhead cid](#131-%E8%8E%B7%E5%8F%96chainhead-cid)
    - [1.3.2  获取block详情](#132--%E8%8E%B7%E5%8F%96block%E8%AF%A6%E6%83%85)
    - [1.3.3 读取对象的原始字节](#133-%E8%AF%BB%E5%8F%96%E5%AF%B9%E8%B1%A1%E7%9A%84%E5%8E%9F%E5%A7%8B%E5%AD%97%E8%8A%82)
    - [1.3.4 根据msgid获取message](#134-%E6%A0%B9%E6%8D%AEmsgid%E8%8E%B7%E5%8F%96message)
    - [1.3.5 手动设置本地节点chainhead（通常用于恢复）](#135-%E6%89%8B%E5%8A%A8%E8%AE%BE%E7%BD%AE%E6%9C%AC%E5%9C%B0%E8%8A%82%E7%82%B9chainhead%E9%80%9A%E5%B8%B8%E7%94%A8%E4%BA%8E%E6%81%A2%E5%A4%8D)
    - [1.3.6 获取chain list](#136-%E8%8E%B7%E5%8F%96chain-list)
  - [1.4 client](#14-client)
    - [1.4.1 导入文件](#141-%E5%AF%BC%E5%85%A5%E6%96%87%E4%BB%B6)
    - [1.4.2 查看本地本导入的文件](#142-%E6%9F%A5%E7%9C%8B%E6%9C%AC%E5%9C%B0%E6%9C%AC%E5%AF%BC%E5%85%A5%E7%9A%84%E6%96%87%E4%BB%B6)
    - [1.4.3 创建订单](#143-%E5%88%9B%E5%BB%BA%E8%AE%A2%E5%8D%95)
    - [1.4.4 在网络上查找文件](#144-%E5%9C%A8%E7%BD%91%E7%BB%9C%E4%B8%8A%E6%9F%A5%E6%89%BE%E6%96%87%E4%BB%B6)
    - [1.4.5 在网络上检索文件](#145-%E5%9C%A8%E7%BD%91%E7%BB%9C%E4%B8%8A%E6%A3%80%E7%B4%A2%E6%96%87%E4%BB%B6)
    - [1.4.6 查询报价单](#146-%E6%9F%A5%E8%AF%A2%E6%8A%A5%E4%BB%B7%E5%8D%95)
  - [1.5 创建存储矿工](#15-%E5%88%9B%E5%BB%BA%E5%AD%98%E5%82%A8%E7%9F%BF%E5%B7%A5)
  - [1.6 获取证明参数](#16-%E8%8E%B7%E5%8F%96%E8%AF%81%E6%98%8E%E5%8F%82%E6%95%B0)
  - [1.7 mpool](#17-mpool)
    - [1.7.1 获取在等待的消息](#171-%E8%8E%B7%E5%8F%96%E5%9C%A8%E7%AD%89%E5%BE%85%E7%9A%84%E6%B6%88%E6%81%AF)
  - [1.8 Net](#18-net)
    - [1.8.1 获取网络中的对等节点](#181-%E8%8E%B7%E5%8F%96%E7%BD%91%E7%BB%9C%E4%B8%AD%E7%9A%84%E5%AF%B9%E7%AD%89%E8%8A%82%E7%82%B9)
    - [1.8.2 连接节点](#182-%E8%BF%9E%E6%8E%A5%E8%8A%82%E7%82%B9)
    - [1.8.3 列出监听的节点](#183-%E5%88%97%E5%87%BA%E7%9B%91%E5%90%AC%E7%9A%84%E8%8A%82%E7%82%B9)
    - [1.8.4 查看本节点id](#184-%E6%9F%A5%E7%9C%8B%E6%9C%AC%E8%8A%82%E7%82%B9id)
  - [1.9 paych](#19-paych)
    - [1.9.1 创建支付渠道或获取以存在的支付渠道](#191-%E5%88%9B%E5%BB%BA%E6%94%AF%E4%BB%98%E6%B8%A0%E9%81%93%E6%88%96%E8%8E%B7%E5%8F%96%E4%BB%A5%E5%AD%98%E5%9C%A8%E7%9A%84%E6%94%AF%E4%BB%98%E6%B8%A0%E9%81%93)
    - [1.9.2 查看已注册的支付渠道](#192-%E6%9F%A5%E7%9C%8B%E5%B7%B2%E6%B3%A8%E5%86%8C%E7%9A%84%E6%94%AF%E4%BB%98%E6%B8%A0%E9%81%93)
    - [1.9.3 voucher](#193-voucher)
      - [1.9.3.1 创建支付凭证](#1931-%E5%88%9B%E5%BB%BA%E6%94%AF%E4%BB%98%E5%87%AD%E8%AF%81)
      - [1.9.3.2 校验支付凭证](#1932-%E6%A0%A1%E9%AA%8C%E6%94%AF%E4%BB%98%E5%87%AD%E8%AF%81)
      - [1.9.3.3 添加支付凭证到本地](#1933-%E6%B7%BB%E5%8A%A0%E6%94%AF%E4%BB%98%E5%87%AD%E8%AF%81%E5%88%B0%E6%9C%AC%E5%9C%B0)
      - [1.9.3.4 列举支付渠道内的所有凭证](#1934-%E5%88%97%E4%B8%BE%E6%94%AF%E4%BB%98%E6%B8%A0%E9%81%93%E5%86%85%E7%9A%84%E6%89%80%E6%9C%89%E5%87%AD%E8%AF%81)
      - [1.9.3.5 获取当前可用的最高价值的凭证](#1935-%E8%8E%B7%E5%8F%96%E5%BD%93%E5%89%8D%E5%8F%AF%E7%94%A8%E7%9A%84%E6%9C%80%E9%AB%98%E4%BB%B7%E5%80%BC%E7%9A%84%E5%87%AD%E8%AF%81)
      - [1.9.3.6 提交凭证上链并更新支付渠道状态](#1936-%E6%8F%90%E4%BA%A4%E5%87%AD%E8%AF%81%E4%B8%8A%E9%93%BE%E5%B9%B6%E6%9B%B4%E6%96%B0%E6%94%AF%E4%BB%98%E6%B8%A0%E9%81%93%E7%8A%B6%E6%80%81)
  - [2.0 send 转账](#20-send-%E8%BD%AC%E8%B4%A6)
  - [2.1 state](#21-state)
    - [2.1.1 查询算力](#211-%E6%9F%A5%E8%AF%A2%E7%AE%97%E5%8A%9B)
    - [2.1.2 查询矿工的扇区集](#212-%E6%9F%A5%E8%AF%A2%E7%9F%BF%E5%B7%A5%E7%9A%84%E6%89%87%E5%8C%BA%E9%9B%86)
    - [2.1.3 查询矿工证明集](#213-%E6%9F%A5%E8%AF%A2%E7%9F%BF%E5%B7%A5%E8%AF%81%E6%98%8E%E9%9B%86)
    - [2.1.4 查询最小的矿工质押](#214-%E6%9F%A5%E8%AF%A2%E6%9C%80%E5%B0%8F%E7%9A%84%E7%9F%BF%E5%B7%A5%E8%B4%A8%E6%8A%BC)
    - [2.1.5 列举网络上所有的actor](#215-%E5%88%97%E4%B8%BE%E7%BD%91%E7%BB%9C%E4%B8%8A%E6%89%80%E6%9C%89%E7%9A%84actor)
    - [2.1.6 列举网络上所有的miner actor](#216-%E5%88%97%E4%B8%BE%E7%BD%91%E7%BB%9C%E4%B8%8A%E6%89%80%E6%9C%89%E7%9A%84miner-actor)
  - [2.2 sync](#22-sync)
    - [2.2.1 查询同步状态](#221-%E6%9F%A5%E8%AF%A2%E5%90%8C%E6%AD%A5%E7%8A%B6%E6%80%81)
    - [2.2.2 等待同步完成](#222-%E7%AD%89%E5%BE%85%E5%90%8C%E6%AD%A5%E5%AE%8C%E6%88%90)
  - [2.3 获取版本](#23-%E8%8E%B7%E5%8F%96%E7%89%88%E6%9C%AC)
  - [2.4 wallet](#24-wallet)
    - [2.4.1 创建钱包](#241-%E5%88%9B%E5%BB%BA%E9%92%B1%E5%8C%85)
    - [2.4.2 列举所有钱包](#242-%E5%88%97%E4%B8%BE%E6%89%80%E6%9C%89%E9%92%B1%E5%8C%85)
    - [2.4.3 获取钱包金额](#243-%E8%8E%B7%E5%8F%96%E9%92%B1%E5%8C%85%E9%87%91%E9%A2%9D)
    - [2.4.4 导出钱包](#244-%E5%AF%BC%E5%87%BA%E9%92%B1%E5%8C%85)
    - [2.4.5 导入钱包](#245-%E5%AF%BC%E5%85%A5%E9%92%B1%E5%8C%85)
    - [2.4.6 获取默认钱包地址](#246-%E8%8E%B7%E5%8F%96%E9%BB%98%E8%AE%A4%E9%92%B1%E5%8C%85%E5%9C%B0%E5%9D%80)
    - [2.4.7 设置默认钱包](#247-%E8%AE%BE%E7%BD%AE%E9%BB%98%E8%AE%A4%E9%92%B1%E5%8C%85)
- [2. lotus-storage-miner](#2-lotus-storage-miner)
  - [2.1 初始化](#21-%E5%88%9D%E5%A7%8B%E5%8C%96)
  - [2.2 开始挖矿](#22-%E5%BC%80%E5%A7%8B%E6%8C%96%E7%9F%BF)
  - [2.3 查看矿工详情](#23-%E6%9F%A5%E7%9C%8B%E7%9F%BF%E5%B7%A5%E8%AF%A6%E6%83%85)
  - [2.4 在扇区中存储任意数据](#24-%E5%9C%A8%E6%89%87%E5%8C%BA%E4%B8%AD%E5%AD%98%E5%82%A8%E4%BB%BB%E6%84%8F%E6%95%B0%E6%8D%AE)
  - [2.5 Auth （同上）](#25-auth-%E5%90%8C%E4%B8%8A)
  - [2.6 chain（同上）](#26-chain%E5%90%8C%E4%B8%8A)
  - [2.7 client（同上）](#27-client%E5%90%8C%E4%B8%8A)
  - [2.8 创建存储矿工（同上）](#28-%E5%88%9B%E5%BB%BA%E5%AD%98%E5%82%A8%E7%9F%BF%E5%B7%A5%E5%90%8C%E4%B8%8A)
  - [2.9 获取证明参数（同上）](#29-%E8%8E%B7%E5%8F%96%E8%AF%81%E6%98%8E%E5%8F%82%E6%95%B0%E5%90%8C%E4%B8%8A)
  - [3.0 mpool（同上）](#30-mpool%E5%90%8C%E4%B8%8A)
  - [3.1 net（同上）](#31-net%E5%90%8C%E4%B8%8A)
  - [3.2 paych（同上）](#32-paych%E5%90%8C%E4%B8%8A)
  - [3.3 send转账（同上）](#33-send%E8%BD%AC%E8%B4%A6%E5%90%8C%E4%B8%8A)
  - [3.4 state（同上）](#34-state%E5%90%8C%E4%B8%8A)
  - [3.5 sync（同上）](#35-sync%E5%90%8C%E4%B8%8A)
  - [3.6 获取版本（同上）](#36-%E8%8E%B7%E5%8F%96%E7%89%88%E6%9C%AC%E5%90%8C%E4%B8%8A)
  - [3.7 wallet（同上）](#37-wallet%E5%90%8C%E4%B8%8A)

## 1.lotus

### 1.1 启动守护进程

```sh
lotus daemon
```

- 参数注释：

```
可选：
api 		端口号，默认1234
```

回到[目录](#m目录)

### 1.2 Auth

#### 1.2.1 创建admin令牌

```sh
lotus auth create-admin-token     //此命令创建的token具有所有权限
```

回到[目录](#m目录)

### 1.3 chain

#### 1.3.1 获取Chainhead cid

```
lotus chain head
```

#### 1.3.2  获取block详情

```
lotus chain getblook <blockcid>
```

- 参数注释

```
必填：
blockcid	block cid
```

#### 1.3.3 读取对象的原始字节

```
lotus chain read-obj <cid>
```

- 参数注释

```
必填：
cid		读取对象的cid
```

#### 1.3.4 根据msgid获取message

```
lotus chain getmessage <msgcid>
```

- 参数注释

```
必填：
msgid		消息cid
```

#### 1.3.5 手动设置本地节点chainhead（通常用于恢复）

```
lotus chain sethead	<newheadcid>
```

- 参数注释

```
可选：
gensis			是否重置chainhead为创世块，bool值，默认false
newheadcid		需要设置的新tipset的cid
```

#### 1.3.6 获取chain list

```
lotus chain list --height=<unit64> --count=<unit64> --format="xxx"
```

- 参数注释

```
可选：
height		指定获取多少高度之前的block;默认值0
count		指定获取的block数量;默认值30
format		以什么格式打印;默认("<height>: (<time>) <blocks>")
```

回到[目录](#m目录)



### 1.4 client

#### 1.4.1 导入文件

```
lotus client import <filepath>
```

目前存在问题：导入报错：“cannot add filestore references outside ipfs root”，测试得知需要将文件放在/root目录下并且需要使用绝对路径

- 参数注释

```
filepath		文件的绝对路径
```

#### 1.4.2 查看本地本导入的文件

```
lotus client local
```

#### 1.4.3 创建订单

```
lotus client deal <datacid> <miner> <price> <duration>
```

- 参数注释

```
datacid			存储文件cid
miner			指定存储矿工地址
price			attoFIL/byte/block
duration		你想要存储多长时间（在约30秒的块时间内）。例如，储存1天（2块/分钟* 60分钟/小时* 24小时/天）= 2880块。
```

#### 1.4.4 在网络上查找文件 

```sh
lotus client find <cid> 
```

- 参数注释

```
cid		filecid

// return: offer的相关信息(RETRIEVAL: offer.Miner, offer.MinerPeerID, offer.MinPrice, offer.Size)
```

#### 1.4.5 在网络上检索文件

未验证

```
lotus client retrieve <cid> <outfile> 
```

- 参数注释

```
必填：
cid			文件cid
outfile		导出文件绝对路径
可选：
address		支付地址，默认钱包地址
```

#### 1.4.6 查询报价单

未验证成功

```
lotus client query-ask <address> --peerid=<peerid> --size=<size> --duration=<duration>
```

- 参数注释

```
必填：
address		指定mineraddr
可选：
peerid		指定要查询节点的peerid
size		指定存储大小
duration	存储多长时间

return：Ask:<mineraddr> ;Price per Byte:<Ask.Price>; Price per Block:<Ask.Price/size>
	Total Price:<Ask.Price*size*duration>
```

回到[目录](#m目录)



### 1.5 创建存储矿工

未验证

```sh
lotus createminer <workeraddr> <owneraddr> <sector size> <peer ID>
```

- 参数注释

```
必填：
workeraddr		worker address
owneraddr		owner address
sector size		指定扇区大小
peerid			对等节点id
```

回到[目录](#m目录)



### 1.6 获取证明参数

```
lotus fetch-params
```



### 1.7 mpool

#### 1.7.1 获取在等待的消息

```
lotus mpool pending
```

回到[目录](#m目录)



### 1.8 Net

#### 1.8.1 获取网络中的对等节点

```
lotus net peers
```

#### 1.8.2 连接节点

```
lotus connect <peerid> ...
```

#### 1.8.3 列出监听的节点

```
lotus net listen 
```

#### 1.8.4 查看本节点id

```
lotus net id
```

回到[目录](#m目录)



### 1.9 paych

未验证

#### 1.9.1 创建支付渠道或获取以存在的支付渠道

```
lotus paych get <from> <to> <available funds>
```

- 参数注释

```
必填：
from			支付地址
to				接收地址
available funds	可用金额
```

#### 1.9.2 查看已注册的支付渠道

```
lotus paych list
```

#### 1.9.3 voucher

##### 1.9.3.1 创建支付凭证

```
lotus paych voucher create <channel> <amount>
```

- 参数注释

```
必填：
channel		支付渠道address
amount		支付金额
```

##### 1.9.3.2 校验支付凭证

```
lotus paych voucher check <channel> <voucher>
```

- 参数注释

```
必填：
channel		支付渠道address
voucher		凭证cid
```

##### 1.9.3.3 添加支付凭证到本地

```
lotus paych voucher add <channel> <voucher>
```

- 参数注释

```
必填：
channel		支付渠道address
voucher		凭证cid
```

##### 1.9.3.4 列举支付渠道内的所有凭证

```
lotus paych voucher list <channel>
```

- 参数注释

```
必填：
channel		支付渠道address
可选：
exports		输出字符串	
```

##### 1.9.3.5 获取当前可用的最高价值的凭证

```
lotus paych voucher best-spendable <channel>
```

- 参数注释

```
必填
channel		支付渠道address
```

##### 1.9.3.6 提交凭证上链并更新支付渠道状态

```
lotus paych voucher submit <channel> <voucher>
```

- 参数注释

```
必填：
channel		支付渠道address
voucher		凭证cid
```

回到[目录](#m目录)



### 2.0 send 转账

```
lotus send <target> <amount> --source=<source>
```

- 参数注释

```
必填：
target		接收方账户地址
amount		转账金额
可选：
source		转账账户地址，默认钱包地址
```

回到[目录](#m目录)



### 2.1 state

#### 2.1.1 查询算力

已验证

```
lotus state power
```

#### 2.1.2 查询矿工的扇区集 

```
lotus state sectors <mineraddr>
```

- 参数注释

```
必填：
mineraddr		miner actor addr
```

#### 2.1.3 查询矿工证明集

```
lotus state proving <mineraddr>
```

- 参数注释

```
必填：
mineraddr		miner actor addr
```

#### 2.1.4 查询最小的矿工质押

```
lotus state pledge-collateral <mineraddr>
```

- 参数注释

```
必填：
mineraddr		miner actor addr
```

#### 2.1.5 列举网络上所有的actor

```
lotus state list-actors
```

#### 2.1.6 列举网络上所有的miner actor

```
lotus state list-miners
```

回到[目录](#m目录)



### 2.2 sync

#### 2.2.1 查询同步状态

```
lotus sync status
```

#### 2.2.2 等待同步完成

```
lotus sync wait
```

回到[目录](#m目录)



### 2.3 获取版本

```
lotus version   //lotus version  and api version
```



### 2.4 wallet

#### 2.4.1 创建钱包

```
lotus wallet new 
```

#### 2.4.2 列举所有钱包

```
lotus wallet list
```

#### 2.4.3 获取钱包金额

```
lotus wallet balance <address>
```

- 参数注释

```
可选：
address		wallet address
```

#### 2.4.4 导出钱包

```
lotus wallet export <address>
```

- 参数注释

```
必填：
address		wallet address
```

#### 2.4.5 导入钱包

```
lotus wallet import 
```

#### 2.4.6 获取默认钱包地址

```
lotus wallet default
```

#### 2.4.7 设置默认钱包

```
lotus wallet set-default <address>
```

- 参数注释

```
必填：
address		wallet address
```

回到[目录](#m目录)



## 2. lotus-storage-miner

### 2.1 初始化

```
lotus-storage-miner init --actor=<actor> --owner=<owneraddr> --worker=<workeraddr>
```

- 参数注释

```
可选：
actor		miner actor address
owner		owner address
worker		worker address
```

回到[目录](#m目录)



### 2.2 开始挖矿

```
lotus-storage-miner run
```



### 2.3 查看矿工详情

```
lotus-storage-miner info
```



### 2.4 在扇区中存储任意数据

```
lotus-storage-miner store-garbage
```

回到[目录](#m目录)

### 2.5 Auth 

[同上](#1.2 Auth)

### 2.6 chain

[同上](#1.3 chain)

### 2.7 client

[同上](#1.4 client)

### 2.8 创建存储矿工

[同上](#1.5 创建存储矿工)

### 2.9 获取证明参数（同上）

### 3.0 mpool（同上）

### 3.1 net（同上）

### 3.2 paych（同上）

### 3.3 send转账（同上）

### 3.4 state（同上）

### 3.5 sync（同上）

### 3.6 获取版本（同上）

### 3.7 wallet（同上）

