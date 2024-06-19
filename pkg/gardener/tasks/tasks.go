// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/common/utils"
	"github.com/gardener/inventory/pkg/core/registry"
)

const (
	// GardenerCollectAllTaskType is a meta task, which enqueues all
	// relevant Gardener tasks.
	GardenerCollectAllTaskType = "g:task:collect-all"

	// GardenerLinkAllTaskType is the task type for linking all Gardener
	// related objects.
	GardenerLinkAllTaskType = "g:task:link-all"
)

// HandleCollectAllTask is the handler, which enqueues tasks for collecting all
// known Gardener resources.
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

// HandleLinkAllTask is the handler, which establishes relationships between the
// various Gardener models.
func HandleLinkAllTask(ctx context.Context, r *asynq.Task) error {
	linkFns := []utils.LinkFunction{
		LinkShootWithProject,
		LinkShootWithSeed,
		LinkMachineWithShoot,
	}

	return utils.LinkObjects(ctx, clients.DB, linkFns)
}

func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_PROJECTS_TYPE, asynq.HandlerFunc(HandleGardenerCollectProjectsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_SEEDS_TYPE, asynq.HandlerFunc(HandleGardenerCollectSeedsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_SHOOTS_TYPE, asynq.HandlerFunc(HandleGardenerCollectShootsTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_MACHINES_TYPE, asynq.HandlerFunc(HandleGardenerCollectMachinesTask))
	registry.TaskRegistry.MustRegister(GARDENER_COLLECT_MACHINES_SEED_TYPE, asynq.HandlerFunc(HandleGardenerCollectMachinesForSeedTask))
	registry.TaskRegistry.MustRegister(GardenerCollectAllTaskType, asynq.HandlerFunc(HandleCollectAllTask))
	registry.TaskRegistry.MustRegister(GardenerLinkAllTaskType, asynq.HandlerFunc(HandleLinkAllTask))
}
