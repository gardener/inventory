package tasks

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/common/utils"
	"github.com/gardener/inventory/pkg/core/registry"
)

const (
	// AWSCollectAllTaskType is a meta task, which enqueues all relevant AWS
	// tasks.
	AWSCollectAllTaskType = "aws:task:collect-all"
)

// HandleAWSCollectAllTask is a handler, which enqueues tasks for collecting all
// AWS objects.
func HandleAWSCollectAllTask(ctx context.Context, t *asynq.Task) error {
	// Task constructors
	taskFns := []utils.TaskConstructor{
		NewCollectRegionsTask,
		NewCollectAzsTask,
		NewCollectVpcsTask,
		NewCollectSubnetsTask,
		NewCollectInstancesTask,
	}

	return utils.Enqueue(taskFns)
}

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(AWS_COLLECT_REGIONS_TYPE, asynq.HandlerFunc(HandleAwsCollectRegionsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_AZS_TYPE, asynq.HandlerFunc(HandleCollectAzsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_AZS_REGION_TYPE, asynq.HandlerFunc(HandleCollectAzsForRegionTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_VPC_TYPE, asynq.HandlerFunc(HandleCollectVpcsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_VPC_REGION_TYPE, asynq.HandlerFunc(HandleCollectVpcsForRegionTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_SUBNETS_TYPE, asynq.HandlerFunc(HandleCollectSubnetsTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_SUBNETS_REGION_TYPE, asynq.HandlerFunc(HandleCollectSubnetsForRegionTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_INSTANCES_TYPE, asynq.HandlerFunc(HandleCollectInstancesTask))
	registry.TaskRegistry.MustRegister(AWS_COLLECT_INSTANCES_REGION_TYPE, asynq.HandlerFunc(HandleCollectInstancesForRegionTask))
	registry.TaskRegistry.MustRegister(AWSCollectAllTaskType, asynq.HandlerFunc(HandleAWSCollectAllTask))
}
