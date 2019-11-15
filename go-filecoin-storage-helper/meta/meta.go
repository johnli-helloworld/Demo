package meta

import (
	"fmt"
	"go-filecoin-storage-helper/filhttp"
	"sync"

	"io"
	"os"
	"path/filepath"
	"strings"
)

type MetaData struct {
	//文件/目录名称
	Name string
	//保存元数据的数据库名称
	DbName string
	//文件大小
	Size int64
	//类型: meta; db
	Type string
	//绝对路径
	Abspath string
	//是否是目录
	IsDir bool
	//统计文件/目录总共分多少片，即chunk总数量
	Slices  int
	sliceLk sync.Mutex
	wg      sync.WaitGroup
	//
	Mstore *MetaStore

	//params
	SectorSize uint64
	Miner      string
	Duration   int64
	AskId      string
}

const (
	AbsPathPrefix = "/abspathprefix"
	FilePrefix    = "/path"
	ChunkPrefix   = "/chunk"
	ChunkSizeKey  = "/chunksize"
	Rejected      = "rejected"
	Accepted      = "accepted"
	Started       = "started"
	Failed        = "failed"
	Staged        = "staged"
	Complete      = "complete"
)

//DealStateMap ,状态映射
var DealStateMap = map[string]int{
	Rejected: 2,
	Accepted: 3,
	Started:  4,
	Failed:   5,
	Staged:   6,
	Complete: 7,
}

var StorageAPI = filhttp.Newhttp("").Storage()

func NewMetaData(fpath string, t string) (*MetaData, error) {
	m := &MetaData{}
	m.Abspath = fpath
	m.Type = t
	return m.getFileInfo()
}

func (m *MetaData) getFileInfo() (*MetaData, error) {
	abspath, err := filepath.Abs(m.Abspath)
	if err != nil {
		return nil, err
	}
	fr, err := os.Stat(abspath)
	if err != nil {
		if os.IsNotExist(err) {
			return m, fmt.Errorf("file or dir is not exist")
		}
	}
	m.Abspath = abspath
	m.Name = fr.Name()
	if fr.IsDir() {
		m.IsDir = true
	} else {
		m.IsDir = false
	}
	m.Size = fr.Size()
	return m, nil
}

type Chunk struct {
	Sequence int
	AbsPath  string
	Data     io.Reader
}

type MetadbFile struct {
	Meta map[string]string `json:"meta"`
}

type DealInfo struct {
	Cid       string //文件cid
	Mineraddr string //存储矿工
	DealId    string //订单id
	State     string //订单状态
	Expire    int64  //过期时间
}

//if abspath=“C:/a”, name：“a”, c.path:"C:/a/1.txt" then cPath: a/1.txt
func (m *MetaData) chunkPath(c string) string {
	absPrefix := strings.TrimSuffix(m.Abspath, m.Name)
	cPath := strings.TrimPrefix(c, absPrefix)
	return cPath
}
