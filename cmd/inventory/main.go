package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/pkg/version"
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
			&cli.StringFlag{
				Name:     "redis-endpoint",
				Usage:    "Redis endpoint to connect to",
				EnvVars:  []string{"REDIS_ENDPOINT"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "dsn",
				Usage:    "DSN to connect to",
				EnvVars:  []string{"DSN"},
				Required: true,
				Aliases:  []string{"database"},
			},
		},
		Commands: []*cli.Command{
			NewDatabaseCommand(),
			NewWorkerCommand(),
			NewSchedulerCommand(),
			NewTaskCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
