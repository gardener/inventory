// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// NewTaskCommand returns a [cli.Command] for interfacing with task-related
// operations.
func NewTaskCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "task",
		Usage:   "task operations",
		Aliases: []string{"t"},
		Subcommands: []*cli.Command{
			{
				Name:    "list",
				Usage:   "list registered tasks",
				Aliases: []string{"ls"},
				Action: func(_ *cli.Context) error {
					tasks := make([]string, 0, registry.TaskRegistry.Length())
					walker := func(name string, _ asynq.Handler) error {
						tasks = append(tasks, name)

						return nil
					}

					if err := registry.TaskRegistry.Range(walker); err != nil {
						return err
					}

					sort.Strings(tasks)
					for _, task := range tasks {
						fmt.Println(task)
					}

					return nil
				},
			},
			{
				Name:    "cancel",
				Usage:   "cancel a running task",
				Aliases: []string{"c"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "task id",
						Required: true,
					},
				},
				Action: func(ctx *cli.Context) error {
					taskID := ctx.String("id")
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck

					return inspector.CancelProcessing(taskID)
				},
			},
			{
				Name:    "delete",
				Usage:   "delete a task",
				Aliases: []string{"d"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "task id",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
				},
				Action: func(ctx *cli.Context) error {
					taskID := ctx.String("id")
					queue := ctx.String("queue")
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck

					return inspector.DeleteTask(queue, taskID)
				},
			},
			{
				Name:    "active",
				Usage:   "list active tasks",
				Aliases: []string{"a"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
					&cli.IntFlag{
						Name:    "page",
						Aliases: []string{"p"},
						Usage:   "page number to retrieve",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "size",
						Aliases: []string{"s"},
						Usage:   "page size to use",
						Value:   50,
					},
				},
				Action: func(ctx *cli.Context) error {
					return printTasksInState(ctx, asynq.TaskStateActive)
				},
			},
			{
				Name:    "pending",
				Usage:   "list pending tasks",
				Aliases: []string{"p"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
					&cli.IntFlag{
						Name:    "page",
						Aliases: []string{"p"},
						Usage:   "page number to retrieve",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "size",
						Aliases: []string{"s"},
						Usage:   "page size to use",
						Value:   50,
					},
				},
				Action: func(ctx *cli.Context) error {
					return printTasksInState(ctx, asynq.TaskStatePending)
				},
			},
			{
				Name:    "archived",
				Usage:   "list archived tasks",
				Aliases: []string{"ar"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
					&cli.IntFlag{
						Name:    "page",
						Aliases: []string{"p"},
						Usage:   "page number to retrieve",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "size",
						Aliases: []string{"s"},
						Usage:   "page size to use",
						Value:   50,
					},
				},
				Action: func(ctx *cli.Context) error {
					return printTasksInState(ctx, asynq.TaskStateArchived)
				},
			},
			{
				Name:  "completed",
				Usage: "list completed tasks",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
					&cli.IntFlag{
						Name:    "page",
						Aliases: []string{"p"},
						Usage:   "page number to retrieve",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "size",
						Aliases: []string{"s"},
						Usage:   "page size to use",
						Value:   50,
					},
				},
				Action: func(ctx *cli.Context) error {
					return printTasksInState(ctx, asynq.TaskStateCompleted)
				},
			},
			{
				Name:    "retried",
				Usage:   "list retried tasks",
				Aliases: []string{"r"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
					&cli.IntFlag{
						Name:    "page",
						Aliases: []string{"p"},
						Usage:   "page number to retrieve",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "size",
						Aliases: []string{"s"},
						Usage:   "page size to use",
						Value:   50,
					},
				},
				Action: func(ctx *cli.Context) error {
					return printTasksInState(ctx, asynq.TaskStateRetry)
				},
			},
			{
				Name:    "scheduled",
				Usage:   "list scheduled tasks",
				Aliases: []string{"s"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
					&cli.IntFlag{
						Name:    "page",
						Aliases: []string{"p"},
						Usage:   "page number to retrieve",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "size",
						Aliases: []string{"s"},
						Usage:   "page size to use",
						Value:   50,
					},
				},
				Action: func(ctx *cli.Context) error {
					return printTasksInState(ctx, asynq.TaskStateScheduled)
				},
			},
			{
				Name:    "enqueue",
				Usage:   "submit a task",
				Aliases: []string{"submit"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "task",
						Aliases:  []string{"t"},
						Usage:    "name of task to enqueue",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "payload",
						Usage: "task payload",
					},
					&cli.PathFlag{
						Name:  "payload-file",
						Usage: "path to a payload file",
					},
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
					&cli.DurationFlag{
						Name:  "timeout",
						Usage: "set timeout for task",
						Value: 30 * time.Minute,
					},
				},
				Action: func(ctx *cli.Context) error {
					conf := getConfig(ctx)
					client := newAsynqClient(conf)
					defer client.Close() // nolint: errcheck

					taskName := ctx.String("task")
					timeout := ctx.Duration("timeout")
					queue := ctx.String("queue")

					var payload []byte
					payloadData := ctx.String("payload")
					payloadFile := ctx.Path("payload-file")
					switch {
					case payloadData != "" && payloadFile != "":
						return fmt.Errorf("cannot use --payload and --payload-file at the same time")
					case payloadData != "":
						payload = []byte(payloadData)
					case payloadFile != "":
						data, err := os.ReadFile(filepath.Clean(payloadFile))
						if err != nil {
							return fmt.Errorf("cannot read payload file: %w", err)
						}
						payload = data
					}

					task := asynq.NewTask(taskName, payload)
					opts := []asynq.Option{
						asynq.Queue(queue),
						asynq.Timeout(timeout),
					}
					info, err := client.EnqueueContext(ctx.Context, task, opts...)
					if err != nil {
						return fmt.Errorf("cannot enqueue %q task: %w", taskName, err)
					}

					fmt.Printf("%s/%s\n", info.Queue, info.ID)

					return nil
				},
			},
			{
				Name:    "inspect",
				Usage:   "inspect a task",
				Aliases: []string{"i"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Aliases: []string{"q"},
						Usage:   "name of queue to use",
						Value:   "default",
					},
					&cli.StringFlag{
						Name:     "id",
						Usage:    "task id",
						Required: true,
					},
				},
				Action: func(ctx *cli.Context) error {
					queueName := ctx.String("queue")
					taskID := ctx.String("id")
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck
					info, err := inspector.GetTaskInfo(queueName, taskID)
					if err != nil {
						return err
					}

					deadline := info.Deadline.String()
					if info.Deadline.IsZero() {
						deadline = na
					}

					lastFailedAt := info.LastFailedAt.String()
					if info.LastFailedAt.IsZero() {
						lastFailedAt = na
					}

					nextProcessAt := info.NextProcessAt.String()
					if info.NextProcessAt.IsZero() {
						nextProcessAt = na
					}

					completedAt := info.CompletedAt.String()
					if info.CompletedAt.IsZero() {
						completedAt = na
					}

					fmt.Printf("%-20s: %s\n", "ID", info.ID)
					fmt.Printf("%-20s: %s\n", "Queue", info.Queue)
					fmt.Printf("%-20s: %s\n", "Type/Name", info.Type)
					fmt.Printf("%-20s: %v\n", "State", info.State)
					fmt.Printf("%-20s: %v\n", "Group", info.Group)
					fmt.Printf("%-20s: %v\n", "Is Orphaned", strconv.FormatBool(info.IsOrphaned))

					fmt.Printf("%-20s: %d/%d\n", "Retry", info.Retried, info.MaxRetry)
					fmt.Printf("%-20s: %s\n", "Timeout", info.Timeout.String())
					fmt.Printf("%-20s: %s\n", "Deadline", deadline)
					fmt.Printf("%-20s: %s\n", "Retention", info.Retention.String())
					fmt.Printf("%-20s: %s\n", "Last Failed At", lastFailedAt)
					fmt.Printf("%-20s: %s\n", "Next Process At", nextProcessAt)
					fmt.Printf("%-20s: %s\n", "Completed At", completedAt)

					fmt.Printf("\nLast Error\n")
					fmt.Println("----------")
					fmt.Printf("%s\n", info.LastErr)

					fmt.Printf("\nPayload\n")
					fmt.Println("-------")
					if info.Payload != nil {
						fmt.Printf("%s\n", string(info.Payload))
					} else {
						fmt.Println("<nil>")
					}

					fmt.Printf("\nResult\n")
					fmt.Println("------")
					if info.Result != nil {
						fmt.Printf("%s\n", string(info.Result))
					} else {
						fmt.Println("<nil>")
					}

					return nil
				},
			},
		},
	}

	return cmd
}

