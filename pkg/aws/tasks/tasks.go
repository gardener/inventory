package tasks

import (
	"context"
	"log/slog"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
)

const (
	// sampleTaskName is the name for the sample task
	sampleTaskName = "aws:sample-task"
)

// NewSampleTask creates a new Sample task
func NewSampleTask() *asynq.Task {
	task := asynq.NewTask(sampleTaskName, nil)

	return task
}

// HandleSampleTask handles our sample task
func HandleSampleTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("handling sample task")

	return nil
}

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(sampleTaskName, asynq.HandlerFunc(HandleSampleTask))

	// Periodic tasks
	sampleTask := NewSampleTask()
	registry.ScheduledTaskRegistry.MustRegister("@every 30s", sampleTask)
}
