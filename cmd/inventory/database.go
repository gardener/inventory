package main

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/urfave/cli/v2"
)

// NewDatabaseCommand returns a new command for interfacing with the database.
func NewDatabaseCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "database",
		Usage:   "database operations",
		Aliases: []string{"db"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dsn",
				Usage:    "DSN to connect to",
				EnvVars:  []string{"DSN"},
				Required: true,
			},
		},
		Subcommands: []*cli.Command{},
	}

	return cmd
}

// dbFromFlags returns a Bun database from the specified flags
func dbFromFlags(ctx *cli.Context) *bun.DB {
	dsn := ctx.String("dsn")
	debug := ctx.Bool("debug")

	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(pgdb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(debug)))

	return db
}
