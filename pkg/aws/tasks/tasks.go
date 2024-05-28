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
	registry.TaskRegistry.MustRegister(AWS_COLLECT_REGIONS_TYPE, asynq.HandlerFunc(HandleAwsCollectRegionsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_AZS_TYPE, asynq.HandlerFunc(HandleCollectAzsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_AZS_REGION_TYPE, asynq.HandlerFunc(HandleCollectAzsRegionTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_VPC_TYPE, asynq.HandlerFunc(HandleCollectVpcsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_VPC_REGION_TYPE, asynq.HandlerFunc(HandleCollectVpcsRegionTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_SUBNETS_TYPE, asynq.HandlerFunc(HandleCollectSubnetsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_SUBNETS_REGION_TYPE, asynq.HandlerFunc(HandleCollectSubnetsRegionTask))

	// Periodic tasks
	sampleTask := NewSampleTask()
	registry.ScheduledTaskRegistry.MustRegister("@every 30s", sampleTask)
}
