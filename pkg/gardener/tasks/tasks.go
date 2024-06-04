package tasks

import (
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/core/registry"
)

func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_PROJECTS_TYPE, asynq.HandlerFunc(HandleGardenerCollectProjectsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_SEEDS_TYPE, asynq.HandlerFunc(HandleGardenerCollectSeedsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_SHOOTS_TYPE, asynq.HandlerFunc(HandleGardenerCollectShootsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_MACHINES_TYPE, asynq.HandlerFunc(HandleGardenerCollectMachinesTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_MACHINES_SEED_TYPE, asynq.HandlerFunc(HandleGardenerCollectMachinesForSeedTask))
}
