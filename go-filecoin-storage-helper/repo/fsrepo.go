package repo

import (
	"os"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/mitchellh/go-homedir"
)

type FsRepo struct {
	path string
}

func NewFS(path string) (*FsRepo, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	return &FsRepo{
		path: path,
	}, nil
}

func (fsr *FsRepo) Exists() (bool, error) {
	_, err := os.Stat(fsr.path)
	notexist := os.IsNotExist(err)
	if notexist {
		err = nil
	}
	return !notexist, err
}

func (fsr *FsRepo) Init() error {
	exist, err := fsr.Exists()
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	err = os.Mkdir(fsr.path, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (fsr *FsRepo) Datastore(ns string) (datastore.Batching, error) {
	opts := badger.DefaultOptions
	opts.Truncate = true
	ds, err := badger.NewDatastore(filepath.Join(fsr.path, ns), &opts)
	if err != nil {
		return nil, err
	}
	return ds, err
}
