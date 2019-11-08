package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go-filecoin-storage-helper/filhttp"
	"go-filecoin-storage-helper/repo"
	"go-filecoin-storage-helper/utils"

	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

const (
	FilePrefix  = "/path"
	ChunkPrefix = "/chunk"
	Accepted    = "accepted"
	Staged      = "staged"
	Rejected    = "rejected"
	Complete    = "complete"
	Failed      = "failed"
)

type MetaService interface {
	Traversefile(context.Context) error
	Slicefile() (io.Reader, error)
	//存储记录元数据库文件
	StorageDbFile(context.Context) (*DealInfo, error)
}

type MetaData struct {
	//文件/目录名称
	Name string
	//保存元数据的数据库名称
	DbName string
	//文件大小
	Size int64
	//绝对路径
	Abspath string
	//是否是目录
	IsDir bool
	//统计文件总共分多少片
	Slices  int
	sliceLk sync.Mutex
	wg      sync.WaitGroup
	//
	Mstore *MetaStore

	//params
	ChunkSize uint64
	Miner     string
	Duration  int64
	AskID     int
}

func NewMetaData(fpath string, ssize uint64, miner string, duration int64) (*MetaData, error) {
	m, err := getFileInfo(fpath)
	if err != nil {
		return nil, err
	}
	m.Miner = miner
	m.ChunkSize = ssize
	m.Duration = duration
	return m, nil
}

func getFileInfo(fpath string) (*MetaData, error) {
	m := &MetaData{}
	//获取文件的绝对路径
	abspath, err := filepath.Abs(fpath)
	if err != nil {
		return nil, err
	}
	//判断文件或文件夹是否存在
	fr, err := os.Stat(abspath)
	if err != nil {
		if os.IsNotExist(err) {
			return m, fmt.Errorf("file or dir is not exist")
		}
	}
	m.Abspath = abspath
	m.Name = fr.Name()
	//判断是目录还是文件
	if fr.IsDir() {
		m.IsDir = true
	} else {
		m.IsDir = false
	}
	m.Size = fr.Size()
	return m, nil
}

func (m *MetaData) Run(ctx context.Context) (*DealInfo, error) {
	if err := m.Traversefile(ctx); err != nil {
		return nil, err
	}
	d, err := m.HandlerFile(ctx)
	if err != nil {
		fmt.Println("Run HandlerFile err", err)
		return d, err
	}
	return d, nil
}

func (m *MetaData) Traversefile(ctx context.Context) error {
	//首先判断是文件还是目录
	if !m.IsDir {
		//不是目录则直接记录文件名称及大小 path/{filepath} : filesize
		key := datastore.KeyWithNamespaces([]string{FilePrefix, m.Abspath})
		err := m.Mstore.DS.Put(key, []byte(strconv.FormatInt(m.Size, 10)))
		if err != nil {
			return err
		}
		return nil
	} else {
		//根据指定的文件路径遍历文件，获取子文件的路径和大小，datastor
		err := filepath.Walk(m.Abspath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			fmt.Println(path, info.Size())
			//将文件记录此时db中保存的都是文件/path/{filepath} : filesize)，
			key := datastore.KeyWithNamespaces([]string{FilePrefix, path})
			m.Mstore.DS.Put(key, []byte(strconv.FormatInt(info.Size(), 10)))
			return nil
		})
		if err != nil {
			return err
		}
	}

	//标识元数据库
	m.Mstore.DS.Put(datastore.NewKey("type"), []byte("meta"))
	//统计分片总数
	if err := m.computeSlice(); err != nil {
		fmt.Println("ComputeSlice err", err)
		return err
	}
	//将分片的规则chunksize记录进去
	m.Mstore.DS.Put(datastore.NewKey("chunksize"), []byte(strconv.FormatUint(m.ChunkSize, 10)))
	//将分片总数也记录进去
	m.Mstore.DS.Put(datastore.NewKey("slices"), []byte(strconv.Itoa(m.Slices)))
	//记录绝对路径前缀，为了以后还原file方便
	abspath := strings.TrimSuffix(m.Abspath, m.Name)
	m.Mstore.DS.Put(datastore.NewKey("absprefix"), []byte(abspath))
	return nil
}

