package main

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v2"
)

const FlagStorageHelperRepo = "storagehelperrepo"

var CtxRepoPath string

func main() {
	local := []*cli.Command{
		initcmd,
		storagecmd,
		dealstatuscmd,
		retrivecmd,
	}
	app := &cli.App{
		Name:    "filecoin-storage-helper",
		Usage:   "make deal, storage file/directory",
		Version: "v0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: FlagStorageHelperRepo,
				// EnvVars:	[]string
				Value: "/home/john/.filstoragehelper",
			},
		},
		Commands: local,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		return
	}
}
