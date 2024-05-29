package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/version"
)

func main() {
	app := &cli.App{
		Name:                 "inventory",
		Version:              version.Version,
		EnableBashCompletion: true,
		Usage:                "command-line tool for managing the inventory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Usage:    "path to config file",
				Required: true,
				Aliases:  []string{"file"},
				EnvVars:  []string{"INVENTORY_CONFIG"},
			},
		},
		Before: func(ctx *cli.Context) error {
			configFile := ctx.String("config")
			conf, err := config.Parse(configFile)
			if err != nil {
				return fmt.Errorf("Cannot parse config: %w", err)
			}
			ctx.Context = context.WithValue(ctx.Context, configKey{}, conf)
			return nil
		},
		Commands: []*cli.Command{
			NewDatabaseCommand(),
			NewWorkerCommand(),
			NewSchedulerCommand(),
			NewTaskCommand(),
			NewQueueCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
