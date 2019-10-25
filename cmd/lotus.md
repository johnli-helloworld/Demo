### 1.流程概述

### 2.源码信息

### 3.源码解析

#### 3.1 LoTus 

##### 3.1.1 daemon

```go
// 构建lotus daemon 启动cli命令行
var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "Start a lotus daemon process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "api",
			Value: "1234",
		},
		&cli.StringFlag{
			Name:   makeGenFlag,
			Value:  "",
			Hidden: true,
		},
		&cli.StringFlag{
			Name:  "genesis",
			Usage: "genesis file to use for first node run",
		},
		&cli.BoolFlag{
			Name:  "bootstrap",
			Value: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
        //获取FsRepo实例对象
		r, err := repo.NewFS(cctx.String("repo"))
		if err != nil {
			return err
		}
		//初始化repo资源目录
		if err := r.Init(); err != nil && err != repo.ErrRepoExists {
			return err
		}
		//proof-params.json文件参数校验
		if err := build.GetParams(false); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}
		//当前的lotus构建将使用build目录中的genesis和bootstrap文件自动加入lotus Devnet.不需要配置.
        //这里预留了genesis参数，为了以后的扩展
		genBytes := build.MaybeGenesis()

		if cctx.String("genesis") != "" {
			genBytes, err = ioutil.ReadFile(cctx.String("genesis"))
			if err != nil {
				return err
			}

		}
		
		genesis := node.Options()
		if len(genBytes) > 0 {
			genesis = node.Override(new(modules.Genesis), modules.LoadGenesis(genBytes))
		}
		if cctx.String(makeGenFlag) != "" {
			genesis = node.Override(new(modules.Genesis), testing.MakeGenesis(cctx.String(makeGenFlag)))
		}
		//构造全节点
		var api api.FullNode
		stop, err := node.New(ctx,
     		//node/impl包中的full.go文件，FulleNodeApI是FullNode interface的具体实现
			node.FullAPI(&api),
                              
			node.Online(),
			node.Repo(r),

			genesis,

			node.Override(node.SetApiEndpointKey, func(lr repo.LockedRepo) error {
				apima, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" + cctx.String("api"))
				if err != nil {
					return err
				}
				return lr.SetAPIEndpoint(apima)
			}),

			node.ApplyIf(func(s *node.Settings) bool { return cctx.Bool("bootstrap") },
				node.Override(node.BootstrapKey, modules.Bootstrap),
			),
		)
		if err != nil {
			return err
		}

        // 监听127.0.0.1:1234 启动api服务端
		return serveRPC(api, stop, "127.0.0.1:"+cctx.String("api"))
	},
}
```

- location：cmd/lotus/rpc.go

- method：serveRPC

```go
func serveRPC(a api.FullNode, stop node.StopFunc, addr string) error {
    //创建rpc服务端实例
	rpcServer := jsonrpc.NewServer()
    //在rpc服务中注册FullNode方法，生成对应的rpcHandle
	rpcServer.Register("Filecoin", api.PermissionedFullAPI(a))

	ah := &auth.Handler{
        //jwt校验方法
		Verify: a.AuthVerify,
		Next:   rpcServer.ServeHTTP,
	}

	http.Handle("/rpc/v0", ah)

	srv := &http.Server{Addr: addr, Handler: http.DefaultServeMux}
	//侦听信号量，收到退出信号，执行相关的的关闭操作
	sigChan := make(chan os.Signal, 2)
	go func() {
		<-sigChan
		if err := srv.Shutdown(context.TODO()); err != nil {
			log.Errorf("shutting down RPC server failed: %s", err)
		}
		if err := stop(context.TODO()); err != nil {
			log.Errorf("graceful shutting down failed: %s", err)
		}
	}()
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	return srv.ListenAndServe()
}

// RPCServer provides a jsonrpc 2.0 http server handler
type RPCServer struct {
	methods handlers
}
```

#### 3.2 Lotus-storage-miner

##### 3.2.1 init