//统计文件/目录 总共要分成多少片
func (m *MetaData) computeSlice() error {
	fmt.Println("ComputeSlice chunksize", m.ChunkSize)
	//取出badger中所有的文件, key:"/path/xxx"
	files, err := m.Mstore.DS.Query(dsq.Query{Prefix: FilePrefix})
	if err != nil {
		return err
	}
	for {
		f, ok := files.NextSync()
		if !ok {
			break
		}
		//计算fsize看是否需要分片
		fsize, err := strconv.ParseUint(string(f.Value), 10, 64)
		if err != nil {
			return err
		}
		num := 1
		if fsize > m.ChunkSize {
			fmt.Println("fpath", f.Key, "fsize", fsize)
			num = utils.ComputeChunks(fsize, m.ChunkSize)
			fmt.Println("num", num)
		}
		m.addSlices(num)
	}
	return nil
}

func (m *MetaData) addSlices(num int) {
	m.sliceLk.Lock()
	m.Slices = m.Slices + num
	m.sliceLk.Unlock()
	return
}

func (m *MetaData) HandlerFile(ctx context.Context) (*DealInfo, error) {
	//取出badger中所有的文件, key:"/path/xxx"
	files, err := m.Mstore.DS.Query(dsq.Query{Prefix: FilePrefix})
	if err != nil {
		fmt.Println("query path err:", err)
		return nil, err
	}
	m.wg.Add(m.Slices)
	fmt.Println("total slice:", m.Slices)
	for {
		f, ok := files.NextSync()
		if !ok {
			break
		}
		//文件路径fpath: "/path/xxx", fsize: xxx
		fpath := strings.TrimPrefix(f.Key, FilePrefix)
		fsize, err := strconv.ParseUint(string(f.Value), 10, 64)
		if err != nil {
			return nil, err
		}
		fmt.Println("fpath==================", fpath, "fsize==================", fsize)
		//若文件大小> 分片大小
		if fsize > m.ChunkSize {
			go m.Slicefile(ctx, strings.TrimPrefix(fpath, FilePrefix), fsize, &m.wg)
			continue
		}
		b, err := ioutil.ReadFile(fpath)
		if err != nil {
			return nil, err
		}
		chunk := &Chunk{
			Sequence: 1,
			AbsPath:  fpath,
			Data:     strings.NewReader(string(b)),
		}
		fmt.Printf("chunk==========%+v\n", chunk)
		go m.StorageMeta(ctx, chunk, &m.wg)
	}
	m.wg.Wait()
	//TODO:等待所有的订单都下单(需要先校验有没有失败的订单)
	//只有元数据没有下单失败的，才能再对元数据库下单
	//校验所有的订单状态，有failed则为失败
	if len(m.Mstore.FailedDeals) != 0 {
		return nil, errors.New("deal failed")
	}

	//这里将badger中的key:value记录到文件中(因为badger本身是个目录)
	if err = m.genaratedbfile(ctx); err != nil {
		return nil, err
	}
	//对记录元数据库文件下单
	//Note:记录元数据库的文件下单成功才算真正的下单成功
	d, err := m.StorageDbFile(ctx)
	if err != nil {
		return nil, err
	}
	return d, nil
}

type Chunk struct {
	Sequence int
	AbsPath  string
	Data     io.Reader
}

func (m *MetaData) Slicefile(ctx context.Context, fPath string, fsize uint64, wg *sync.WaitGroup) ([]io.Reader, error) {
	// 计算需要分多少片
	num := utils.ComputeChunks(fsize, m.ChunkSize)
	fmt.Println("ComputeChunks:", num)
	// TODO分片处理

	fi, err := os.OpenFile(fPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}
	b := make([]byte, m.ChunkSize)
	var i int64 = 1
	for ; i <= int64(num); i++ {

		fi.Seek((i-1)*(int64(m.ChunkSize)), 0)

		if len(b) > int((int64(fsize) - (i-1)*int64(m.ChunkSize))) {
			b = make([]byte, int64(fsize)-(i-1)*int64(m.ChunkSize))
		}

		fi.Read(b)

		c := &Chunk{
			Sequence: int(i),
			AbsPath:  fPath,
			Data:     strings.NewReader((string(b))),
		}
		go m.StorageMeta(ctx, c, wg)
	}
	return nil, nil
}

func (m *MetaData) StorageDbFile(ctx context.Context) (*DealInfo, error) {
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return nil, errors.New("ctx value repopath not found")
	}
	fileName := utils.NewPath([]string{repopath, "dbfile", m.DbName})
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	cid, err := filhttp.Import(ctx, strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	// resp, err := filhttp.ProposeStorageDeal(ctx, m.Miner, cid, 1, m.Duration)
	// if err != nil {
	// 	return nil, errors.New("StorageDbFile deal err")
	// }
	d := &DealInfo{
		Cid:       cid,
		Mineraddr: m.Miner,
		DealID:    "111111111111111111111111",
		State:     Accepted,
		Expire:    m.Duration,
	}
	return d, nil
}

