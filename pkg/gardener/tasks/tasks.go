package tasks

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/common/utils"
	"github.com/gardener/inventory/pkg/core/registry"
)

const (
	// GardenerCollectAllTaskType is a meta task, which enqueues all
	// relevant Gardener tasks.
	GardenerCollectAllTaskType = "g:task:collect-all"
)

func HandleCollectAllTask(ctx context.Context, t *asynq.Task) error {
	// Task constructors
	taskFns := []utils.TaskConstructor{
		NewGardenerCollectProjectsTask,
		NewGardenerCollectSeedsTask,
		NewGardenerCollectShootsTask,
		NewGardenerCollectMachinesTask,
	}

	return utils.Enqueue(taskFns)
}

func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_PROJECTS_TYPE, asynq.HandlerFunc(HandleGardenerCollectProjectsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_SEEDS_TYPE, asynq.HandlerFunc(HandleGardenerCollectSeedsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_SHOOTS_TYPE, asynq.HandlerFunc(HandleGardenerCollectShootsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_MACHINES_TYPE, asynq.HandlerFunc(HandleGardenerCollectMachinesTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_MACHINES_SEED_TYPE, asynq.HandlerFunc(HandleGardenerCollectMachinesForSeedTask))
	registry.TaskRegistry.MustRegister(GardenerCollectAllTaskType, asynq.HandlerFunc(HandleCollectAllTask))
}