```go
var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize a lotus storage miner repo",
	Flags: []cli.Flag{
        ...
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Initializing lotus storage miner")

		//校验proof-params.json文件参数
		if err := build.GetParams(true); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}
		//检测是否已经被初始化过
		repoPath := cctx.String(FlagStorageRepo)
		r, err := repo.NewFS(repoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if ok {
			return xerrors.Errorf("repo at '%s' is already initialized", cctx.String(FlagStorageRepo))
		}

		log.Info("Trying to connect to full node RPC")

		api, closer, err := lcli.GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		log.Info("Checking full node version")

		v, err := api.Version(ctx)
		if err != nil {
			return err
		}

		if v.APIVersion&build.MinorMask != build.APIVersion&build.MinorMask {
			return xerrors.Errorf("Remote API version didn't match (local %x, remote %x)", build.APIVersion, v.APIVersion)
		}

		log.Info("Initializing repo")
		//初始化storage-miner的资源目录./lotusstorage
		if err := r.Init(); err != nil {
			return err
		}
		//初始化storage-miner
		if err := storageMinerInit(ctx, cctx, api, r); err != nil {
			fmt.Printf("ERROR: failed to initialize lotus-storage-miner: %s\n", err)
			fmt.Println("Cleaning up after attempt...")
			if err := os.RemoveAll(repoPath); err != nil {
				fmt.Println("ERROR: failed to clean up failed storage repo: ", err)
			}
			return fmt.Errorf("storage-miner init failed")
		}

		// TODO: Point to setting storage price, maybe do it interactively or something
		log.Info("Storage miner successfully created, you can now start it with 'lotus-storage-miner run'")

		return nil
	},
}

```

- storageMinerInit：初始化storage-miner

```go
func storageMinerInit(ctx context.Context, cctx *cli.Context, api api.FullNode, r repo.Repo) error {
	lr, err := r.Lock()
	if err != nil {
		return err
	}
	defer lr.Close()

	log.Info("Initializing libp2p identity")
    //创建host对应的私钥(./lotusstorage/kestorage/xxx)
	p2pSk, err := makeHostKey(lr)
	if err != nil {
		return err
	}
	//返回与密钥对应的peerid
	peerid, err := peer.IDFromPrivateKey(p2pSk)
	if err != nil {
		return err
	}

	var addr address.Address
	if act := cctx.String("actor"); act != "" {
		a, err := address.NewFromString(act)
		if err != nil {
			return err
		}

		if err := configureStorageMiner(ctx, api, a, peerid, cctx.Bool("genesis-miner")); err != nil {
			return xerrors.Errorf("failed to configure storage miner: %w", err)
		}

		addr = a
	} else {
		a, err := createStorageMiner(ctx, api, peerid, cctx)
		if err != nil {
			return err
		}

		addr = a
	}

	log.Infof("Created new storage miner: %s", addr)

	ds, err := lr.Datastore("/metadata")
	if err != nil {
		return err
	}
	if err := ds.Put(datastore.NewKey("miner-address"), addr.Bytes()); err != nil {
		return err
	}

	return nil
}
```

- configureStorageMiner

```go
func configureStorageMiner(ctx context.Context, api api.FullNode, addr address.Address, peerid peer.ID, genmine bool) error {
    
	if genmine {
		log.Warn("Starting genesis mining. This shouldn't happen when connecting to the real network.")
		//如果是创世区块的存储矿工，需要先执行挖矿在我们执行连操作之前，否则消息不会被挖掘
		if err := api.MinerRegister(ctx, addr); err != nil {
			return err
		}

		defer func() {
			if err := api.MinerUnregister(ctx, addr); err != nil {
				log.Errorf("failed to call api.MinerUnregister: %s", err)
			}
		}()
	}
	// This really just needs to be an api call at this point...
    //获取woker address
	recp, err := api.StateCall(ctx, &types.Message{
		To:     addr,
		From:   addr,
		Method: actors.MAMethods.GetWorkerAddr,
	}, nil)
	if err != nil {
		return xerrors.Errorf("failed to get worker address: %w", err)
	}

	if recp.ExitCode != 0 {
		return xerrors.Errorf("getWorkerAddr returned exit code %d", recp.ExitCode)
	}

	waddr, err := address.NewFromBytes(recp.Return)
	if err != nil {
		return xerrors.Errorf("getWorkerAddr returned bad address: %w", err)
	}

	enc, err := actors.SerializeParams(&actors.UpdatePeerIDParams{PeerID: peerid})
	if err != nil {
		return err
	}

	msg := &types.Message{
		To:       addr,
		From:     waddr,
		Method:   actors.MAMethods.UpdatePeerID,
		Params:   enc,
		Value:    types.NewInt(0),
		GasPrice: types.NewInt(0),
		GasLimit: types.NewInt(100000000),
	}
	//消息入池
	smsg, err := api.MpoolPushMessage(ctx, msg)
	if err != nil {
		return err
	}

	log.Info("Waiting for message: ", smsg.Cid())
    //等待消息被执行
	ret, err := api.StateWaitMsg(ctx, smsg.Cid())
	if err != nil {
		return err
	}

	if ret.Receipt.ExitCode != 0 {
		return xerrors.Errorf("update peer id message failed with exit code %d", ret.Receipt.ExitCode)
	}

	return nil
}
```

- createStorageMiner

