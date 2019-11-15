package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-filecoin-storage-helper/repo"

	// "go-filecoin-storage-helper/utils"
	"io/ioutil"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

type DealStatus struct {
	DealId string
	State  string
}

func (m *MetaData) Deal(ctx context.Context, cid string) error {
	// repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	// if !ok {
	// 	return errors.New("ctx value repopath not found")
	// }
	//首先对记录元数据的db下单，若分多片，则往上还原db最后对元数据cid下单
	if err := m.Mstore.proposelstoragedeal(ctx, m.Miner, cid, m.AskId, m.Duration); err != nil {
		return err
	}
	//通过cid获取元数据库
	resp, err := StorageAPI.Cat(ctx, cid)
	if err != nil {
		return err
	}

	//TODO:对所有的元数据库及元数据下单

	_, err = ioutil.ReadAll(resp)
	if err != nil {
		return err
	}
	// dbgzpath := utils.NewPath([]string{repopath, "tmp.tar.gz"})

	if err != nil {
		return err
	}

	return nil
}

func (m *MetaData) deals(ctx context.Context, cid string) error {
	repopath, ok := ctx.Value(repo.CtxRepoPath).(string)
	if !ok {
		return errors.New("ctx value repopath not found")
	}
	res, err := m.Mstore.DS.Query(dsq.Query{Prefix: ChunkPrefix})
	if err != nil {
		return err
	}
	//这里做个优化，将cid都放进map里，若有重复的cid则可以去重
	cidmap := make(map[string]string)
	for {
		f, ok := res.NextSync()
		if !ok {
			break
		}
		cidmap[f.Key] = ""
	}

	ds, err := NewMetastore(repopath, "status", cid)
	defer ds.DS.Close()
	for cid := range cidmap {
		if err := ds.proposelstoragedeal(ctx, m.Miner, cid, m.AskId, m.Duration); err != nil {
			fmt.Println("proposelstoragedeal err:", err, " cid:", cid)
		}
	}
	return nil
}

func (ms *MetaStore) proposelstoragedeal(ctx context.Context, miner string, cid string, askid string, duration int64) error {
	d := &DealStatus{}
	resp, err := StorageAPI.ProposeStorageDeal(ctx, miner, cid, askid, duration)
	if err != nil {
		d.State = Failed
	} else {
		d.State = resp.State
		d.DealId = resp.DealId
	}
	dealenc, _ := json.Marshal(d)
	ms.DS.Put(datastore.NewKey(cid), dealenc)
	return nil
}

func (ms *MetaStore) QueryDealStatus(ctx context.Context) (map[string]*DealStatus, error) {
	//key: cid  value: dealid
	res, err := ms.DS.Query(dsq.Query{})
	if err != nil {
		return nil, err
	}
	statusmap := make(map[string]*DealStatus)
	for {
		f, ok := res.NextSync()
		if !ok {
			break
		}
		d := &DealStatus{}
		if err := json.Unmarshal(f.Value, d); err != nil {
			return nil, err
		}
		//查询所有的
		statusmap[string(f.Key)] = d
	}
	return statusmap, nil
}
