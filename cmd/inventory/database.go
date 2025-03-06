// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

// NewDatabaseCommand returns a new command for interfacing with the database.
func NewDatabaseCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "database",
		Usage:   "database operations",
		Aliases: []string{"db"},
		Subcommands: []*cli.Command{
			{
				Name:    "init",
				Usage:   "initialize migration tables",
				Aliases: []string{"i"},
				Action:  execDatabaseInitCmd,
			},
			{
				Name:    "migrate",
				Usage:   "apply pending migrations",
				Aliases: []string{"m"},
				Action:  execDatabaseMigrateCmd,
			},
			{
				Name:    "rollback",
				Usage:   "rollback last migration group",
				Aliases: []string{"r"},
				Action:  execDatabaseRollbackCmd,
			},
			{
				Name:    "lock",
				Usage:   "lock migrations",
				Aliases: []string{"l"},
				Action:  execDatabaseLockCmd,
			},
			{
				Name:    "unlock",
				Usage:   "unlock migrations",
				Aliases: []string{"u"},
				Action:  execDatabaseUnlockCmd,
			},
			{
				Name:    "create",
				Usage:   "create a new migration",
				Aliases: []string{"c"},
				Action:  execDatabaseCreateMigrationCmd,
			},
			{
				Name:    "status",
				Usage:   "display migration status",
				Aliases: []string{"s"},
				Action:  execDatabaseStatusCmd,
			},
			{
				Name:    "applied",
				Usage:   "display the list of applied migrations",
				Aliases: []string{"a"},
				Action:  execDatabaseAppliedCmd,
			},
			{
				Name:    "pending",
				Usage:   "display the list of pending migrations",
				Aliases: []string{"p"},
				Action:  execDatabasePendingCmd,
			},
		},
	}

	return cmd
}

// tabulateMigrations adds the given migration items to a table and returns it.
// The returned table can be further customized, if needed, and rendered.
func tabulateMigrations(items migrate.MigrationSlice) *tablewriter.Table {
	headers := []string{
		"ID",
		"NAME",
		"COMMENT",
		"GROUP-ID",
		"MIGRATED-AT",
	}
	table := newTableWriter(os.Stdout, headers)

	for _, item := range items {
		id := na
		groupID := na
		migratedAt := na

		if item.ID > 0 {
			id = strconv.FormatInt(item.ID, 10)
		}

		if item.GroupID > 0 {
			groupID = strconv.FormatInt(item.GroupID, 10)
		}

		if !item.MigratedAt.IsZero() {
			migratedAt = item.MigratedAt.String()
		}

		row := []string{
			id,
			item.Name,
			item.Comment,
			groupID,
			migratedAt,
		}
		table.Append(row)
	}

	return table
}

// execDatabaseInitCmd executes the command for initializing database schema.
func execDatabaseInitCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}

	return migrator.Init(ctx.Context)
}

// execDatabaseMigrateCmd runs the database migration command.
func execDatabaseMigrateCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}

	if err := migrator.Lock(ctx.Context); err != nil {
		return err
	}
	defer func() {
		err := migrator.Unlock(ctx.Context)
		if err != nil {
			slog.Error("failed to unlock migrations", "error", err)
		}
	}()

	group, err := migrator.Migrate(ctx.Context)
	if err != nil {
		return err
	}

	if group.IsZero() {
		fmt.Printf("database is up to date\n")
		return nil
	}

	fmt.Printf("database migrated to %s\n", group)
	return nil
}

// execDatabaseRollbackCmd executes the command for rolling back migrations.
func execDatabaseRollbackCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}

	if err := migrator.Lock(ctx.Context); err != nil {
		return err
	}

	defer func() {
		err := migrator.Unlock(ctx.Context)
		if err != nil {
			slog.Error("failed to unlock migrations", "error", err)
		}
	}()

	group, err := migrator.Rollback(ctx.Context)
	if err != nil {
		return err
	}

	if group.IsZero() {
		fmt.Printf("there are no migration groups for rollback\n")
		return nil
	}

	fmt.Printf("rolled back %s\n", group)
	return nil
}

// execDatabaseLockCmd executes the command for locking migrations.
func execDatabaseLockCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}

	return migrator.Lock(ctx.Context)
}

// execDatabaseUnlockCmd unlocks the database for migrations.
func execDatabaseUnlockCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}
	return migrator.Unlock(ctx.Context)
}

// execDatabaseCreateMigrationCmd creates a new migration sequence.
func execDatabaseCreateMigrationCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}

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
}

// execDatabaseStatusCmd runs the database migration status command.
func execDatabaseStatusCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}

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
}

// execDatabaseAppliedCmd runs the command for displaying applied migrations.
func execDatabaseAppliedCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}

	ms, err := migrator.MigrationsWithStatus(ctx.Context)
	if err != nil {
		return err
	}

	items := ms.Applied()
	if len(items) == 0 {
		return nil
	}

	table := tabulateMigrations(items)
	table.Render()

	return nil
}

// execDatabasePendingCmd displays the list of pending migrations.
func execDatabasePendingCmd(ctx *cli.Context) error {
	conf := getConfig(ctx)
	db, err := newDB(conf)
	if err != nil {
		return err
	}
	defer db.Close()
	migrator, err := newMigrator(conf, db)
	if err != nil {
		return err
	}

	ms, err := migrator.MigrationsWithStatus(ctx.Context)
	if err != nil {
		return err
	}

	items := ms.Unapplied()
	if len(items) == 0 {
		return nil
	}

	table := tabulateMigrations(items)
	table.Render()

	return nil
}