type MetadbFile struct {
	Meta map[string]string `json:"meta"`
}

func (m *MetaData) handlerDbfile(ctx context.Context) (*DealInfo, error) {
	if err := m.genaratedbfile(ctx); err != nil {
		return nil, err
	}
	dbmeta, err := m.generatedbMetadata(ctx)
	if err != nil {
		return nil, err
	}
	d := &DealInfo{}
	//若记录元数据的文件又大于扇区大小，则还需要继续分割
	if dbmeta.Size > int64(dbmeta.ChunkSize) {
		if err := m.Traversefile(ctx); err != nil {
			return d, err
		}
	} else {
		d, err = m.StorageDbFile(ctx)
		if err != nil {
			return d, err
		}
	}
	return d, nil
}

// 元数据库下单(可以将元数据库中的key：value键值对全部取出保存到一个文件中，对文件下单)
func (m *MetaData) genaratedbfile(ctx context.Context) error {
	res, err := m.Mstore.DS.Query(dsq.Query{})
	if err != nil {
		return err
	}
	//取出所有的键值对
	kv, err := res.Rest()
	if err != nil {
		return err
	}
	tmpmap := make(map[string]string, len(kv))
	for _, v := range kv {
		tmpmap[v.Key] = string(v.Value)
	}
	mf := &MetadbFile{}
	mf.Meta = tmpmap
	enc, err := json.Marshal(mf)
	if err != nil {
		return err
	}
	//生成文件
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return errors.New("ctx value repopath not found")
	}

	fileName := utils.NewPath([]string{repopath, "dbfile", m.DbName})
	fmt.Println("fileName:", fileName)
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Println("GenarateDbFile err:", err)
		return err
	}
	if _, err := f.Write(enc); err != nil {
		return err
	}
	return nil
}

func (m *MetaData) generatedbMetadata(ctx context.Context) (*MetaData, error) {
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return nil, errors.New("ctx value repopath not found")
	}
	dbfilepath := utils.NewPath([]string{repopath, "dbfile", m.DbName})
	md, err := NewMetaData(dbfilepath, m.ChunkSize, m.Miner, m.Duration)
	if err != nil {
		return nil, err
	}
	//用 memrepo记录dbfile
	dbstore, err := NewMemstore()
	if err != nil {
		return nil, err
	}
	md.Mstore = dbstore
	// md.DsName = m.DsName
	return md, err
}

type DealInfo struct {
	Cid       string //文件cid
	Mineraddr string //存储矿工
	DealID    string //订单id
	State     string //订单状态
	Expire    int64  //过期时间
}

//元数据下单
func (m *MetaData) StorageMeta(ctx context.Context, c *Chunk, wg *sync.WaitGroup) error {
	defer wg.Done()
	//import文件
	cid, err := filhttp.Import(ctx, c.Data)
	if err != nil {
		fmt.Println("meta Import file err:", err)
		return err
	}
	d := &DealInfo{}
	//记录 key: /chunk/"path"/seq/cid; value:dealinfo{}
	//Note: 这里的path不应该是绝对路径,如C:/a/1.txt,若保存a目录,则path: a/1.txt
	cPath := m.chunkPath(c.AbsPath)
	key := datastore.KeyWithNamespaces([]string{ChunkPrefix, cPath, strconv.Itoa(c.Sequence)})
	// denc, err := json.Marshal(d)
	m.Mstore.DS.Put(key, []byte(""))
	//下单
	// resp, err := filhttp.ProposeStorageDeal(ctx, m.Miner, cid, 1, m.Duration)
	// if err != nil {
	// 	fmt.Println("meta ProposeStorageDeal err:", err)
	// 	return err
	// }
	// fmt.Println("storage resp:", resp)

	//记录失败的订单
	// if resp.State != Accepted {
	// 	m.Mstore.FailedDeals[c.AbsPath] = make(chan struct{})
	// }

	// key := datastore.KeyWithNamespaces([]string{ChunkPrefix, cPath, strconv.Itoa(c.Sequence), cid})
	d = &DealInfo{
		Cid:       cid,
		Mineraddr: m.Miner,
		DealID:    c.AbsPath,
		State:     Accepted,
		Expire:    m.Duration,
	}
	fmt.Println("key======", key, "dealinfo===", d)
	dealenc, err := json.Marshal(d)
	if err != nil {
		return err
	}
	m.Mstore.DS.Put(key, dealenc)
	return nil
}

