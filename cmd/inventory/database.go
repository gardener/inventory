package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gardener/inventory/internal/pkg/migrations"
	"github.com/olekukonko/tablewriter"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
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
			&cli.StringFlag{
				Name:    "migration-dir",
				Usage:   "path to the directory with migration files",
				EnvVars: []string{"MIGRATION_DIR"},
			},
		},
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "initialize migration tables",
				Action: func(ctx *cli.Context) error {
					db := newDBFromFlags(ctx)
					migrator := newMigratorFromFlags(ctx, db)
					return migrator.Init(ctx.Context)
				},
			},
			{
				Name:  "migrate",
				Usage: "apply pending migrations",
				Action: func(ctx *cli.Context) error {
					db := newDBFromFlags(ctx)
					migrator := newMigratorFromFlags(ctx, db)
					if err := migrator.Lock(ctx.Context); err != nil {
						return err
					}
					defer migrator.Unlock(ctx.Context)

					group, err := migrator.Migrate(ctx.Context)
					if err != nil {
						return err
					}

					if group.IsZero() {
						fmt.Printf("database is up to date")
						return nil
					}

					fmt.Printf("database migrated to %s\n", group)
					return nil
				},
			},
			{
				Name:  "rollback",
				Usage: "rollback last migration group",
				Action: func(ctx *cli.Context) error {
					db := newDBFromFlags(ctx)
					migrator := newMigratorFromFlags(ctx, db)
					if err := migrator.Lock(ctx.Context); err != nil {
						return err
					}
					defer migrator.Unlock(ctx.Context)

					group, err := migrator.Rollback(ctx.Context)
					if err != nil {
						return err
					}

					if group.IsZero() {
						fmt.Printf("there are no migration groups for rollback")
						return nil
					}

					fmt.Printf("rolled back %s\n", group)
					return nil
				},
			},
			{
				Name:  "lock",
				Usage: "lock migrations",
				Action: func(ctx *cli.Context) error {
					db := newDBFromFlags(ctx)
					migrator := newMigratorFromFlags(ctx, db)
					return migrator.Lock(ctx.Context)
				},
			},
			{
				Name:  "unlock",
				Usage: "unlock migrations",
				Action: func(ctx *cli.Context) error {
					db := newDBFromFlags(ctx)
					migrator := newMigratorFromFlags(ctx, db)
					return migrator.Unlock(ctx.Context)
				},
			},
			{
				Name:  "create",
				Usage: "create a new migration",
				Action: func(ctx *cli.Context) error {
					db := newDBFromFlags(ctx)
					migrator := newMigratorFromFlags(ctx, db)
					name := strings.Join(ctx.Args().Slice(), "_")
					if name == "" {
						return errors.New("must specify migration description")
					}

					files, err := migrator.CreateTxSQLMigrations(ctx.Context, name)
					if err != nil {
						return err
					}

					for _, item := range files {
						fmt.Println(item.Path)
					}

					return nil
				},
			},
			{
				Name:  "status",
				Usage: "display migration status",
				Action: func(ctx *cli.Context) error {
					db := newDBFromFlags(ctx)
					migrator := newMigratorFromFlags(ctx, db)
					ms, err := migrator.MigrationsWithStatus(ctx.Context)
					if err != nil {
						return err
					}

					pending := ms.Unapplied()
					group := ms.LastGroup()

					fmt.Printf("pending migration(s): %d\n", len(pending))
					fmt.Printf("database version: %s\n", group)

					if len(pending) == 0 {
						fmt.Println("database is up-to-date")
					} else {
						fmt.Println("database is out-of-date")
					}

					return nil
				},
			},
			{
				Name:  "applied",
				Usage: "display the applied migrations",
				Action: func(ctx *cli.Context) error {
					db := newDBFromFlags(ctx)
					migrator := newMigratorFromFlags(ctx, db)
					ms, err := migrator.MigrationsWithStatus(ctx.Context)
					if err != nil {
						return err
					}

					applied := ms.Applied()
					if len(applied) == 0 {
						return nil
					}

					table := tablewriter.NewWriter(os.Stdout)
					headers := []string{
						"ID",
						"NAME",
						"COMMENT",
						"GROUP-ID",
						"MIGRATED-AT",
					}
					table.SetHeader(headers)
					table.SetAutoWrapText(false)
					table.SetBorder(false)

					for _, item := range applied {
						row := []string{
							strconv.FormatInt(item.ID, 10),
							item.Name,
							item.Comment,
							strconv.FormatInt(item.GroupID, 10),
							item.MigratedAt.String(),
						}
						table.Append(row)
					}

					table.Render()
					return nil
				},
			},
		},
	}

	return cmd
}

// newDbFromFlags returns a Bun database from the specified flags
func newDBFromFlags(ctx *cli.Context) *bun.DB {
	dsn := ctx.String("dsn")
	debug := ctx.Bool("debug")

	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(pgdb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(debug)))

	return db
}

// newMigratorFromFlags returns a new [github.com/uptrace/bun/migrate.Migrator]
// from the specified flags.
func newMigratorFromFlags(ctx *cli.Context, db *bun.DB) *migrate.Migrator {
	// By default we will use the bundled migrations, unless we have an
	// explicitely specified alternate migrations directory.
	m := migrations.Migrations
	migrationDir := ctx.String("migration-dir")
	if migrationDir != "" {
		m = migrate.NewMigrations(migrate.WithMigrationsDirectory(migrationDir))
	}

	return migrate.NewMigrator(db, m)
}
