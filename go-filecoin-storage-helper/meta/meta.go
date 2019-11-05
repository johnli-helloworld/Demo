package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"go-filecoin-storage-helper/filhttp"
	"go-filecoin-storage-helper/utils"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

const (
	FilePrefix  = "path"
	ChunkPrefix = "chunk"
	Staged      = "staged"
	Rejected    = "rejected"
	Complete    = "complete"
	Failed      = "failed"
)

type MetaService interface {
	Traversefile(context.Context) error
	Slicefile() (io.Reader, error)
}

type MetaData struct {
	Name      string
	DbName    string
	Size      int64
	Abspath   string
	IsDir     bool
	DS        datastore.Batching
	ChunkSize uint64 //sectorsize
	Miner     string
	Duration  int64
	AskID     int
}

func NewMetaData(fpath string, ssize uint64, miner string, duration int64) (*MetaData, error) {
	fr, err := getFileInfo(fpath)
	if err != nil {
		return nil, err
	}
	fr.Miner = miner
	fr.ChunkSize = ssize
	fr.Duration = duration
	return fr, nil
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
		//是文件则计算文件的大小
		m.Size = fr.Size()
	}
	return m, nil
}

func (m *MetaData) Run(ctx context.Context) error {
	if err := m.Traversefile(ctx); err != nil {
		return err
	}
	if err := m.HandlerFile(ctx); err != nil {
		return err
	}
	return nil
}

func (m *MetaData) Traversefile(ctx context.Context) error {
	//首先判断是文件还是目录
	if !m.IsDir {
		//不是目录则直接记录文件名称及大小 path/{filepath} : filesize
		err := m.DS.Put(datastore.NewKey("/path/filepath"), []byte(strconv.FormatInt(m.Size, 10)))
		if err != nil {
			return err
		}
		return nil
	}
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
		m.DS.Put(datastore.NewKey("path/{path}"), []byte(strconv.FormatInt(m.Size, 10)))
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *MetaData) HandlerFile(ctx context.Context) error {
	//取出badger中所有的文件,"/path/xxx"
	files, err := m.DS.Query(dsq.Query{Prefix: FilePrefix})
	if err != nil {
		fmt.Println("quert path err:", err)
		return err
	}
	for {
		f, ok := files.NextSync()
		if !ok {
			break
		}
		//文件路径fpath: "/path/xxx", fsize: xxx
		fpath := f.Key
		fsize, err := strconv.ParseInt(string(f.Value), 10, 64)
		if err != nil {
			return err
		}
		fmt.Println("fpath==================", fpath, "fsize==================", fsize)
		//若文件大小> 分片大小
		if fsize > m.Size {
			go m.Slicefile(ctx, fpath, fsize)
			continue
		}
		b, err := ioutil.ReadFile(FilePrefix)
		if err != nil {
			return err
		}
		chunk := &Chunk{
			Sequence: 1,
			AbsPath:  fpath,
			Data:     strings.NewReader(string(b)),
		}
		go m.StorageMeta(ctx, chunk)
	}
	return nil
}

type Chunk struct {
	Sequence int
	AbsPath  string
	Data     io.Reader
}

func (m *MetaData) Slicefile(ctx context.Context, fPath string, fsize int64) ([]io.Reader, error) {
	// 计算需要分多少片
	num := utils.ComputeChunks(fsize, int64(m.ChunkSize))
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

		if len(b) > int((fsize - (i-1)*int64(m.ChunkSize))) {
			b = make([]byte, fsize-(i-1)*int64(m.ChunkSize))
		}

		fi.Read(b)

		c := &Chunk{
			Sequence: int(i),
			AbsPath:  fPath,
			Data:     strings.NewReader((string(b))),
		}
		go m.StorageMeta(ctx, c)
	}
	return nil, nil
}

type DealInfo struct {
	Mineraddr string //存储矿工
	DealID    string //订单id
	State     string //订单状态
	Expire    int64  //过期时间
}

//元数据下单
func (m *MetaData) StorageMeta(ctx context.Context, c *Chunk) error {
	//import文件
	cid, err := filhttp.Import(ctx, c.Data)
	if err != nil {
		fmt.Println("meta Import file err:", err)
		return err
	}
	//下单
	resp, err := filhttp.ProposeStorageDeal(ctx, m.Miner, cid, 1, m.Duration)
	if err != nil {
		fmt.Println("meta ProposeStorageDeal err:", err)
		return err
	}
	fmt.Println("storage resp:", resp)
	//记录 key: chunk/"path"/seq/cid; value:dealinfo{}
	//Note: 这里的path不应该是绝对路径,如C:/a/1.txt,若保存a目录,则path: a/1.txt
	cPath := m.chunkPath(c.AbsPath)

	key := datastore.KeyWithNamespaces([]string{ChunkPrefix, cPath, strconv.Itoa(c.Sequence), cid})
	d := &DealInfo{
		Mineraddr: m.Miner,
		DealID:    resp.DealId,
		State:     resp.State,
		Expire:    m.Duration,
	}

	dealenc, err := json.Marshal(d)
	if err != nil {
		return err
	}
	m.DS.Put(key, dealenc)
	return nil
}

// 元数据库下单
func (m *MetaData) StorageMetaDb() {
}

func (m *MetaData) QueryDealStatus(ctx context.Context) error {
	//取出所有的chunk，判断state(rejected, accepted, staged, complete, failed)
	files, err := m.DS.Query(dsq.Query{Prefix: ChunkPrefix})
	if err != nil {
		return err
	}
	chunks, err := files.Rest()
	if err != nil {
		return err
	}
	// for {
	// 	f, ok := files.NextSync()
	// 	if !ok {
	// 		break
	// 	}
	// 	//key: "chunk/path/seq/cid", dealinfo: struct
	// 	fpath := f.Key
	// 	dealInfo := &DealInfo{}
	// 	if err = json.Unmarshal(f.Value, dealInfo); err != nil {
	// 		return err
	// 	}
	// 	//获取dealid查询订单状态。
	// 	resp, err := filhttp.QueryStorageDeal(ctx, dealInfo.DealID)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if resp.State == Failed {

	// 	}
	// }
	return nil
}

func Query(ctx context.Context, chunk string, dealInfo DealInfo) error {
	resp, err := filhttp.QueryStorageDeal(ctx, dealInfo.DealID)
	if err != nil {
		return err
	}
	if resp.State == Failed {

	}
}

func HandleDealInfo() {

}

//假设abspath=“C:/a”, name：“a”, c.path:"C:/a/1.txt" 这时候应记录 a/1.txt
func (m *MetaData) chunkPath(c string) string {
	absPrefix := strings.TrimSuffix(m.Abspath, m.Name)
	cPath := strings.TrimPrefix(c, absPrefix)
	return cPath
}
