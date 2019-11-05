package main

import (
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
		//首先需要通过dealid获取元数据库

		return nil
	},
}
