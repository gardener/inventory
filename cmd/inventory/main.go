package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/version"
)

func main() {
	app := &cli.App{
		Name:                 "inventory",
		Version:              version.Version,
		EnableBashCompletion: true,
		Usage:                "command-line tool for managing the inventory",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "enables debug mode, if set",
				Value: false,
			},
		},
		Commands: []*cli.Command{},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
