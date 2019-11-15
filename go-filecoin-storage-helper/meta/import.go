package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-filecoin-storage-helper/repo"
	"go-filecoin-storage-helper/utils"
	"path/filepath"

	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

func (m *MetaData) Import(ctx context.Context) (string, error) {
	if err := m.Traversefile(ctx); err != nil {
		return "", err
	}
	d, err := m.ImportFile(ctx)
	if err != nil {
		return "", err
	}
	return d, nil
}

func (m *MetaData) Traversefile(ctx context.Context) error {
	if !m.IsDir {
		//key: /path/{abspath}; value: size
		key := datastore.KeyWithNamespaces([]string{FilePrefix, m.Abspath})
		err := m.Mstore.DS.Put(key, []byte(strconv.FormatInt(m.Size, 10)))
		if err != nil {
			return err
		}
	} else {
		err := filepath.Walk(m.Abspath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			fmt.Println(path, info.Size())
			//key: /path/{abspath}; value: size
			key := datastore.KeyWithNamespaces([]string{FilePrefix, path})
			m.Mstore.DS.Put(key, []byte(strconv.FormatInt(info.Size(), 10)))
			return nil
		})
		if err != nil {
			return err
		}
	}
	m.Mstore.DS.Put(datastore.NewKey("type"), []byte(m.Type))
	if err := m.computeSlice(); err != nil {
		fmt.Println("ComputeSlice err", err)
		return err
	}
	m.Mstore.DS.Put(datastore.NewKey(ChunkSizeKey), []byte(strconv.FormatUint(m.SectorSize, 10)))
	m.Mstore.DS.Put(datastore.NewKey("slices"), []byte(strconv.Itoa(m.Slices)))
	abspath := strings.TrimSuffix(m.Abspath, m.Name)
	m.Mstore.DS.Put(datastore.NewKey(AbsPathPrefix), []byte(abspath))
	return nil
}

func (m *MetaData) ImportFile(ctx context.Context) (string, error) {
	files, err := m.Mstore.DS.Query(dsq.Query{Prefix: FilePrefix})
	if err != nil {
		return "", err
	}
	m.wg.Add(m.Slices)
	fmt.Println("total slice:", m.Slices)
	for {
		f, ok := files.NextSync()
		if !ok {
			break
		}
		//fpath: "{abspath}"
		fpath := strings.TrimPrefix(f.Key, FilePrefix)
		fsize, err := strconv.ParseUint(string(f.Value), 10, 64)
		if err != nil {
			return "", err
		}
		// fmt.Println("fpath=======", fpath, "fsize======", fsize)
		if fsize > m.SectorSize {
			go m.slicefile(ctx, fpath, fsize, &m.wg)
			continue
		}
		b, err := ioutil.ReadFile(fpath)
		if err != nil {
			return "", err
		}
		chunk := &Chunk{
			Sequence: 1,
			AbsPath:  fpath,
			Data:     strings.NewReader(string(b)),
		}
		go m.importchunk(ctx, chunk, &m.wg)
	}
	m.wg.Wait()

	cid, err := m.handlerdb(ctx)
	if err != nil {
		return "", err
	}
	return cid, nil
}

func (m *MetaData) handlerdb(ctx context.Context) (string, error) {
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return "", errors.New("ctx value repopath not found")
	}
	if err := m.compressdb(ctx); err != nil {
		return "", errors.New("handlerdb.compressdb err")
	}
	//判断meta下是否已经存在原文件的datastore，有则删除
	srcDirPath := utils.NewPath([]string{repopath, "meta", m.DbName})
	if utils.Exists(srcDirPath) {
		utils.RemoveFileOrDir(srcDirPath)
	}
	dbmeta, err := m.generatedbmeta(ctx)
	if err != nil {
		return "", errors.New("generatedbmeta err")
	}

	var cid string
	if dbmeta.Size > int64(dbmeta.SectorSize) {
		if err := dbmeta.Traversefile(ctx); err != nil {
			return "", nil
		}
		return dbmeta.ImportFile(ctx)
	} else {
		cid, err = dbmeta.importdb(ctx)
		if err != nil {
			fmt.Println("handlerdb importdb err:", err)
			return "", err
		}
	}
	return cid, nil
}

