package main

import (
	"log"

	"go-filecoin-storage-helper/repo"

	"gopkg.in/urfave/cli.v2"
)

var initcmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize filecoin storage helper repo",
	Action: func(cctx *cli.Context) error {
		repoPath := cctx.String(FlagStorageHelperRepo)
		r, err := repo.NewFS(repoPath)
		if err != nil {
			return err
		}

		log.Println("Initializing storage helper repo")
		if err := r.Init(); err != nil {
			log.Println("Initializing storage helper err")
			return err
		}
		//生成meta目录，存放badger的数据
		if err := r.GenerateMetaDir(); err != nil {
			log.Println("Initializing meta dir err")
			return err
		}
		if err := r.GenerateDbDir(); err != nil {
			log.Println("Initializing db dir err")
			return err
		}
		return nil
	},
}
