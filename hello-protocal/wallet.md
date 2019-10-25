## 1.源码信息

- version
- location
  - wallet

## 2.wallet包概述

```go
// KeyInfo is a key and its type used for signing.
type KeyInfo struct {
	// Private key.
	PrivateKey []byte `json:"privateKey"`
	// Curve used to generate private key.
	Curve string `json:"curve"`
}

Privatekey生成Publickey
Publickey生成address
```



## 3.源码分析

### 3.1 wallet

#### 3.1.1 数据结构

```go
type Wallet struct {
	lk sync.Mutex

	backends map[reflect.Type][]Backend
}
```

#### 3.1.2 函数

- New()：构造wallet，管理所有backend存储的地址

```go
func New(backends ...Backend) *Wallet {}
```

- NewAddress()：新建一个账户地址，用默认的dsbacktype

```go
func NewAddress(w *Wallet) (address.Address, error) {}
```

#### 3.1.3 方法

```go
//检索并返回所有被存储的地址
- Addresses() []address.Address
//根据类型返回[]backend
- Backends(kind reflect.Type) []Backend
//Ecrecover返回一个未压缩的公钥，该公钥可以从数据生成给定的签名。(注意:返回的公钥不能用于验证‘data’是否有效，因为一个公钥可能有N个私钥对)
- Ecrecover(data []byte, sig types.Signature) ([]byte, error)
//导出返回给定钱包地址的密钥
- Export(addrs []address.Address) ([]*types.KeyInfo, error)
//在所有的backend中检索指定的addr,返回相应的backend
- Find(addr address.Address) (Backend, error)
//根据给定的公钥返回其地址
- GetAddressForPubKey(pk []byte) (address.Address, error)
//根据给定addr返回其公钥
- GetPubKeyForAddress(addr address.Address) ([]byte, error)
//检查给定的addr是否被存储
- HasAddress(a address.Address)
//根据给定的密钥信息将地址添加到钱包
- Import(kinfos ...*types.KeyInfo) ([]address.Address, error)
//根据给定的addr获取其密钥信息
- keyInfoForAddr(addr address.Address) (*types.KeyInfo, error)
//生成新的密钥信息
- NewKeyInfo() (*types.KeyInfo, error)
//使用与地址“addr”对应的私钥对“数据”进行密码签名
- SignBytes(data []byte, addr address.Address)
//校验sig是data的散列hash,它的公钥是pk
- Verify(data []byte, pk []byte, sig types.Signature)

```

### 3.2 backend

#### 3.2.1 接口

```go
type Backend interface {
	// Addresses returns a list of all accounts currently stored in this backend.
	Addresses() []address.Address

	// Contains returns true if this backend stores the passed in address.
	HasAddress(addr address.Address) bool

	// Sign cryptographically signs `data` using the private key `priv`.
	SignBytes(data []byte, addr address.Address) (types.Signature, error)

	// Verify cryptographically verifies that 'sig' is the signed hash of 'data' with
	// the public key `pk`.
	Verify(data, pk []byte, sig types.Signature) bool

	// GetKeyInfo will return the keyinfo associated with address `addr`
	// iff backend contains the addr.
	GetKeyInfo(addr address.Address) (*types.KeyInfo, error)
}

type Importer interface {
	// ImportKey imports the key described by the given keyinfo
	// into the backend
	ImportKey(ki *types.KeyInfo) error
}
```

### 3.3 dsbackend

```go
//重点：wallet数据是如何在dsDatastore中存储的  key:walletaddress  value:keyinfo
func (backend *DSBackend) putKeyInfo(ki *types.KeyInfo) error {
	a, err := ki.Address()
	if err != nil {
		return err
	}

	backend.lk.Lock()
	defer backend.lk.Unlock()

	kib, err := ki.Marshal()
	if err != nil {
		return err
	}

	if err := backend.ds.Put(ds.NewKey(a.String()), kib); err != nil {
		return errors.Wrap(err, "failed to store new address")
	}

	backend.cache[a] = struct{}{}
	return nil
}
```

#### 3.3.1 数据结构

```go
// DSBackend is a wallet backend implementation for storing addresses in a datastore.
type DSBackend struct {
	lk sync.RWMutex

	// TODO: use a better interface that supports time locks, encryption, etc.
	ds repo.Datastore

	// TODO: proper cache
	cache map[address.Address]struct{}
}
```

#### 3.3.2 函数

```go
func NewDSBackend(ds repo.Datastore) (*DSBackend, error) {}
```

#### 3.3.3 方法

```go
//返回所有存储在dsbackstore的address
- Addresses() []address.Address
//根据给定的addr返回其密钥信息
- GetKeyInfo(addr address.Address) (*types.KeyInfo, error)
//检查传入的地址是否存储在这个backend
- HasAddress(addr address.Address) bool
//根据密钥信息导入钱包地址
- ImportKey(ki *types.KeyInfo) error
//创建新地址并返回
- NewAddress() (address.Address, error)
//将地址信息持久化到ds
- putKeyInfo(ki *types.KeyInfo) error
//使用私钥对数据签名
- SignBytes(data []byte, addr address.Address) (types.Signature, error)
//校验sig是data的散列hash,它的公钥是pk
- Verify(data, pk []byte, sig types.Signature) bool
```

