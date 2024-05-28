package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"

	// Imports only for their side effects of registering our tasks
	_ "github.com/gardener/inventory/pkg/aws/tasks"
)

func NewTaskCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "task",
		Usage:   "task type",
		Aliases: []string{"t"},
		Subcommands: []*cli.Command{
			{
				Name:    "list",
				Usage:   "list registered tasks",
				Aliases: []string{"ls"},
				Action: func(ctx *cli.Context) error {
					tasks := make([]string, 0, registry.TaskRegistry.Length())
					registry.TaskRegistry.Range(func(name string, handler asynq.Handler) error {
						tasks = append(tasks, name)
						return nil
					})
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
					inspector := newInspectorFromFlags(ctx)
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
						Name:  "queue",
						Usage: "name of queue to use",
						Value: "default",
					},
				},
				Action: func(ctx *cli.Context) error {
					taskID := ctx.String("id")
					queue := ctx.String("queue")
					inspector := newInspectorFromFlags(ctx)
					return inspector.DeleteTask(queue, taskID)
				},
			},
			{
				Name:    "enqueue",
				Usage:   "submit a task",
				Aliases: []string{"submit"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "task",
						Usage:    "name of task to enqueue",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "payload",
						Usage: "path to a payload file",
					},
					&cli.StringFlag{
						Name:  "queue",
						Usage: "name of queue to use",
						Value: "default",
					},
				},
				Action: func(ctx *cli.Context) error {
					client := newAsynqClientFromFlags(ctx)
					defer client.Close()

					taskName := ctx.String("task")
					queue := ctx.String("queue")
					payloadFile := ctx.String("payload")

					_, ok := registry.TaskRegistry.Get(taskName)
					if !ok {
						return fmt.Errorf("Task %q not found in the registry", taskName)
					}

					var payload []byte
					if payloadFile != "" {
						data, err := os.ReadFile(payloadFile)
						if err != nil {
							return fmt.Errorf("Cannot read payload file: %w", err)
						}
						payload = data
					}

					task := asynq.NewTask(taskName, payload)
					info, err := client.EnqueueContext(ctx.Context, task, asynq.Queue(queue))
					if err != nil {
						return fmt.Errorf("Cannot enqueue %q task: %w", taskName, err)
					}

					fmt.Printf("%s/%s\n", info.Queue, info.ID)
					return nil
				},
			},
		},
	}

	return cmd
}
