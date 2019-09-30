## 目录
- filecoin源码协议层分析之hello握手协议
    -   1. [目的](#目的)
    -   2. [执行流程](执行流程)
    -   3. [源码信息](#源码信息)
    - 4. [源码分析](源码分析)
    
      - 4.1 [数据结构](#数据结构)
      - 4.2 [方法](#方法)

## 1.目的

- Hello协议负责TipSet的区块同步

## 2.执行流程

当我们执行go-filecoin daemon命令时，会执行go-filecoin/commands/daemon.go中的daemonRun（）函数

如下：

![](Z:\go\src\Demo\hello-protocal\pic\node-daemon1.png)

## 3.源码信息

- version
- package
- location

## 4.源码分析

### 4.1数据结构

- 获取协议名称

```go
// protocol is the libp2p protocol identifier for the hello protocol.
func helloProtocol(networkName string) protocol.ID {
	return protocol.ID(fmt.Sprintf("/fil/hello/%s", networkName))
}
```

- 日志标识

```go
var log = logging.Logger("/fil/hello")
```

- 定义hello协议中单个消息的数据结构
  - tipset切片（cid集合）
    - tipset概念：Filecoin的共识算法叫Expected Consensus，简称EC共识机制。Expected Consensus每一轮会生成一个Ticket，每个节点通过一定的计算，确定是否是该轮的Leader。如果选为Leader，节点可以打包区块。也就是说，每一轮可能没有Leader（所有节点都不符合Leader的条件），或者多个Leader（有多个节点符合Leader）。Filecoin使用TipSet的概念，表明一轮中多个Leader产生的指向同一个父亲区块的区块集合。
  - 最重的tipset（主链）的高度
  - 创世区块cid

```go
// Message is the data structure of a single message in the hello protocol.
type Message struct {
	HeaviestTipSetCids   types.TipSetKey
	HeaviestTipSetHeight uint64
	GenesisHash          cid.Cid
}
// TipSetKey is an immutable set of CIDs forming a unique key for a TipSet.
// Equal keys will have equivalent iteration order, but note that the CIDs are *not* maintained in
// the same order as the canonical iteration order of blocks in a tipset (which is by ticket).
// TipSetKey is a lightweight value type; passing by pointer is usually unnecessary.
type TipSetKey struct {
	// The slice is wrapped in a struct to enforce immutability.
	cids []cid.Cid
}
```

- Handler结构体,当有Node连接到自己时(1)会发送包含本节点信息的hello 消息给对方; (2) 对端会回复一个包含对端节点信息的消息体过来
  - host对应libp2p主机上的主机
  - 创世区块cid
  - 区块同步回调函数
  - 检索当前最重的tipset
  - 网络名称

```go
type Handler struct {
	host host.Host

	genesis cid.Cid

	// callBack is called when new peers tell us about their chain
	callBack helloCallback

	//  is used to retrieve the current heaviest tipset
	// for filling out our hello messages.
	getHeaviestTipSet getTipSetFunc

	networkName string
}
```

### 4.2 方法

- 流函数处理，响应远端节点的连接，回复hello消息体

```go
func (h *Handler) handleNewStream(s net.Stream) {
	defer s.Close() // nolint: errcheck
	if err := h.sendHello(s); err != nil {
		log.Debugf("failed to send hello message:%s", err)
	}
	return
}
// sendHello send a hello message on stream `s`.
func (h *Handler) sendHello(s net.Stream) error {
	msg, err := h.getOurHelloMessage()
	if err != nil {
		return err
	}
	return cbu.NewMsgWriter(s).WriteMsg(&msg)
}

func (h *Handler) getOurHelloMessage() (*Message, error) {
	heaviest, err := h.getHeaviestTipSet()
	if err != nil {
		return nil, err
	}
	height, err := heaviest.Height()
	if err != nil {
		return nil, err
	}

	return &Message{
		GenesisHash:          h.genesis,
		HeaviestTipSetCids:   heaviest.Key(),
		HeaviestTipSetHeight: height,
	}, nil
}
```

