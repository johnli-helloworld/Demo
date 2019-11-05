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
		ctx := context.Background()
		//首先校验参数
		if cctx.Args().Len() != 3 {
			return fmt.Errorf("")
		}
		args := cctx.Args().Slice()
		filepath := args[0]

		ssize, err := strconv.ParseUint(args[1], 10, 64)

		minerAddr := args[2]
		fmt.Printf("addr====%+v\n", minerAddr)

		duration := args[3]

		repoPath := cctx.String(FlagStorageHelperRepo)
		r, err := repo.NewFS(repoPath)
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
		m, err := meta.NewMetaData(filepath, ssize, minerAddr, strconv.ParseInt(duration,10,64))
		//分片大小
		m.ChunkSize = ssize
		//生成元数据对象datastore(对应每个待存储的文件生成一个唯一的datastore)
		//TODO: 元数据库命名：meta_xxxxxxx   存储元数据库信息：db_xxxxxx
		metaName := utils.GeneratedbName(filepath)
		m.DS, err = r.Datastore(metaName)
		if err != nil {
			return err
		}
		m.Run(ctx)
		return nil
	},
}
