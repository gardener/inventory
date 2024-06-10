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
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "enables debug mode, if set",
				Value: false,
			},
			&cli.StringFlag{
				Name:     "config",
				Usage:    "path to config file",
				Required: true,
				Aliases:  []string{"file"},
				EnvVars:  []string{"INVENTORY_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "redis-endpoint",
				Usage:   "redis endpoint to connect to",
				EnvVars: []string{"REDIS_ENDPOINT"},
			},
			&cli.StringFlag{
				Name:    "database-uri",
				Usage:   "database uri to connect to",
				EnvVars: []string{"DATABASE_URI"},
			},
			&cli.StringFlag{
				Name:    "environment",
				Usage:   "garden environment",
				EnvVars: []string{"ENVIRONMENT"},
				Aliases: []string{"env"},
			},
		},
		Before: func(ctx *cli.Context) error {
			configFile := ctx.String("config")
			conf, err := config.Parse(configFile)
			if err != nil {
				return fmt.Errorf("Cannot parse config: %w", err)
			}

			// Overrides from flags/options
			if ctx.IsSet("debug") {
				conf.Debug = ctx.Bool("debug")
			}

			if ctx.IsSet("redis-endpoint") {
				conf.Redis.Endpoint = ctx.String("redis-endpoint")
			}

			if ctx.IsSet("database-uri") {
				conf.Database.DSN = ctx.String("database-uri")
			}

			if ctx.IsSet("environment") {
				conf.VirtualGarden.Environment = ctx.String("environment")
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
			NewModelCommand(),
			NewDashboardCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
