// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/pkg/core/registry"
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
					conf := getConfig(ctx)
					scheduler := newScheduler(conf)

					// Add the periodic tasks from the registry
					walker := func(spec string, task *asynq.Task) error {
						// TODO(dnaeon): add support for specifying queue for tasks
						// originating from the registry.
						queue := conf.Scheduler.DefaultQueue
						id, err := scheduler.Register(
							spec,
							task,
							asynq.Queue(queue),
						)
						if err != nil {
							return err
						}
						slog.Info(
							"periodic task registered",
							"id", id,
							"name", task.Type(),
							"spec", spec,
							"queue", queue,
							"source", "registry",
						)
						return nil
					}
					if err := registry.ScheduledTaskRegistry.Range(walker); err != nil {
						return err
					}

					// Add tasks from configuration file as well
					for _, job := range conf.Scheduler.Jobs {
						task := asynq.NewTask(job.Name, []byte(job.Payload))
						queue := conf.Scheduler.DefaultQueue
						if job.Queue != "" {
							queue = job.Queue
						}

						id, err := scheduler.Register(job.Spec, task, asynq.Queue(queue))
						if err != nil {
							return err
						}

						slog.Info(
							"periodic task registered",
							"id", id,
							"name", task.Type(),
							"spec", job.Spec,
							"desc", job.Desc,
							"queue", queue,
							"source", "config",
						)
					}

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
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck
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
						"OPTS",
					}

					table := newTableWriter(os.Stdout, headers)
					for _, item := range items {
						nextIn := time.Until(item.Next)
						prev := item.Prev.String()
						if item.Prev.IsZero() {
							prev = na
						}

						opts := make([]string, 0)
						for _, opt := range item.Opts {
							opts = append(opts, opt.String())
						}

						row := []string{
							item.ID,
							item.Spec,
							item.Task.Type(),
							prev,
							fmt.Sprintf("In %s", nextIn.String()),
							strings.Join(opts, ", "),
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
