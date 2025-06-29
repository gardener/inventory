// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

// NewQueueCommand returns a new [cli.Command] for queue-related operations.
func NewQueueCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "queue",
		Usage:   "queue operations",
		Aliases: []string{"q"},
		Subcommands: []*cli.Command{
			{
				Name:    "list",
				Usage:   "list queues",
				Aliases: []string{"ls"},
				Action: func(ctx *cli.Context) error {
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck
					queues, err := inspector.Queues()
					if err != nil {
						return err
					}

					for _, item := range queues {
						fmt.Println(item)
					}

					return nil
				},
			},
			{
				Name:    "info",
				Usage:   "get queue info",
				Aliases: []string{"inspect", "i"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Usage:   "queue name",
						Value:   "default",
						Aliases: []string{"name"},
					},
				},
				Action: func(ctx *cli.Context) error {
					queueName := ctx.String("name")
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck
					q, err := inspector.GetQueueInfo(queueName)
					if err != nil {
						return err
					}

					fmt.Printf("%-20s: %s\n", "Name", q.Queue)
					fmt.Printf("%-20s: %d\n", "Memory Usage", q.MemoryUsage)
					fmt.Printf("%-20s: %s\n", "Latency", q.Latency.String())
					fmt.Printf("%-20s: %d\n", "Size", q.Size)
					fmt.Printf("%-20s: %d\n", "Groups", q.Groups)
					fmt.Printf("%-20s: %d\n", "Pending", q.Pending)
					fmt.Printf("%-20s: %d\n", "Active", q.Active)
					fmt.Printf("%-20s: %d\n", "Scheduled", q.Scheduled)
					fmt.Printf("%-20s: %d\n", "Retry", q.Retry)
					fmt.Printf("%-20s: %d\n", "Archived", q.Archived)
					fmt.Printf("%-20s: %d\n", "Completed", q.Completed)
					fmt.Printf("%-20s: %d\n", "Aggregating", q.Aggregating)
					fmt.Printf("%-20s: %d\n", "Processed (daily)", q.Processed)
					fmt.Printf("%-20s: %d\n", "Failed (daily)", q.Failed)
					fmt.Printf("%-20s: %s\n", "Is Paused", strconv.FormatBool(q.Paused))

					return nil
				},
			},
			{
				Name:    "pause",
				Usage:   "pause a queue",
				Aliases: []string{"p"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Usage:   "queue name",
						Value:   "default",
						Aliases: []string{"name"},
					},
				},
				Action: func(ctx *cli.Context) error {
					queueName := ctx.String("name")
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck

					return inspector.PauseQueue(queueName)
				},
			},
			{
				Name:    "resume",
				Usage:   "resume a queue",
				Aliases: []string{"r"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Usage:   "queue name",
						Value:   "default",
						Aliases: []string{"name"},
					},
				},
				Action: func(ctx *cli.Context) error {
					queueName := ctx.String("name")
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck

					return inspector.UnpauseQueue(queueName)
				},
			},
			{
				Name:    "drain",
				Usage:   "drain queue messages",
				Aliases: []string{"d"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "queue",
						Usage:   "queue name",
						Value:   "default",
						Aliases: []string{"name"},
					},
					&cli.StringFlag{
						Name:     "type",
						Usage:    "message type to drain",
						Value:    "scheduled",
						Required: true,
					},
				},
				Action: func(ctx *cli.Context) error {
					queueName := ctx.String("name")
					messageType := ctx.String("type")
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close() // nolint: errcheck

					typeToFunc := map[string]func(queue string) (int, error){
						"scheduled": inspector.DeleteAllScheduledTasks,
						"pending":   inspector.DeleteAllPendingTasks,
						"archived":  inspector.DeleteAllArchivedTasks,
						"completed": inspector.DeleteAllCompletedTasks,
						"retry":     inspector.DeleteAllRetryTasks,
					}

					deleteFunc, ok := typeToFunc[messageType]
					if !ok {
						messageTypes := make([]string, 0)
						for k := range typeToFunc {
							messageTypes = append(messageTypes, k)
						}

						return fmt.Errorf("message type should be one of %s", strings.Join(messageTypes, ", "))
					}

					_, err := deleteFunc(queueName)

					return err
				},
			},
		},
	}

	return cmd
}