func (m *MetaData) generatedbmeta(ctx context.Context) (*MetaData, error) {
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return nil, errors.New("ctx value repopath not found")
	}
	destFilePath := utils.NewPath([]string{repopath, "dbfile", m.DbName + ".tar.zip"})
	dbmeta, err := NewMetaData(destFilePath, "db")
	s, err := NewMetastore(repopath, "meta", m.DbName)
	if err != nil {
		return nil, err
	}
	dbmeta.Mstore = s
	dbmeta.DbName = m.DbName
	dbmeta.SectorSize = m.SectorSize
	return dbmeta, err
}

//压缩数据库
func (m *MetaData) compressdb(ctx context.Context) error {
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return errors.New("ctx value repopath not found")
	}
	srcDirPath := utils.NewPath([]string{repopath, "meta", m.DbName})
	destFilePath := utils.NewPath([]string{repopath, "dbfile", m.DbName + ".tar.zip"})
	//判断是否存在destFilePath，有则删除
	if utils.Exists(destFilePath) {
		if err := utils.RemoveFileOrDir(destFilePath); err != nil {
			return errors.New("compressdb remove zip err")
		}
	}
	utils.TarGz(srcDirPath, destFilePath)
	return nil
}

func (m *MetaData) slicefile(ctx context.Context, fPath string, fsize uint64, wg *sync.WaitGroup) error {
	num := utils.ComputeChunks(fsize, m.SectorSize)
	// TODO分片处理
	fi, err := os.OpenFile(fPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	b := make([]byte, m.SectorSize)
	var i int64 = 1
	for ; i <= int64(num); i++ {

		fi.Seek((i-1)*(int64(m.SectorSize)), 0)

		if len(b) > int((int64(fsize) - (i-1)*int64(m.SectorSize))) {
			b = make([]byte, int64(fsize)-(i-1)*int64(m.SectorSize))
		}

		fi.Read(b)

		c := &Chunk{
			Sequence: int(i),
			AbsPath:  fPath,
			Data:     strings.NewReader((string(b))),
		}
		go m.importchunk(ctx, c, wg)
	}
	return nil
}

type ChunkInfo struct {
	Cid   string
	Miner string
}

func (m *MetaData) importchunk(ctx context.Context, c *Chunk, wg *sync.WaitGroup) (string, error) {
	defer wg.Done()
	cid, err := StorageAPI.Import(ctx, c.Data)
	if err != nil {
		fmt.Println("meta Import file err:", err)
		return "", err
	}
	cPath := m.chunkPath(c.AbsPath)
	key := datastore.KeyWithNamespaces([]string{ChunkPrefix, cPath, strconv.Itoa(c.Sequence)})
	fmt.Printf("chunkey*********%+v\n", key)
	d := &ChunkInfo{
		Cid:   cid,
		Miner: m.Miner,
	}
	dealenc, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	m.Mstore.DS.Put(key, dealenc)
	return cid, nil
}

func (m *MetaData) importdb(ctx context.Context) (string, error) {
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return "", errors.New("ctx value repopath not found")
	}
	fileName := utils.NewPath([]string{repopath, "dbfile", m.DbName + ".tar.zip"})
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	cid, err := StorageAPI.Import(ctx, strings.NewReader(string(b)))
	if err != nil {
		return "", err
	}
	return cid, nil
}

//Calculate the total number of slices of files/directories
func (m *MetaData) computeSlice() error {
	//query {key: /path/{abspath}}
	files, err := m.Mstore.DS.Query(dsq.Query{Prefix: FilePrefix})
	if err != nil {
		return err
	}
	for {
		f, ok := files.NextSync()
		if !ok {
			break
		}
		fsize, err := strconv.ParseUint(string(f.Value), 10, 64)
		if err != nil {
			return err
		}
		num := 1
		if fsize > m.SectorSize {
			fmt.Println("fpath", f.Key, "fsize", fsize)
			num = utils.ComputeChunks(fsize, m.SectorSize)
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
