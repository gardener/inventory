package tasks

import (
	"context"
	"log/slog"

	taskregistry "github.com/gardener/inventory/pkg/core/registry/task"
	"github.com/hibiken/asynq"
)

// NewSampleTask creates a new Sample task
func NewSampleTask() (*asynq.Task, error) {
	task := asynq.NewTask("sample", nil)

	return task, nil
}

// HandleSampleTask handles our sample task
func HandleSampleTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("handling sample task")

	return nil
}

func init() {
	taskregistry.Default.MustRegister("aws:sample-task", asynq.HandlerFunc(HandleSampleTask))
}
