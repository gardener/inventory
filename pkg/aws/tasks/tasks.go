package tasks

import (
	"context"
	"log/slog"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
)

const (
	// Asqynq task type for collecting AWS resources
	AWS_COLLECT_VPC_TYPE       = "aws:collect-vpcs"
	AWS_COLLECT_SUBNETS_TYPE   = "aws:collect-subnets"
	AWS_COLLECT_INSTANCES_TYPE = "aws:collect-instances"

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
	registry.TaskRegistry.MustRegister(AWS_COLLECT_REGIONS_TYPE, asynq.HandlerFunc(HandleAwsCollectRegionsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_AZS_TYPE, asynq.HandlerFunc(HandleAwsCollectAzsTask))

	// Periodic tasks
	sampleTask := NewSampleTask()
	registry.ScheduledTaskRegistry.MustRegister("@every 30s", sampleTask)
}
