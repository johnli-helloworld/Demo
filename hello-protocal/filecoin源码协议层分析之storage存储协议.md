## 目录

- 1. [协议概览图](#协议概览图)

- 2. [源码信息](#源码信息)

- 3. [源码分析](#源码分析)

  - 3.1 [存储客户](#存储客户)
    - 3.1.1 [数据结构](#3.1.1 数据结构)
    - 3.1.2 [函数](#3.1.2 函数)
    - 3.1.3 [client方法](#3.1.3 方法)
      - [ProposeDeal()：下单](ProposeDeal()：下单)
      - [QueryDeal()：查询订单状态](#QueryDeal()：查询订单状态)
  - 3.2 [存储矿工](#存储矿工)
    - 3.2.1 [数据结构](#3.2.1 数据结构)
    - 3.2.2 [函数](#3.2.1 函数)
    - 3.2.3 [miner方法](#3.2.1 方法)
      - [handleMakeDeal()：侦听交易请求](#handleMakeDeal())
      - [handleQueryDeal()：侦听交易查询请求](#handleQueryDeal())
      - [OnCommitmentSent()：](#OnCommitmentSent())
      - [OnNewHeaviestTipSet()：](#OnNewHeaviestTipSet())
    - 3.2.4 [dealsAwaitingSeal](#dealsAwaitingSeal)
      - [数据结构](#数据结构)
      - [函数](#函数)
      - [dealsAwaitingSeal方法](#)
        - [attachDealToSector():](#attachDealToSector())

## 1. 协议概览图



## 2. 源码信息

- version
- package
- location

## 3. 源码分析

### 3.1 存储客户

#### 3.1.1 数据结构

```go
// Client is used to make deals directly with storage miners.
type Client struct {
	api                 clientPorcelainAPI
    //对应libp2p上主机号
	host                host.Host
	log                 logging.EventLogger
    //使用给定的协议发起请求，等待响应
	ProtocolRequestFunc func(ctx context.Context, protocol protocol.ID, peer peer.ID, host host.Host, request interface{}, response interface{}) error
}
```

#### 3.1.2 函数

- NewClient()：新建客户端实例

```go
func NewClient(host host.Host, api clientPorcelainAPI) *Client {}
```

- MakeProtocolRequest()

```go
// MakeProtocolRequest makes a request and expects a response from the host using the given protocol.
func MakeProtocolRequest(ctx context.Context, protocol protocol.ID, peer peer.ID,
	host host.Host, request interface{}, response interface{}) error {
	s, err := host.NewStream(ctx, peer, protocol)
    ...
	if err := cbu.NewMsgWriter(s).WriteMsg(request);
	...
	if err := cbu.NewMsgReader(s).ReadMsg(response);
    ...
}
```

#### 3.1.3 方法

##### ProposeDeal（）：下单

- miner：存储矿工地址
- data：待存储文件cid
- askID：报价单id（存储矿工生成）
- duration：存储多久（新块大约每30秒生成一次，所以给定的时间应该是这样的表示为30秒间隔的计数。例如，1分钟就可以是2， 1小时是120， 1天是2880。）

- 流程：

  1. 获取矿工对应节点 pid

  2. 启动 goroutine, 测试能否连接上对方节点 
  3. 构造 Proposal 对象
  4. 创建支付渠道
  5. 创建订单请求 , 进行数据交换
  6. 校验持久化交易信息

```go
func (smc *Client) ProposeDeal(ctx context.Context, miner address.Address, data cid.Cid, askID uint64, duration uint64, allowDuplicates bool) (*storagedeal.SignedResponse, error) {
    //获取矿工对应节点 pid
	pid, err := smc.api.MinerGetPeerID(ctx, miner)
	if err != nil {
		return nil, err
	}
	//启动 goroutine, 测试能否连接上对方节点 ;
	minerAlive := make(chan error, 1)
	go func() {
		defer close(minerAlive)
		minerAlive <- smc.api.PingMinerWithTimeout(ctx, pid, 15*time.Second)
	}()
	//根据指定的cid获取文件大小
	pieceSize, err := smc.api.DAGGetFileSize(ctx, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine the size of the data")
	}
	//获取存储矿工的扇区大小
	sectorSize, err := smc.api.MinerGetSectorSize(ctx, miner)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sector size")
	}
	//可用的存储扇区大小，由于位填充，应小于sectorsize
	maxUserBytes := go_sectorbuilder.GetMaxUserBytesPerStagedSector(sectorSize.Uint64())
	//存储文件不可大于可用的存储扇区大小
    if pieceSize > maxUserBytes {
		return nil, fmt.Errorf("piece is %d bytes but sector size is %d bytes", pieceSize, maxUserBytes)
	}

	pieceReader, err := smc.api.DAGCat(ctx, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make piece reader")
	}
	//根据存储的文件生成客户对数据的承诺
	pieceCommitmentResponse, err := proofs.GeneratePieceCommitment(proofs.GeneratePieceCommitmentRequest{
		PieceReader: pieceReader,
		PieceSize:   types.NewBytesAmount(pieceSize),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate piece commitment")
	}
	//依据报价单获取单价每字节每块
	ask, err := smc.api.MinerGetAsk(ctx, miner, askID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ask price")
	}
	price := ask.Price
	//获取区块高度
	headKey := smc.api.ChainHeadKey()
	head, err := smc.api.ChainTipSet(headKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get head tipset: %s", headKey.String())
	}

	h, err := head.Height()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get height of tipset: %s", headKey.String())
	}
	chainHeight := types.NewBlockHeight(h)
	//默认钱包地址，支付地址
	fromAddress, err := smc.api.WalletDefaultAddress()
	if err != nil {
		return nil, err
	}
	
	minerOwner, err := smc.api.MinerGetOwnerAddress(ctx, miner)
	if err != nil {
		return nil, err
	}

	minerWorker, err := smc.api.MinerGetWorkerAddress(ctx, miner, headKey)
	if err != nil {
		return nil, err
	}
	//计算存储的总花费
	totalPrice := price.MulBigInt(big.NewInt(int64(pieceSize * duration)))
	//解析参数构造 Proposal 对象
	proposal := &storagedeal.Proposal{
		PieceRef:     data,
		Size:         types.NewBytesAmount(pieceSize),
		TotalPrice:   totalPrice,
		Duration:     duration,
		MinerAddress: miner,
	}
	//是否允许重复下单
	if smc.isMaybeDupDeal(ctx, proposal) && !allowDuplicates {
		return nil, Errors[ErrDuplicateDeal]
	}

	// see if we managed to connect to the miner
	err = <-minerAlive
	if err == net.ErrPingSelf {
		return nil, errors.New("attempting to make storage deal with self. This is currently unsupported.  Please use a separate go-filecoin node as client")
	} else if err != nil {
		return nil, err
	}

	// Always set payer because it is used for signing
	proposal.Payment.Payer = fromAddress

	totalCost := price.MulBigInt(big.NewInt(int64(pieceSize * duration)))
    // 创建支付渠道（若totoalcost为0表示免费存储，则不需要创建支付渠道）
	if totalCost.GreaterThan(types.ZeroAttoFIL) {
		cpResp, err := smc.api.CreatePayments(ctx, porcelain.CreatePaymentsParams{
			From:            fromAddress,
			To:              minerOwner,
			Value:           totalCost,
			Duration:        duration,
			MinerAddress:    miner,
			CommP:           pieceCommitmentResponse.CommP,
			PaymentInterval: VoucherInterval,
			PieceSize:       types.NewBytesAmount(pieceSize),
			ChannelExpiry:   *chainHeight.Add(types.NewBlockHeight(duration + ChannelExpiryInterval)),
			GasPrice:        types.NewAttoFIL(big.NewInt(CreateChannelGasPrice)),
			GasLimit:        types.NewGasUnits(CreateChannelGasLimit),
		})
		if err != nil {
			return nil, errors.Wrap(err, "error creating payment")
		}

		proposal.Payment.Channel = cpResp.Channel
		proposal.Payment.PayChActor = address.PaymentBrokerAddress
		proposal.Payment.ChannelMsgCid = &cpResp.ChannelMsgCid
		proposal.Payment.Vouchers = cpResp.Vouchers
	}

	signedProposal, err := proposal.NewSignedProposal(fromAddress, smc.api)
	if err != nil {
		return nil, err
	}

	var response storagedeal.SignedResponse
    // 向存储矿工发送订单请求(MakeProtocolRequest)
	err = smc.ProtocolRequestFunc(ctx, makeDealProtocol, pid, smc.host, signedProposal, &response)
	if err != nil {
		return nil, errors.Wrap(err, "error sending proposal")
	}

	if err := smc.checkDealResponse(ctx, &response, minerWorker); err != nil {
		return nil, errors.Wrap(err, "response check failed")
	}

	// Note: currently the miner requests the data out of band

	if err := smc.recordResponse(ctx, &response, miner, signedProposal, pieceCommitmentResponse.CommP); err != nil {
		return nil, errors.Wrap(err, "failed to track response")
	}
	smc.log.Debugf("proposed deal for: %s, %v\n", miner.String(), proposal)

	return &response, nil
}
```

- (smc *Client) checkDealResponse()：对返回的消息进行签名校验，并解析订单的状态

```go
func (smc *Client) checkDealResponse(ctx context.Context, resp *storagedeal.SignedResponse, workerAddr address.Address) error {
    valid, err := resp.VerifySignature(workerAddr)
	if err != nil {
		return errors.Wrap(err, "Could not verify response signature")
	}

	if !valid {
		return errors.New("Response signature is invalid")
	}

	switch resp.State {
	case storagedeal.Rejected:
		return fmt.Errorf("deal rejected: %s", resp.Message)
	case storagedeal.Failed:
		return fmt.Errorf("deal failed: %s", resp.Message)
	case storagedeal.Accepted:
		return nil
	default:
		return fmt.Errorf("invalid proposal response: %s", resp.State)
	}
}
```

- (smc *Client) recordResponse()：持久化交易信息到磁盘

```go
func (smc *Client) recordResponse(ctx context.Context, resp *storagedeal.SignedResponse, miner address.Address, p *storagedeal.SignedProposal, commP types.CommP) error {
    	proposalCid, err := convert.ToCid(p)
	if err != nil {
		return errors.New("failed to get cid of proposal")
	}
	if !proposalCid.Equals(resp.ProposalCid) {
		return fmt.Errorf("cids not equal %s %s", proposalCid, resp.ProposalCid)
	}
	_, err = smc.api.DealGet(ctx, proposalCid)
	if err == nil {
		return fmt.Errorf("deal [%s] is already in progress", proposalCid.String())
	}
	if err != porcelain.ErrDealNotFound {
		return errors.Wrapf(err, "failed to check for existing deal: %s", proposalCid.String())
	}

	return smc.api.DealPut(&storagedeal.Deal{
		Miner:    miner,
		Proposal: p,
		Response: resp,
		CommP:    commP,
	})
}
```

##### QueryDeal（）：查询订单状态

```go
func (smc *Client) QueryDeal(ctx context.Context, proposalCid cid.Cid) (*storagedeal.SignedResponse, error) {
	mineraddr, err := smc.minerForProposal(ctx, proposalCid)
	if err != nil {
		return nil, err
	}

	workerAddr, err := smc.api.MinerGetWorkerAddress(ctx, mineraddr, smc.api.ChainHeadKey())
	if err != nil {
		return nil, err
	}

	minerpid, err := smc.api.MinerGetPeerID(ctx, mineraddr)
	if err != nil {
		return nil, err
	}

	q := storagedeal.QueryRequest{Cid: proposalCid}
	var resp storagedeal.SignedResponse
    //发起查询订单请求
	err = smc.ProtocolRequestFunc(ctx, queryDealProtocol, minerpid, smc.host, q, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "error querying deal")
	}
	//签名校验
	valid, err := resp.VerifySignature(workerAddr)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errors.New("deal response has invalid signature")
	}

	return &resp, nil
}
```

- (smc *Client) minerForProposal()：根据proposalid获取交易订单信息，拿到存储矿工地址

```go
func (smc *Client) minerForProposal(ctx context.Context, c cid.Cid) (address.Address, error) {
	storageDeal, err := smc.api.DealGet(ctx, c)
	if err != nil {
		return address.Undef, errors.Wrapf(err, "failed to fetch deal: %s", c)
	}
	return storageDeal.Miner, nil
}
```



### 3.2 存储矿工

#### 3.2.1 数据结构

- package : storage
- location : miner.go

```go
type Miner struct {
	minerAddr address.Address
	ownerAddr address.Address
	//本地持久化存储所有等待密封的deal信息，通过此接口操作
	dealsAwaitingSealDs repo.Datastore

	postInProcessLk sync.Mutex
	postInProcess   *types.BlockHeight
	
	dealsAwaitingSeal *dealsAwaitingSeal

	prover     prover
	sectorSize *types.BytesAmount

	porcelainAPI minerPorcelain
	node         node

	proposalProcessor func(context.Context, *Miner, cid.Cid)
}

```

- package : storage
- location : deals_awaiting_seal.go

```go
type dealsAwaitingSeal struct {
	l sync.Mutex
    // 记录此扇区有哪些deal  key:sectorid ; value:proposalcid  
	SectorsToDeals map[uint64][]cid.Cid
    // 扇区密封相关信息 key:sectorid ; value:sectorInfo
	SealedSectors map[uint64]*sectorInfo

	// onSuccess will be called only after the sector has been successfully sealed
	onSuccess func(ctx context.Context, dealCid cid.Cid, sector *sectorbuilder.SealedSectorMetadata)

	// onFail will be called if an error occurs during sealing or commitment
	onFail func(ctx context.Context, dealCid cid.Cid, message string)
}

type sectorInfo struct {
	// Metadata contains information about the sealed sector needed to verify the seal
	Metadata *sectorbuilder.SealedSectorMetadata

	// CommitMessageCid is the cid of the commitSector message sent for sealed sector. It allows the client to coordinate on timing.
	CommitMessageCid cid.Cid

	// Succeeded indicates whether sealing was and committing was successful
	Succeeded bool

	// ErrorMessage indicate what went wrong if sealing or committing was not successful
	ErrorMessage string
}
```



#### 3.2.2 函数

- NewMiner()：创建miner实例

```go
func NewMiner(minerAddr, ownerAddr address.Address, prover prover, sectorSize *types.BytesAmount, nd node, dealsDs repo.Datastore, porcelainAPI minerPorcelain) (*Miner, error) {
	sm := &Miner{
		minerAddr:           minerAddr,
		ownerAddr:           ownerAddr,
		porcelainAPI:        porcelainAPI,
		dealsAwaitingSealDs: dealsDs,
		prover:              prover,
		sectorSize:          sectorSize,
		node:                nd,
		proposalProcessor:   processStorageDeal,
	}

	if err := sm.loadDealsAwaitingSeal(); err != nil {
		return nil, errors.Wrap(err, "failed to load dealAwaitingSeal when creating miner")
	}
	sm.dealsAwaitingSeal.onSuccess = sm.onCommitSuccess
	sm.dealsAwaitingSeal.onFail = sm.onCommitFail
	//makeDealProtocol协议流处理回调函数
	nd.Host().SetStreamHandler(makeDealProtocol, sm.handleMakeDeal)
    //queryDealProtocol协议流处理回调函数
	nd.Host().SetStreamHandler(queryDealProtocol, sm.handleQueryDeal)

	return sm, nil
}
```

- ValidatePaymentVoucherCondition()：验证为存储付款创建的凭证满足预期的条件

```go
func ValidatePaymentVoucherCondition(ctx context.Context, condition *types.Predicate, minerAddr address.Address, commP types.CommP, pieceSize *types.BytesAmount) error {
	// a nil condition is always valid
	if condition == nil {
		return nil
	}

	if condition.To != minerAddr {
		return errors.Errorf("voucher condition addressed to %s, should be %s", condition.To, minerAddr)
	}

	if condition.Method != verifyPieceInclusionMethod {
		return errors.Errorf("payment condition method, %s, should be %s", condition.Method, verifyPieceInclusionMethod)
	}

	if condition.Params == nil || len(condition.Params) != 2 {
		return errors.New("payment condition does not contain exactly 2 parameters")
	}

	var clientCommP types.CommP
	clientCommPBytes, ok := condition.Params[0].([]byte)
	if ok {
		copy(clientCommP[:], clientCommPBytes)
	} else {
		return errors.New("piece commitment is not a CommP")
	}

	if clientCommP != commP {
		return errors.Errorf("piece commitment, [% x] does not match payment condition commitment: [% x]", clientCommP[:], commP[:])
	}

	var clientPieceSize *types.BytesAmount
	clientPieceSizeBytes, ok := condition.Params[1].([]byte)
	if ok {
		clientPieceSize = types.NewBytesAmountFromBytes(clientPieceSizeBytes)
	} else {
		return errors.New("piece size is not a bytes amount")
	}

	if !pieceSize.Equal(clientPieceSize) {
		return errors.Errorf("piece size, %v,  does not match piece size supplied in payment condition: %v", pieceSize, clientPieceSize)
	}

	return nil
}
```



#### 3.2.3 方法

##### handleMakeDeal()

- (sm *Miner)handleMakeDeal()：侦听交易请求

```go
func (sm *Miner) handleMakeDeal(s inet.Stream) {
    ...
    //接收client的消息
    if err := cbu.NewMsgReader(s).ReadMsg(&signedProposal);
    ...
    //调用receiveStorageProposal方法对接收的消息处理
    resp, err := sm.receiveStorageProposal(ctx, &signedProposal)
}
```

- (sm *Miner)receiveStorageProposal()：

```go
// receiveStorageProposal is the entry point for the miner storage protocol
func (sm *Miner) receiveStorageProposal(ctx context.Context, sp *storagedeal.SignedProposal) (*storagedeal.SignedResponse, error) {
	// Validate deal signature
	bdp, err := sp.Proposal.Marshal()
	if err != nil {
		return nil, err
	}
    
	if !types.IsValidSignature(bdp, sp.Payment.Payer, sp.Signature) {
		return sm.rejectProposal(ctx, sp, fmt.Sprint("invalid deal signature"))
	}

	// compute expected total price for deal (storage price * duration * bytes)
	price, err := sm.getStoragePrice()
	if err != nil {
		return sm.rejectProposal(ctx, sp, err.Error())
	}

	// 跳过支付验证，若矿工不收取任何费用
	if price.GreaterThan(types.ZeroAttoFIL) {
		if err := sm.validateDealPayment(ctx, sp, price); err != nil {
			return sm.rejectProposal(ctx, sp, err.Error())
		}
	}
	//文件大小不可大于扇区大小
	maxUserBytes := types.NewBytesAmount(go_sectorbuilder.GetMaxUserBytesPerStagedSector(sm.sectorSize.Uint64()))
	if sp.Size.GreaterThan(maxUserBytes) {
		return sm.rejectProposal(ctx, sp, fmt.Sprintf("piece is %s bytes but sector size is %s bytes", sp.Size.String(), maxUserBytes))
	}

	// Payment is valid, everything else checks out, let's accept this proposal
	return sm.acceptProposal(ctx, sp)
}
```

- validateDealPayment()：支付验证

```go
func (sm *Miner) validateDealPayment(ctx context.Context, p *storagedeal.SignedProposal, price types.AttoFIL) error {
	if p.Size == nil {
		return fmt.Errorf("proposed deal has no size")
	}

	durationBigInt := big.NewInt(0).SetUint64(p.Duration)
	priceBigInt := big.NewInt(0).SetUint64(p.Size.Uint64())
	expectedPrice := price.MulBigInt(durationBigInt).MulBigInt(priceBigInt)
	if p.TotalPrice.LessThan(expectedPrice) {
		return fmt.Errorf("proposed price (%s) is less than expected (%s) given asking price of %s", p.TotalPrice.String(), expectedPrice.String(), price.String())
	}

	// get channel
	channel, err := sm.getPaymentChannel(ctx, p)
	if err != nil {
		return err
	}

	// confirm we are target of channel
	if channel.Target != sm.ownerAddr {
		return fmt.Errorf("miner account (%s) is not target of payment channel (%s)", sm.ownerAddr.String(), channel.Target.String())
	}

	// confirm channel contains enough funds
	if channel.Amount.LessThan(expectedPrice) {
		return fmt.Errorf("payment channel does not contain enough funds (%s < %s)", channel.Amount.String(), expectedPrice.String())
	}

	// start with current block height
	head, err := sm.porcelainAPI.ChainTipSet(sm.porcelainAPI.ChainHeadKey())
	if err != nil {
		return fmt.Errorf("could not access head tipset")
	}
	h, err := head.Height()
	if err != nil {
		return fmt.Errorf("could not get current block height")
	}
	blockHeight := types.NewBlockHeight(h)

	// require at least one payment
	if len(p.Payment.Vouchers) < 1 {
		return errors.New("deal proposal contains no payment vouchers")
	}

	// first payment must be before blockHeight + VoucherInterval
	expectedFirstPayment := blockHeight.Add(types.NewBlockHeight(VoucherInterval))
	firstPayment := p.Payment.Vouchers[0].ValidAt
	if firstPayment.GreaterThan(expectedFirstPayment) {
		return errors.New("payments start after deal start interval")
	}

	lastValidAt := expectedFirstPayment
	for _, v := range p.Payment.Vouchers {
		// confirm signature is valid against expected actor and channel id
		if !paymentbroker.VerifyVoucherSignature(p.Payment.Payer, p.Payment.Channel, v.Amount, &v.ValidAt, v.Condition, v.Signature) {
			return errors.New("invalid signature in voucher")
		}

		// make sure voucher validAt is not spaced to far apart
		expectedValidAt := lastValidAt.Add(types.NewBlockHeight(VoucherInterval))
		if v.ValidAt.GreaterThan(expectedValidAt) {
			return fmt.Errorf("interval between vouchers too high (%s - %s > %d)", v.ValidAt.String(), lastValidAt.String(), VoucherInterval)
		}

		// confirm voucher amounts increase linearly
		// We want the ratio of voucher amount / (valid at - expected start) >= total price / duration
		// this is implied by amount*duration >= total price*(valid at - expected start).
		lhs := v.Amount.MulBigInt(big.NewInt(int64(p.Duration)))
		rhs := p.TotalPrice.MulBigInt(v.ValidAt.Sub(blockHeight).AsBigInt())
		if lhs.LessThan(rhs) {
			return fmt.Errorf("voucher amount (%s) less than expected for voucher valid at (%s)", v.Amount.String(), v.ValidAt.String())
		}

		lastValidAt = &v.ValidAt
	}

	// confirm last voucher value is for full amount
	lastVoucher := p.Payment.Vouchers[len(p.Payment.Vouchers)-1]
	if lastVoucher.Amount.LessThan(p.TotalPrice) {
		return fmt.Errorf("last payment (%s) does not cover total price (%s)", lastVoucher.Amount.String(), p.TotalPrice.String())
	}

	// require channel expires at or after last voucher + ChannelExpiryInterval
	expectedEol := lastVoucher.ValidAt.Add(types.NewBlockHeight(ChannelExpiryInterval))
	if channel.Eol.LessThan(expectedEol) {
		return fmt.Errorf("payment channel eol (%s) less than required eol (%s)", channel.Eol, expectedEol)
	}

	return nil
}
```

- acceptProposal（），更改订单状态为已接受

```go
func (sm *Miner) acceptProposal(ctx context.Context, p *storagedeal.SignedProposal) (*storagedeal.SignedResponse, error) {
	if sm.porcelainAPI.SectorBuilder() == nil {
		return nil, errors.New("Mining disabled, can not process proposal")
	}
	//用client传过来的SignedProposal生成的proposalCid
	proposalCid, err := convert.ToCid(p)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cid of proposal")
	}
	//改变订单的状态为accept
	resp := storagedeal.Response{State: storagedeal.Accepted, ProposalCid: proposalCid}
	//对响应签名
    signed, err := sm.signResponse(ctx, resp)

	if err != nil {
		return nil, errors.Wrap(err, "could not sign deal response")
	}

	storageDeal := &storagedeal.Deal{
		Miner:    sm.minerAddr,
		Proposal: p,
		Response: signed,
	}
	//持久化Deal信息到本地
	if err := sm.porcelainAPI.DealPut(storageDeal); err != nil {
		return nil, errors.Wrap(err, "Could not persist miner deal")
	}

	// TODO: use some sort of nicer scheduler
    //其协程执行processStorageDeal()
	go sm.proposalProcessor(ctx, sm, proposalCid)

	return signed, nil
}
```

- processStorageDeal( ):

```go
func processStorageDeal(ctx context.Context, sm *Miner, proposalCid cid.Cid) {
	log.Debugf("Miner.processStorageDeal(%s)", proposalCid.String())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	//获取存储订单信息
	d, err := sm.porcelainAPI.DealGet(ctx, proposalCid)
	if err != nil {
		log.Errorf("could not retrieve deal with proposal CID %s: %s", proposalCid.String(), err)
	}
    //若deal状态不是accept则表明已经开始密封
	if d.Response.State != storagedeal.Accepted {
		// TODO: handle resumption of deal processing across miner restarts
		log.Error("attempted to process an already started deal")
		return
	}

	// 'Receive' the data, this could also be a truck full of hard drives. (TODO: proper abstraction)
	// TODO: this is not a great way to do this. At least use a session
	// Also, this needs to be fetched into a staging area for miners to prepare and seal in data
	log.Debug("Miner.processStorageDeal - FetchGraph")
    //通过cid去获取file data
	if err := dag.FetchGraph(ctx, d.Proposal.PieceRef, dag.NewDAGService(sm.node.BlockService())); err != nil {
		log.Errorf("failed to fetch data: %s", err)
		err := sm.updateDealResponse(ctx, proposalCid, func(resp *storagedeal.Response) {
			resp.Message = "Transfer failed"
			resp.State = storagedeal.Failed
		})
		if err != nil {
			log.Errorf("could not update to deal to 'Failed' state: %s", err)
		}
		return
	}

	fail := func(message, logerr string) {
		log.Errorf(logerr)
		err := sm.updateDealResponse(ctx, proposalCid, func(resp *storagedeal.Response) {
			resp.Message = message
			resp.State = storagedeal.Failed
		})
		if err != nil {
			log.Errorf("could not update to deal to 'Failed' state in fail callback: %s", err)
		}
	}

	dagService := dag.NewDAGService(sm.node.BlockService())

	rootIpldNode, err := dagService.Get(ctx, d.Proposal.PieceRef)
	if err != nil {
		fail("internal error", fmt.Sprintf("failed to add piece: %s", err))
		return
	}

	// Before adding piece, confirm that client has generated payment conditions correctly now that
	// we can compute CommP
    //校验client已经生成了支付条件
	if err := sm.validatePieceCommitments(ctx, d, rootIpldNode, dagService); err != nil {
		fail("payment error", fmt.Sprintf("failed to add piece: %s", err))
		return
	}

	r, err := uio.NewDagReader(ctx, rootIpldNode, dagService)
	if err != nil {
		fail("internal error", fmt.Sprintf("failed to add piece: %s", err))
		return
	}

	// There is a race here that requires us to use dealsAwaitingSeal below. If the
	// sector gets sealed and OnCommitmentSent is called right after
	// AddPiece returns but before we record the sector/deal mapping we might
	// miss it. Hence, dealsAwaitingSeal. I'm told that sealing in practice is
	// so slow that the race only exists in tests, but tests were flaky so
	// we fixed it with dealsAwaitingSeal.
	//
	// Also, this pattern of not being able to set up book-keeping ahead of
	// the call is inelegant.
    //调用未密封的扇区密封数据
	sectorID, err := sm.porcelainAPI.SectorBuilder().AddPiece(ctx, d.Proposal.PieceRef, d.Proposal.Size.Uint64(), r)
	if err != nil {
		fail("failed to submit seal proof", fmt.Sprintf("failed to add piece: %s", err))
		return
	}
	//更新状态
	err = sm.updateDealResponse(ctx, proposalCid, func(resp *storagedeal.Response) {
		resp.State = storagedeal.Staged
	})
	if err != nil {
		log.Errorf("could update to 'Staged': %s", err)
	}

	// Careful: this might update state to success or failure so it should go after
	// updating state to Staged.
    //更新状态有可能失败,继续执行将状态更新到暂存
	sm.dealsAwaitingSeal.attachDealToSector(ctx, sectorID, proposalCid)
	if err := sm.saveDealsAwaitingSeal(); err != nil {
		log.Errorf("could not save deal awaiting seal: %s", err)
	}
}
```

- (sm *Miner)validatePieceCommitments()：校验client生成的数据承诺

```go
func (sm *Miner) validatePieceCommitments(ctx context.Context, deal *storagedeal.Deal, rootIpldNode format.Node, serv format.NodeGetter) error {
	pieceReader, err := uio.NewDagReader(ctx, rootIpldNode, serv)
	if err != nil {
		return err
	}

	// Generating the piece commitment is a computationally expensive operation and can take
	// many minutes depending on the size of the piece.
	pieceCommitmentResponse, err := proofs.GeneratePieceCommitment(proofs.GeneratePieceCommitmentRequest{
		PieceReader: pieceReader,
		PieceSize:   types.NewBytesAmount(deal.Proposal.Size.Uint64()),
	})
	if err != nil {
		return errors.Wrap(err, "failed to generate pieceCommitmentResponse commitment")
	}

	for _, voucher := range deal.Proposal.Payment.Vouchers {
        //ValidatePaymentVoucherCondition()验证为存储付款创建的凭证满足预期的条件【client.go中生成了Vouchers凭证 []PaymentVoucher】
		err := porcelain.ValidatePaymentVoucherCondition(ctx, voucher.Condition, sm.minerAddr, pieceCommitmentResponse.CommP, deal.Proposal.Size)
		if err != nil {
			return err
		}
	}

	return nil
}
```

- location: porcelain/Payments.go

```go
type PaymentVoucher struct {
	// Channel is the id of this voucher's payment channel.
	Channel ChannelID `json:"channel"`

	// Payer is the address of the account that created the channel.
	Payer address.Address `json:"payer"`

	// Target is the address of the account that will receive funds from the channel.
	Target address.Address `json:"target"`

	//每阶段可赎回资金
	Amount AttoFIL `json:"amount"`

	// 有效期
	ValidAt BlockHeight `json:"valid_at"`

	// 创建存储凭证需满足的预期条件
	Condition *Predicate `json:"condition"`

	// 签名
	Signature Signature `json:"signature"`
}
```

##### handleQueryDeal()

- (sm *Miner)handleQueryDeal()：侦听查询交易请求

```go
func (sm *Miner) handleQueryDeal(s inet.Stream) {
	defer s.Close() // nolint: errcheck

	ctx := context.Background()

	var q storagedeal.QueryRequest
	if err := cbu.NewMsgReader(s).ReadMsg(&q); err != nil {
		log.Errorf("received invalid query: %s", err)
		return
	}

	resp := sm.Query(ctx, q.Cid)

	if err := cbu.NewMsgWriter(s).WriteMsg(resp); err != nil {
		log.Errorf("failed to write query response: %s", err)
	}
}
```

- (sm *Miner)Query()：从本地存储的deal信息里查询
  - c：proposalcid

```go
func (sm *Miner) Query(ctx context.Context, c cid.Cid) *storagedeal.SignedResponse {
	deal, err := sm.porcelainAPI.DealGet(ctx, c)
	if err != nil {
		return &storagedeal.SignedResponse{
			Response: storagedeal.Response{
				State:   storagedeal.Unknown,
				Message: "no such deal",
			},
		}
	}

	return deal.Response
}
```

##### OnCommitmentSent()

- (sm *Miner)OnCommitmentSent()：侦听密封结果，提交复制证明时回调，在node/node.go中的StartMining()中调用

```go
// OnCommitmentSent is a callback, called when a sector seal message was posted to the chain.
func (sm *Miner) OnCommitmentSent(sector *sectorbuilder.SealedSectorMetadata, msgCid cid.Cid, err error) {
	ctx := context.Background()
	sectorID := sector.SectorID
	log.Debug("Miner.OnCommitmentSent")

	if err != nil {
		log.Errorf("failed sealing sector: %d: %s:", sectorID, err)
		errMsg := fmt.Sprintf("failed sealing sector: %d", sectorID)
		sm.dealsAwaitingSeal.onSealFail(ctx, sector.SectorID, errMsg)
	} else {
		sm.dealsAwaitingSeal.onSealSuccess(ctx, sector, msgCid)
	}
	if err := sm.saveDealsAwaitingSeal(); err != nil {
		log.Errorf("failed persisting deals awaiting seal: %s", err)
		sm.dealsAwaitingSeal.onSealFail(ctx, sector.SectorID, "failed persisting deals awaiting seal")
	}
}
```

- (sm *Miner)onCommitSuccess()：提交复制证明成功(扇区密封成功)时调用方法,更新本地的存储的deal信息体

```go
func (sm *Miner) onCommitSuccess(ctx context.Context, dealCid cid.Cid, sector *sectorbuilder.SealedSectorMetadata) {
	pieceInfo, err := sm.findPieceInfo(ctx, dealCid, sector)
	if err != nil {
		// log error, but continue to update deal with the information we have
		log.Errorf("commit succeeded, but could not find piece info %s", err)
	}

	// failure to locate commitmentMessage should not block update
	commitMessageCid, ok := sm.dealsAwaitingSeal.commitMessageCid(sector.SectorID)
	if !ok {
		log.Errorf("commit succeeded, but could not find commit message cid.")
	}

	// update response
	err = sm.updateDealResponse(ctx, dealCid, func(resp *storagedeal.Response) {
		resp.State = storagedeal.Complete
		resp.ProofInfo = &storagedeal.ProofInfo{
			SectorID:          sector.SectorID,
			CommitmentMessage: commitMessageCid,
			CommD:             sector.CommD[:],
			CommR:             sector.CommR[:],
			CommRStar:         sector.CommRStar[:],
		}
		if pieceInfo != nil {
			resp.ProofInfo.PieceInclusionProof = pieceInfo.InclusionProof
		}
	})
	if err != nil {
		log.Errorf("commit succeeded but could not update to deal 'Complete' state: %s", err)
	}
}
```

- (sm *Miner)onCommitFail()：提交复制证明失败(扇区密封失败)时调用方法,更新本地的存储的deal信息体

```go
func (sm *Miner) onCommitFail(ctx context.Context, dealCid cid.Cid, message string) {
	err := sm.updateDealResponse(ctx, dealCid, func(resp *storagedeal.Response) {
		resp.Message = message
		resp.State = storagedeal.Failed
	})
	log.Errorf("commit failure but could not update to deal 'Failed' state: %s", err)
}
```

##### OnNewHeaviestTipSet()

- (sm *Miner)OnNewHeaviestTipSet()：每当有新块产生时的时空证明回调，在node/node.go的node.Start()方法中另起协程go node.handleNewChainHeads()调用

```go
func (sm *Miner) OnNewHeaviestTipSet(ts types.TipSet) (*moresync.Latch, error) {
	ctx := context.Background()
	doneLatch := moresync.NewLatch(0)

	isBootstrapMinerActor, err := sm.isBootstrapMinerActor(ctx)
	if err != nil {
		return doneLatch, errors.Errorf("could not determine if actor created for bootstrapping: %s", err)
	}

	if isBootstrapMinerActor {
		// this is not an error condition, so log quietly
		log.Info("bootstrap miner actor skips PoSt-generation flow")
		return doneLatch, nil
	}

	commitments, err := sm.getActorSectorCommitments(ctx)
	if err != nil {
		return doneLatch, errors.Errorf("failed to get miner actor commitments: %s", err)
	}

	// get ProvingSet
	// iterate through ProvingSetValues pulling commitment from commitments

	var inputs []PoStInputs
	for k, v := range commitments {
		n, err := strconv.ParseUint(k, 10, 64)
		if err != nil {
			return doneLatch, errors.Errorf("failed to parse commitment sector id to uint64: %s", err)
		}

		inputs = append(inputs, PoStInputs{
			CommD:     v.CommD,
			CommR:     v.CommR,
			CommRStar: v.CommRStar,
			SectorID:  n,
		})
	}

	if len(inputs) == 0 {
		// no sector sealed, nothing to do
		return doneLatch, nil
	}

	provingWindowStart, provingWindowEnd, err := sm.getProvingWindow()
	if err != nil {
		return doneLatch, errors.Errorf("failed to get proving period: %s", err)
	}

	sm.postInProcessLk.Lock()
	defer sm.postInProcessLk.Unlock()

	if sm.postInProcess != nil && sm.postInProcess.Equal(provingWindowEnd) {
		// post is already being generated for this period, nothing to do
		return doneLatch, nil
	}

	height, err := ts.Height()
	if err != nil {
		return doneLatch, errors.Errorf("failed to get block height: %s", err)
	}

	// the block height of the new heaviest tipset
	h := types.NewBlockHeight(height)

	if h.GreaterEqual(provingWindowStart.Add(types.NewBlockHeight(challengeDelayRounds))) {
		if h.LessThan(provingWindowEnd) {
			// we are in a new proving period, lets get this post going
			sm.postInProcess = provingWindowEnd
			postLatch := moresync.NewLatch(1)
			go func() {
				sm.submitPoSt(ctx, provingWindowStart, provingWindowEnd, inputs)
				postLatch.Done()
			}()
			return postLatch, nil
		}
		// we are too late
		// TODO: figure out faults and payments here #3406
		return doneLatch, errors.Errorf("too late start=%s  end=%s current=%s", provingWindowStart, provingWindowEnd, h)
	}

	return doneLatch, nil
}
```

- (sm *Miner)getActorSectorCommitments() : 

```go
func (sm *Miner) getActorSectorCommitments(ctx context.Context) (map[string]types.Commitments, error) {
	returnValues, err := sm.porcelainAPI.MessageQuery(
		ctx,
		address.Undef,
		sm.minerAddr,
		"getProvingSetCommitments",
		sm.porcelainAPI.ChainHeadKey(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "query method failed")
	}
	sig, err := sm.porcelainAPI.ActorGetSignature(ctx, sm.minerAddr, "getProvingSetCommitments")
	if err != nil {
		return nil, errors.Wrap(err, "query method failed")
	}

	commitmentsVal, err := abi.Deserialize(returnValues[0], sig.Return[0])
	if err != nil {
		return nil, errors.Wrap(err, "deserialization failed")
	}

	commitments, ok := commitmentsVal.Val.(map[string]types.Commitments)
	if !ok {
		return nil, errors.Wrap(err, "type assertion failed")
	}

	return commitments, nil
}
```



#### 3.2.4 dealsAwaitingSeal

##### 数据结构

```go
type dealsAwaitingSeal struct {
	l sync.Mutex
    // 记录sector存了哪些deal. key:sectorid; value:[]proposalcid
	SectorsToDeals map[uint64][]cid.Cid
	// Maps from sector id to information about sector seal.
	SealedSectors map[uint64]*sectorInfo

	// 密封成功调用
	onSuccess func(ctx context.Context, dealCid cid.Cid, sector *sectorbuilder.SealedSectorMetadata)

	// 密封或者证明过程中失败被调用
	onFail func(ctx context.Context, dealCid cid.Cid, message string)
}

type sectorInfo struct {
	// Metadata contains information about the sealed sector needed to verify the seal
	Metadata *sectorbuilder.SealedSectorMetadata

	// CommitMessageCid is the cid of the commitSector message sent for sealed sector. It allows the client to coordinate on timing.
	CommitMessageCid cid.Cid

	// Succeeded indicates whether sealing was and committing was successful
	Succeeded bool

	// ErrorMessage indicate what went wrong if sealing or committing was not successful
	ErrorMessage string
}
```

##### 函数

- newDealsAwaitingSeal()

```go
func newDealsAwaitingSeal() *dealsAwaitingSeal {
	return &dealsAwaitingSeal{
		SectorsToDeals: make(map[uint64][]cid.Cid),
		SealedSectors:  make(map[uint64]*sectorInfo),
	}
}
```

##### 方法

###### attachDealToSector()

- attachDealToSector()：attachDealToSector检查密封扇区的列表，查看一个扇区是否已被密封。如果这个扇区的密封完成，onSuccess或onFailure将被立即调用，否则，将其添加到扇区stodeals中，以便我们可以在密封完成时响应。

```go
func (dealsAwaitingSeal *dealsAwaitingSeal) attachDealToSector(ctx context.Context, sectorID uint64, dealCid cid.Cid) {
	dealsAwaitingSeal.l.Lock()
	defer dealsAwaitingSeal.l.Unlock()

	sector, ok := dealsAwaitingSeal.SealedSectors[sectorID]

	// if sector sealing hasn't succeed or failed yet, just add to SectorToDeals and exit
	if !ok {
		deals, ok := dealsAwaitingSeal.SectorsToDeals[sectorID]
		if ok {
			dealsAwaitingSeal.SectorsToDeals[sectorID] = append(deals, dealCid)
		} else {
			dealsAwaitingSeal.SectorsToDeals[sectorID] = []cid.Cid{dealCid}
		}
		return
	}

	// We have sealing information, so process deal with sector data immediately
	if sector.Succeeded {
		dealsAwaitingSeal.onSuccess(ctx, dealCid, sector.Metadata)
	} else {
		dealsAwaitingSeal.onFail(ctx, dealCid, sector.ErrorMessage)
	}

	// Don't keep references to sectors around forever. Assume that at most
	// one onSealSuccess-before-attachDealToSector call will happen (eg, in a test). Sector sealing
	// outside of tests is so slow that it shouldn't happen in practice.
	// So now that it has happened once, clean it up. If we wanted to keep
	// the state around for longer for some reason we need to limit how many
	// sectors we hang onto, eg keep a fixed-length slice of successes
	// and failures and shift the oldest off and the newest on.
	delete(dealsAwaitingSeal.SealedSectors, sectorID)
}
```

