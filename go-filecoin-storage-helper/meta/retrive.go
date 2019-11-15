package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"go-filecoin-storage-helper/utils"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badger "github.com/ipfs/go-ds-badger"
)

func RunReduct(ctx context.Context, cid string, destpath string) error {
	resp, err := StorageAPI.Cat(ctx, cid)
	if err != nil {
		return err
	}
	info, err := ioutil.ReadAll(resp)
	if err != nil {
		return err
	}
	//这里应该用一个随机数生成  xxx.tar.gz
	dbgzpath := utils.NewPath([]string{destpath, "111.tar.gz"})
	f, err := os.OpenFile(dbgzpath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	if _, err = f.Write(info); err != nil {
		return err
	}

	if err := reductdb(ctx, dbgzpath, destpath); err != nil {
		return err
	}
	return nil
}

func reductdb(ctx context.Context, dbgzpath string, destpath string) error {
	dirName, err := utils.UnTarGz(dbgzpath, destpath)
	if err != nil {
		return err
	}

	opts := badger.DefaultOptions
	opts.Truncate = true
	destdb := utils.NewPath([]string{destpath, dirName})
	ds, err := badger.NewDatastore(destdb, &opts)
	if err != nil {
		return err
	}

	t, err := ds.Get(datastore.NewKey("type"))
	if err != nil {
		return err
	}
	m := &MetaStore{
		DS:          ds,
		FailedDeals: map[string]chan struct{}{},
	}
	if string(t) == "db" {
		if err := m.generateFile(ctx, destpath); err != nil {
			return err
		}
		reductdb(ctx, dbgzpath, destpath)
	} else {
		m.generateFile(ctx, destpath)
	}
	return nil
}

// func

func (ds *MetaStore) generateFile(ctx context.Context, destpath string) error {
	paths, err := ds.DS.Query(dsq.Query{Prefix: FilePrefix})
	if err != nil {
		return err
	}
	cs, err := ds.DS.Get(datastore.NewKey("chunksize"))
	if err != nil {
		return err
	}
	chunksize, err := strconv.ParseUint(string(cs), 10, 64)

	abspath, err := ds.DS.Get(datastore.NewKey("abspathprefix"))
	if err != nil {
		return err
	}
	for {
		f, ok := paths.NextSync()
		if !ok {
			break
		}
		//f.Key: /path/C:/a/b/1.txt;
		fpath := strings.TrimPrefix(f.Key, FilePrefix+string(abspath))
		fsize, err := strconv.ParseUint(string(f.Value), 10, 64)
		if err != nil {
			return err
		}
		fmt.Println("fpath===========", fpath, "fsize", fsize)
		chunknum := utils.ComputeChunks(fsize, chunksize)
		//if destpath: D:/a; targetpath: D:/a/1.txt
		targetpath := utils.NewPath([]string{destpath, fpath})
		if err := utils.GenerateFileByPath(targetpath); err != nil {
			return err
		}

		ds.generatechildfile(ctx, targetpath, fpath, chunknum)
	}
	return nil
}

func (ds *MetaStore) generatechildfile(ctx context.Context, targetpath string, chunkpath string, num int) error {
	f, err := os.OpenFile(targetpath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := 1; i <= num; i++ {
		//Restore the key in the badger database;
		key := datastore.KeyWithNamespaces([]string{ChunkPrefix, chunkpath, strconv.Itoa(i)})
		d, err := ds.DS.Get(key)
		if err != nil {
			return err
		}
		chunkinfo := &ChunkInfo{}
		if err = json.Unmarshal(d, chunkinfo); err != nil {
			return err
		}
		resp, err := StorageAPI.Cat(ctx, chunkinfo.Cid)
		if err != nil {
			return err
		}
		info, err := ioutil.ReadAll(resp)
		if err != nil {
			return err
		}
		_, err = f.Write(info)
		if err != nil {
			return err
		}
	}
	return nil
}
