package main

import (
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"runtime"
	"strconv"
)

func main() {
	// 默认参数
	concurrencyN := runtime.NumCPU()

	app := &cli.App{
		Name:  "downloader",
		Usage: "File concurrency downloader",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Usage:    "`URL` to download",
				Required: true,
				Aliases:  []string{"u"},
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output `filename`",
			},
			&cli.StringFlag{
				Name:    "concurrency",
				Aliases: []string{"n"},
				Value:   strconv.Itoa(concurrencyN),
				Usage:   "Concurrency `number`",
			},
		},
		After: func(c *cli.Context) error {
			strUrl := c.String("url")
			filename := c.String("output")
			concurrency := c.Int("concurrency")
			log.Printf("download file(%d) : %s \n", concurrency, strUrl)
			return NewDownloader(concurrency).Download(strUrl, filename)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