```go
func createStorageMiner(ctx context.Context, api api.FullNode, peerid peer.ID, cctx *cli.Context) (addr address.Address, err error) {
	log.Info("Creating StorageMarket.CreateStorageMiner message")
	
	var owner address.Address
	if cctx.String("owner") != "" {
		owner, err = address.NewFromString(cctx.String("owner"))
	} else {
		owner, err = api.WalletDefaultAddress(ctx)
	}
	if err != nil {
		return address.Undef, err
	}

	worker := owner
	if cctx.String("worker") != "" {
		worker, err = address.NewFromString(cctx.String("worker"))
	} else if cctx.Bool("create-worker-key") { // TODO: Do we need to force this if owner is Secpk?
		worker, err = api.WalletNew(ctx, types.KTBLS)
	}
	// TODO: Transfer some initial funds to worker
	if err != nil {
		return address.Undef, err
	}

	collateral, err := api.StatePledgeCollateral(ctx, nil)
	if err != nil {
		return address.Undef, err
	}

	params, err := actors.SerializeParams(&actors.CreateStorageMinerParams{
		Owner:      owner,
		Worker:     worker,
		SectorSize: types.NewInt(build.SectorSize),
		PeerID:     peerid,
	})
	if err != nil {
		return address.Undef, err
	}
	//创建一条给存储市场创建矿工的消息
	createStorageMinerMsg := &types.Message{
		To:    actors.StorageMarketAddress,
		From:  owner,
		Value: collateral,

		Method: actors.SPAMethods.CreateStorageMiner,
		Params: params,

		GasLimit: types.NewInt(10000000),
		GasPrice: types.NewInt(0),
	}
	//消息入池
	signed, err := api.MpoolPushMessage(ctx, createStorageMinerMsg)
	if err != nil {
		return address.Undef, err
	}

	log.Infof("Pushed StorageMarket.CreateStorageMiner, %s to Mpool", signed.Cid())
	log.Infof("Waiting for confirmation")
	//等待消息被执行
	mw, err := api.StateWaitMsg(ctx, signed.Cid())
	if err != nil {
		return address.Undef, err
	}

	if mw.Receipt.ExitCode != 0 {
		return address.Undef, xerrors.Errorf("create storage miner failed: exit code %d", mw.Receipt.ExitCode)
	}

	addr, err = address.NewFromBytes(mw.Receipt.Return)
	if err != nil {
		return address.Undef, err
	}

	log.Infof("New storage miners address is: %s", addr)
	return addr, nil
}

```

- run

```go
var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start a lotus storage miner process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "api",
			Value: "2345",
		},
	},
	Action: func(cctx *cli.Context) error {
		if err := build.GetParams(true); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}

		nodeApi, ncloser, err := lcli.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer ncloser()
		ctx := lcli.DaemonContext(cctx)

		v, err := nodeApi.Version(ctx)
		if err != nil {
			return err
		}
		//在运行存储矿工之前，先做初始化的校验
		storageRepoPath := cctx.String(FlagStorageRepo)
		r, err := repo.NewFS(storageRepoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if !ok {
			return xerrors.Errorf("repo at '%s' is not initialized, run 'lotus-storage-miner init' to set it up", storageRepoPath)
		}

		var minerapi api.StorageMiner
		stop, err := node.New(ctx,
			node.StorageMiner(&minerapi),
			node.Online(),
			node.Repo(r),

			node.Override(node.SetApiEndpointKey, func(lr repo.LockedRepo) error {
				apima, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" + cctx.String("api"))
				if err != nil {
					return err
				}
				return lr.SetAPIEndpoint(apima)
			}),
			node.Override(new(*sectorbuilder.SectorBuilderConfig), modules.SectorBuilderConfig(storageRepoPath)),
			node.Override(new(api.FullNode), nodeApi),
		)
		if err != nil {
			return err
		}

		// Bootstrap with full node
		remoteAddrs, err := nodeApi.NetAddrsListen(ctx)
		if err != nil {
			return err
		}

		if err := minerapi.NetConnect(ctx, remoteAddrs); err != nil {
			return err
		}

		log.Infof("Remote version %s", v)

		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", api.PermissionedStorMinerAPI(minerapi))

		ah := &auth.Handler{
			Verify: minerapi.AuthVerify,
			Next:   rpcServer.ServeHTTP,
		}

		http.Handle("/rpc/v0", ah)
		//监听127.0.0.1：2345，启动storage-miner api服务
		srv := &http.Server{Addr: "127.0.0.1:" + cctx.String("api"), Handler: http.DefaultServeMux}

		sigChan := make(chan os.Signal, 2)
		go func() {
			<-sigChan
			log.Warn("Shutting down..")
			if err := stop(context.TODO()); err != nil {
				log.Errorf("graceful shutting down failed: %s", err)
			}
			if err := srv.Shutdown(context.TODO()); err != nil {
				log.Errorf("shutting down RPC server failed: %s", err)
			}
			log.Warn("Graceful shutdown successful")
		}()
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

		return srv.ListenAndServe()
	},
}
```



