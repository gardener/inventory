package utils

import (
	"context"
	"log/slog"

	"github.com/gardener/inventory/pkg/clients"
	"github.com/hibiken/asynq"
	"github.com/uptrace/bun"
)

// TaskConstructor is a function which creates and returns a new [asynq.Task].
type TaskConstructor func() *asynq.Task

// Enqueue enqueues the tasks produced by the given task constructors.
func Enqueue(items []TaskConstructor) error {
	for _, fn := range items {
		task := fn()
		info, err := clients.Client.Enqueue(task)
		if err != nil {
			slog.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"reason", err,
			)
			return err
		}

		slog.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
		)
	}

	return nil
}

// LinkFunction is a function, which establishes relationships between models.
type LinkFunction func(ctx context.Context, db *bun.DB) error

// LinkObjects links objects by using the provided [LinkFunction] items.
func LinkObjects(ctx context.Context, db *bun.DB, items []LinkFunction) error {
	for _, linkFunc := range items {
		if err := linkFunc(ctx, db); err != nil {
			slog.Error("failed to link objects", "reason", err)
			return err
		}
	}

	return nil
}
