// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/common/utils"
	"github.com/gardener/inventory/pkg/core/registry"
)

const (
	// TaskCollectAll is a meta task, which enqueues all
	// relevant Gardener tasks.
	TaskCollectAll = "g:task:collect-all"

	// TaskLinkAll is the task type for linking all Gardener
	// related objects.
	TaskLinkAll = "g:task:link-all"
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
		NewGardenerCollectBackupBucketsTask,
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

	return utils.LinkObjects(ctx, db.DB, linkFns)
}

func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(TaskCollectProjects, asynq.HandlerFunc(HandleGardenerCollectProjectsTask))
	registry.TaskRegistry.MustRegister(TaskCollectSeeds, asynq.HandlerFunc(HandleGardenerCollectSeedsTask))
	registry.TaskRegistry.MustRegister(TaskCollectShoots, asynq.HandlerFunc(HandleGardenerCollectShootsTask))
	registry.TaskRegistry.MustRegister(TasksCollectMachines, asynq.HandlerFunc(HandleGardenerCollectMachinesTask))
	registry.TaskRegistry.MustRegister(TaskCollectMachinesForSeed, asynq.HandlerFunc(HandleGardenerCollectMachinesForSeedTask))
	registry.TaskRegistry.MustRegister(GardenerCollectBackupBucketsType, asynq.HandlerFunc(HandleGardenerCollectBackupBucketsTask))
	registry.TaskRegistry.MustRegister(TaskCollectAll, asynq.HandlerFunc(HandleCollectAllTask))
	registry.TaskRegistry.MustRegister(TaskLinkAll, asynq.HandlerFunc(HandleLinkAllTask))
}
