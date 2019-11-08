package meta

import (
	"go-filecoin-storage-helper/repo"
	"go-filecoin-storage-helper/utils"

	"github.com/ipfs/go-datastore"
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
