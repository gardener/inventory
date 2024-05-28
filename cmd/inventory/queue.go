package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

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
					inspector := newInspectorFromFlags(ctx)
					queues, err := inspector.Queues()
					if err != nil {
						return err
					}

					if len(queues) == 0 {
						return nil
					}

					table := newTableWriter(os.Stdout, []string{"NAME"})
					for _, item := range queues {
						table.Append([]string{item})
					}

					table.Render()
					return nil
				},
			},
			{
				Name:    "info",
				Usage:   "get queue info",
				Aliases: []string{"i"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "queue",
						Usage:    "queue name",
						Value:    "default",
						Required: true,
						Aliases:  []string{"name"},
					},
				},
				Action: func(ctx *cli.Context) error {
					queueName := ctx.String("name")
					inspector := newInspectorFromFlags(ctx)
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
					fmt.Printf("%-20s: %v\n", "Paused", q.Paused)

					return nil
				},
			},
			{
				Name:    "pause",
				Usage:   "pause a queue",
				Aliases: []string{"p"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "queue",
						Usage:    "queue name",
						Value:    "default",
						Required: true,
						Aliases:  []string{"name"},
					},
				},
				Action: func(ctx *cli.Context) error {
					queueName := ctx.String("name")
					inspector := newInspectorFromFlags(ctx)
					return inspector.PauseQueue(queueName)
				},
			},
			{
				Name:    "resume",
				Usage:   "resume a queue",
				Aliases: []string{"r"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "queue",
						Usage:    "queue name",
						Value:    "default",
						Required: true,
						Aliases:  []string{"name"},
					},
				},
				Action: func(ctx *cli.Context) error {
					queueName := ctx.String("name")
					inspector := newInspectorFromFlags(ctx)
					return inspector.UnpauseQueue(queueName)
				},
			},
			{
				Name:    "drain",
				Usage:   "drain queue messages",
				Aliases: []string{"d"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "queue",
						Usage:    "queue name",
						Value:    "default",
						Required: true,
						Aliases:  []string{"name"},
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
					inspector := newInspectorFromFlags(ctx)

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
						return fmt.Errorf("Message type should be one of %s", strings.Join(messageTypes, ", "))
					}

					_, err := deleteFunc(queueName)
					return err
				},
			},
		},
	}

	return cmd
}
