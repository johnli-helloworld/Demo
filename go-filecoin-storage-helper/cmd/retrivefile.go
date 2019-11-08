package main

import (
	"context"
	"errors"
	"io/ioutil"
	"go-filecoin-storage-helper/filhttp"
	"go-filecoin-storage-helper/repo"
	"go-filecoin-storage-helper/utils"
	"gopkg.in/urfave/cli.v2"
)

var retrivecmd = &cli.Command{
	Name: "retrive",
	Usage: "retrive file",
	Flags:[]cli.Flag{
		&cli.StringFlag{
			Name: "cid",
			Usage:	"file cid",
		},
	},
	Action:	func(cctx *cli.Context) error {
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
		
		res, err := filhttp.Cat(ctx, cid)
		if err != nil {
			return errors.New("cat file err")
		}
		b, err := ioutil.ReadAll(res)

		metafilepath := utils.NewPath([]string{repopath, cid})
		f, err := utils.FileChecker(metafilepath)
		
		if err != nil {
			return errors.New("create file err")
		}
		if _, err := f.Write(b); err != nil {
			return errors.New("Incorrect source file")
		}
		return nil
	},
}