func (m *MetaData) QueryDealStatus(ctx context.Context) error {
	//取出所有的chunk，判断state(rejected, accepted, staged, complete, failed)
	files, err := m.Mstore.DS.Query(dsq.Query{Prefix: ChunkPrefix})
	if err != nil {
		return err
	}

	for {
		f, ok := files.NextSync()
		if !ok {
			break
		}
		//key: "chunk/path/seq/cid", dealinfo: struct
		fpath := f.Key
		dealInfo := &DealInfo{}
		if err = json.Unmarshal(f.Value, dealInfo); err != nil {
			return err
		}
		//获取dealid查询订单状态。
		resp, err := filhttp.QueryStorageDeal(ctx, dealInfo.DealID)
		if err != nil {
			return err
		}
		if resp.State == Failed {
			m.Mstore.FailedDeals[fpath] = make(chan struct{})
		}
		//状态都是success时
	}
	return nil
}

func ReductionDS(ctx context.Context, cid string) (*MetaData, error) {
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return nil, errors.New("ctx value repopath not found")
	}
	//通过cid获取文件
	res, err := filhttp.Cat(ctx, cid)
	if err != nil {
		return nil, errors.New("ReductionDS cat err")
	}
	b, err := ioutil.ReadAll(res)

	metafile := &MetadbFile{}
	if err = json.Unmarshal(b, metafile); err != nil {
		return nil, errors.New("Unmarshal err")
	}
	//保存到repo目录下
	metafilepath := utils.NewPath([]string{repopath})
	f, err := os.Create(metafilepath)
	if err != nil {
		return nil, errors.New("create db file err")
	}
	if _, err := f.Write(b); err != nil {
		return nil, err
	}
	if _, ok := metafile.Meta["/type"]; ok {
		return nil, errors.New("Incorrect source file")
	}
	//判断取到的文件是否是记录元数据库的文件
	//不是则继续根据cid寻找
	m := &MetaData{}
	if metafile.Meta["/type"] == "meta" {
		//是则将metafile的数据刷入数据库中还原
		chunksize, err := strconv.ParseUint(metafile.Meta["/chunksize"], 10, 64)
		if err != nil {
			return nil, errors.New("parse chunksize err")
		}
		m, err = NewMetaData(metafilepath, chunksize, "xxx", 2880)
		if err != nil {
			return nil, errors.New("NewMetaData meta err")
		}
		m.AbsPath = metafile.Meta["/absprefix"] 
		s, err := NewMetastore(repopath, "meta", cid)
		if err != nil {
			return nil, errors.New("new metastore err")
		}
		m.Mstore = s

		for k, v := range metafile.Meta {
			m.Mstore.DS.Put(datastore.NewKey(k), []byte(v))
		}
	}
	return m, nil
}

//还原文件的操作
func (m *MetaData) ReductionFile(destpath string) error {
	//从badger中获取所有的path构建原目录
	paths, err := m.Mstore.DS.Query(dsq.Query{Prefix: FilePrefix})
	if err != nil {
		return err
	}
	for {
		f, ok := paths.NextSync()
		if !ok {
			break
		}
		//先还原目录 fpath: /path/"C:/a/b/1.txt"
		fpath := strings.TrimPrefix(f.Key, FilePrefix + m.Abspath)
		fsize, err := strconv.ParseUint(string(f.Value), 10, 64)
		if err != nil {
			return err
		}
		//根据fize和chunksize可以算出piece有多少chunk
		chunknum := utils.ComputeChunks(fsize, m.ChunkSize)
		if err := utils.GenerateFileByPath(destpath + fpath); err != nil{
			return err
		}
		go m.generateChildFile(destpath + fpath, chunknum)
	}
	return nil
}

func (m *MetaData) generateChildFile(filepath string, num int) error {
	for i := 1; i <= num; i++ {
		m.Mstore.DS.Get("")
	}
	return nil
}

func Query(ctx context.Context, chunk string, dealInfo DealInfo) error {
	resp, err := filhttp.QueryStorageDeal(ctx, dealInfo.DealID)
	if err != nil {
		return err
	}
	if resp.State == Failed {

	}
	return nil
}

//假设abspath=“C:/a”, name：“a”, c.path:"C:/a/1.txt" 这时候应记录 a/1.txt
func (m *MetaData) chunkPath(c string) string {
	absPrefix := strings.TrimSuffix(m.Abspath, m.Name)
	cPath := strings.TrimPrefix(c, absPrefix)
	return cPath
}
