package main

import (
	"context"
	"errors"
	"fmt"
	"go-filecoin-storage-helper/meta"
	"go-filecoin-storage-helper/repo"

	"gopkg.in/urfave/cli.v2"
)

var retrivecmd = &cli.Command{
	Name:  "retrive",
	Usage: "retrive file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "cid",
			Usage: "file cid",
		},
		&cli.StringFlag{
			Name:  "target",
			Usage: "file target path",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		//首先需要通过cid获取元数据库
		cid := cctx.Args().Get(0)
		fmt.Println("cid============", cid)
		target := cctx.Args().Get(1)
		fmt.Println("target============", target)
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

		meta.RunReduct(ctx, cid, target)
		return nil
	},
}
