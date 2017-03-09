package main

import (
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli"
)

func main() {
	a := cli.NewApp()
	a.Version = "0.1.0"
	a.Usage = "magic crud and restful api for experimenting with ql database"
	a.Authors = []cli.Author{
		{
			Name:  "Geofrey Ernest",
			Email: "geofreyernest@live.com",
		},
	}
	a.Commands = []cli.Command{
		{
			Name:   "serve",
			Usage:  "automated crud & resful api  on ql database",
			Action: serv,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "dir",
					Usage:  "directory where ql database files are stored",
					Value:  "_qlfu",
					EnvVar: "QLFU_DIR",
				},
				cli.StringFlag{
					Name:   "baseurl",
					Usage:  "directory where ql database files are stored",
					Value:  "http://localhost:8090",
					EnvVar: "QLFU_BASEURL",
				},
			},
		},
	}
	err := a.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func serv(ctx *cli.Context) error {
	dir := ctx.String("dir")
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}
	base := ctx.String("baseurl")
	a, err := newAPI(dir, base)
	if err != nil {
		return err
	}
	return http.ListenAndServe(":8090", a)
}
