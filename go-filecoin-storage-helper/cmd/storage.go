package main

import (
	"context"
	"errors"
	"fmt"
	"go-filecoin-storage-helper/meta"
	"go-filecoin-storage-helper/repo"
	"go-filecoin-storage-helper/utils"
	"strconv"

	"gopkg.in/urfave/cli.v2"
)

var storagecmd = &cli.Command{
	Name:      "storage",
	Usage:     "storage data",
	ArgsUsage: "<filepath> <sectorsize> <miner> <duration>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "askid",
			Usage: "go-filecoin ask id",
		},
		&cli.StringFlag{
			Name:  "price",
			Usage: "lotus price",
		},
	},
	Action: func(cctx *cli.Context) error {
		fmt.Println("****************start storage************")
		ctx := context.Background()
		//首先校验参数
		if cctx.Args().Len() != 3 {
			fmt.Println("len====", cctx.Args().Len())
			fmt.Println("****************start stop************")
		}
		args := cctx.Args().Slice()
		filepath := args[0]
		fmt.Printf("filepath====%+v\n", filepath)
		filepath = "/home/john/firmware/john"

		ssize, err := strconv.ParseUint(args[1], 10, 64)
		fmt.Printf("ssize====%+v\n", ssize)
		ssize = 2048

		minerAddr := args[2]
		fmt.Printf("minerAddr====%+v\n", minerAddr)
		minerAddr = "sss"

		duration := args[3]
		fmt.Printf("duration====%+v\n", duration)
		duration = "2800"

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
		//获取元数据对象
		d, _ := strconv.ParseInt(duration, 10, 64)
		m, err := meta.NewMetaData(filepath, ssize, minerAddr, d)
		//生成元数据对象metads(对应每个待存储的文件生成一个唯一的datastore)
		//TODO: 元数据库命名：meta_xxxxxxx   存储元数据库信息：db_xxxxxx
		metaName := utils.GeneratedbName(m.Name)
		m.DbName = metaName
		s, err := meta.NewMetastore(repopath, "meta", metaName)
		if err != nil {
			return err
		}
		m.Mstore = s
		ctx = context.WithValue(ctx, repo.CtxRepoPath, repopath)
		dealinfo, err := m.Run(ctx)
		if err != nil {
			fmt.Println("state:", "failed")
		} else {
			fmt.Println("state:", dealinfo.State)
			fmt.Println("dealid:", dealinfo.Cid)
		}
		return nil
	},
}
