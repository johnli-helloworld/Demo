package main

import (
	"context"
	"errors"
	"fmt"
	"go-filecoin-storage-helper/meta"
	"go-filecoin-storage-helper/repo"
	"go-filecoin-storage-helper/utils"

	"gopkg.in/urfave/cli.v2"
)

var importcmd = &cli.Command{
	Name:  "import",
	Usage: "import data",
	// ArgsUsage: "<filepath>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "filepath",
			Usage: "file abspath",
		},
		&cli.StringFlag{
			Name:  "sectorsize",
			Usage: "file size",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()

		filepath := "/home/ars/johntest/john"
		sectorsize := uint64(2048)
		repopath := cctx.String(FlagStorageHelperRepo)
		r, err := repo.NewFS(repopath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("repo at is not initialized, run 'filecoin-storage-helper init' to set it up")
		}
		ctx = context.WithValue(ctx, repo.CtxRepoPath, repopath)

		//生成meta/dbfile目录，存放badger/dbfile的数据
		if err := r.GenerateMetaDir(); err != nil {
			return err
		}
		if err := r.GenerateDbDir(); err != nil {
			return err
		}
		m, err := meta.NewMetaData(filepath, "meta")
		if err != nil {
			return err
		}
		m.SectorSize = sectorsize
		metaName := utils.GeneratedbName(m.Abspath)
		m.DbName = metaName
		s, err := meta.NewMetastore(repopath, "meta", metaName)
		if err != nil {
			return err
		}
		m.Mstore = s

		cid, err := m.Import(ctx)
		if err != nil {
			fmt.Println("import failed", err)
		}
		fmt.Println("cid:", cid)
		return nil
	},
}
