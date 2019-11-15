package meta

import (
	"go-filecoin-storage-helper/repo"
	"go-filecoin-storage-helper/utils"
	"strconv"

	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

type MetaStore struct {
	// MetaDS      datastore.Batching
	// DbDS        datastore.Batching
	DS          datastore.Batching
	FailedDeals map[string]chan struct{}
}

func NewMetastore(repopath string, dbtype string, ns string) (*MetaStore, error) {
	Fs, err := repo.NewFS(repopath)
	if err != nil {
		return nil, err
	}
	Fs.Path = utils.NewPath([]string{Fs.Path, dbtype})
	ds, err := Fs.Datastore(ns)
	if err != nil {
		return nil, err
	}
	return &MetaStore{
		DS:          ds,
		FailedDeals: map[string]chan struct{}{},
	}, nil
}

func NewMemstore() (*MetaStore, error) {
	ds, _ := repo.NewMemory().DataStore()
	return &MetaStore{
		DS:          ds,
		FailedDeals: map[string]chan struct{}{},
	}, nil
}

func (ds *MetaStore) Querystatus() (string, error) {
	dealStatus, err := ds.DS.Query(dsq.Query{})
	if err != nil {
		return "", err
	}
	status := Complete
	var t int
	for {
		f, ok := dealStatus.NextSync()
		if !ok {
			break
		}
		s := string(f.Value)
		tmpstatus, err := strconv.Atoi(s)
		if err != nil {
			return Failed, err
		}
		if (t - tmpstatus) < 0 {
			t = tmpstatus
			status = s
		}
	}
	return status, nil
}
