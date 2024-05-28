package main

import (
	"log/slog"
	"os"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"
)

// NewSchedulerCommand returns a new command for interfacing with the scheduler.
func NewSchedulerCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "scheduler",
		Usage:   "scheduler operations",
		Aliases: []string{"s"},
		Subcommands: []*cli.Command{
			{
				Name:    "start",
				Usage:   "start the scheduler",
				Aliases: []string{"s"},
				Action: func(ctx *cli.Context) error {
					scheduler := newSchedulerFromFlags(ctx)

					// Register our periodic tasks
					registry.ScheduledTaskRegistry.Range(func(spec string, task *asynq.Task) error {
						slog.Info("registering periodic task", "spec", spec, "name", task.Type())
						scheduler.Register(spec, task)
						return nil
					})

					if err := scheduler.Run(); err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:    "jobs",
				Usage:   "list periodic jobs",
				Aliases: []string{"j"},
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "page",
						Usage: "page number to retrieve",
						Value: 1,
					},
					&cli.IntFlag{
						Name:  "size",
						Usage: "page size to use",
						Value: 50,
					},
				},
				Action: func(ctx *cli.Context) error {
					inspector := newInspectorFromFlags(ctx)
					items, err := inspector.SchedulerEntries()
					if err != nil {
						return err
					}

					if len(items) == 0 {
						return nil
					}

					headers := []string{
						"ID",
						"SPEC",
						"TYPE",
						"PREV",
						"NEXT",
					}
					table := newTableWriter(os.Stdout, headers)
					for _, item := range items {
						row := []string{
							item.ID,
							item.Spec,
							item.Task.Type(),
							item.Prev.String(),
							item.Next.String(),
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