// printTasksInState prints the tasks in the given state
func printTasksInState(ctx *cli.Context, state asynq.TaskState) error {
	page := ctx.Int("page")
	size := ctx.Int("size")
	queueName := ctx.String("queue")
	conf := getConfig(ctx)
	inspector := newInspector(conf)
	defer inspector.Close() // nolint: errcheck
	headers := []string{
		"ID",
		"TYPE",
		"RETRIED",
		"IS ORPHANED",
	}
	table := newTableWriter(os.Stdout, headers)

	stateToFunc := map[asynq.TaskState]func(queue string, opts ...asynq.ListOption) ([]*asynq.TaskInfo, error){
		asynq.TaskStateActive:    inspector.ListActiveTasks,
		asynq.TaskStatePending:   inspector.ListPendingTasks,
		asynq.TaskStateArchived:  inspector.ListArchivedTasks,
		asynq.TaskStateCompleted: inspector.ListCompletedTasks,
		asynq.TaskStateRetry:     inspector.ListRetryTasks,
		asynq.TaskStateScheduled: inspector.ListScheduledTasks,
	}

	getFunc, ok := stateToFunc[state]
	if !ok {
		return fmt.Errorf("unknown task state: %v", state)
	}

	items, err := getFunc(queueName, asynq.Page(page), asynq.PageSize(size))
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		row := []string{
			item.ID,
			item.Type,
			fmt.Sprintf("%d/%d", item.Retried, item.MaxRetry),
			strconv.FormatBool(item.IsOrphaned),
		}
		if err := table.Append(row); err != nil {
			return err
		}
	}

	return table.Render()
}
