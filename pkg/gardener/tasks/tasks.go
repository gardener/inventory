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
		NewCollectProjectsTask,
		NewCollectSeedsTask,
		NewCollectShootsTask,
		NewCollectMachinesTask,
		NewCollectBackupBucketsTask,
		NewCollectCloudProfilesTask,
		NewCollectPersistentVolumesTask,
	}

	return utils.Enqueue(ctx, taskFns)
}

// HandleLinkAllTask is the handler, which establishes relationships between the
// various Gardener models.
func HandleLinkAllTask(ctx context.Context, r *asynq.Task) error {
	linkFns := []utils.LinkFunction{
		LinkShootWithProject,
		LinkShootWithSeed,
		LinkMachineWithShoot,
		LinkAWSImageWithCloudProfile,
		LinkGCPImageWithCloudProfile,
		LinkAzureImageWithCloudProfile,
		LinkProjectWithMember,
	}

	return utils.LinkObjects(ctx, db.DB, linkFns)
}

func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(TaskCollectProjects, asynq.HandlerFunc(HandleCollectProjectsTask))
	registry.TaskRegistry.MustRegister(TaskCollectSeeds, asynq.HandlerFunc(HandleCollectSeedsTask))
	registry.TaskRegistry.MustRegister(TaskCollectShoots, asynq.HandlerFunc(HandleCollectShootsTask))
	registry.TaskRegistry.MustRegister(TaskCollectMachines, asynq.HandlerFunc(HandleCollectMachinesTask))
	registry.TaskRegistry.MustRegister(TaskCollectBackupBuckets, asynq.HandlerFunc(HandleCollectBackupBucketsTask))
	registry.TaskRegistry.MustRegister(TaskCollectCloudProfiles, asynq.HandlerFunc(HandleCollectCloudProfilesTask))
	registry.TaskRegistry.MustRegister(TaskCollectAWSMachineImages, asynq.HandlerFunc(HandleCollectAWSMachineImagesTask))
	registry.TaskRegistry.MustRegister(TaskCollectGCPMachineImages, asynq.HandlerFunc(HandleCollectGCPMachineImagesTask))
	registry.TaskRegistry.MustRegister(TaskCollectAzureMachineImages, asynq.HandlerFunc(HandleCollectAzureMachineImagesTask))
	registry.TaskRegistry.MustRegister(TaskCollectPersistentVolumes, asynq.HandlerFunc(HandleCollectPersistentVolumesTask))
	registry.TaskRegistry.MustRegister(TaskCollectAll, asynq.HandlerFunc(HandleCollectAllTask))
	registry.TaskRegistry.MustRegister(TaskLinkAll, asynq.HandlerFunc(HandleLinkAllTask))
}
