package main

import (
	"context"
	"errors"
	"go-filecoin-storage-helper/meta"
	"go-filecoin-storage-helper/repo"

	"gopkg.in/urfave/cli.v2"
)

var proposaldealcmd = &cli.Command{
	Name:  "proposaldeal",
	Usage: "make deal file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "cid",
			Usage: "file cid",
		},
		&cli.StringFlag{
			Name:  "miner",
			Usage: "storage miner",
		},
		&cli.StringFlag{
			Name:  "duration",
			Usage: "duration",
		},
		&cli.StringFlag{
			Name:  "askid",
			Usage: "askid",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		cid := cctx.Args().Get(0)

		miner := cctx.Args().Get(1)

		duration := cctx.Args().Get(2)

		askid := cctx.Args().Get(3)

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
		// ds, err := meta.ReductDS(ctx, cid, target)
		// // fmt.Println("retrive s============", s)
		// if err != nil {
		// 	fmt.Println("err===========", err)
		// 	return err
		// }
		// if err = ds.ReductFile(ctx, target); err != nil {
		// 	return err
		// }

		meta.Deal(ctx, cid)
		return nil
	},
}
