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
			return err
		}
		return nil
	},
}
