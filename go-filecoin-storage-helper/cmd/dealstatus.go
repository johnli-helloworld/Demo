package main

import (
	"context"
	"errors"
	"go-filecoin-storage-helper/meta"
	"go-filecoin-storage-helper/repo"

	"gopkg.in/urfave/cli.v2"
)

var dealstatuscmd = &cli.Command{
	Name:  "dealstatus",
	Usage: "query deal status",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "cid",
			Usage: "Query a storage deal's status",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()

		//首先需要通过cid获取元数据库
		cid := cctx.Args().Get(0)

		//校验是否初始化repo
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
		//生成元数据库
		m, err := meta.ReductionDS(ctx, cid)
		if err != nil {
			return err
		}

		//
		return nil
	},
}
