// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/registry"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	dbutils "github.com/gardener/inventory/pkg/utils/db"
)

const (
	// TaskCollectAll is a meta task, which enqueues all relevant OpenStack
	// tasks.
	TaskCollectAll = "openstack:task:collect-all"

	// TaskLinkAll is a task, which creates links between the OpenStack
	// models.
	TaskLinkAll = "openstack:task:link-all"
)

// HandleCollectAllTask is a handler, which enqueues tasks for collecting all
// OpenStack objects.
func HandleCollectAllTask(ctx context.Context, t *asynq.Task) error {
	queue := asynqutils.GetQueueName(ctx)

	// Task constructors
	taskFns := []asynqutils.TaskConstructor{
		NewCollectServersTask,
		NewCollectNetworksTask,
		NewCollectLoadBalancersTask,
		NewCollectSubnetsTask,
		NewCollectFloatingIPsTask,
	}

	return asynqutils.Enqueue(ctx, taskFns, asynq.Queue(queue))
}

// HandleLinkAllTask is a handler, which establishes links between the various
// OpenStack models.
func HandleLinkAllTask(ctx context.Context, t *asynq.Task) error {
	linkFns := []dbutils.LinkFunction{
		LinkSubnetsWithNetworks,
		LinkLoadBalancersWithSubnets,
	}

	return dbutils.LinkObjects(ctx, db.DB, linkFns)
}

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(TaskCollectServers, asynq.HandlerFunc(HandleCollectServersTask))
	registry.TaskRegistry.MustRegister(TaskCollectNetworks, asynq.HandlerFunc(HandleCollectNetworksTask))
	registry.TaskRegistry.MustRegister(TaskCollectLoadBalancers, asynq.HandlerFunc(HandleCollectLoadBalancersTask))
	registry.TaskRegistry.MustRegister(TaskCollectSubnets, asynq.HandlerFunc(HandleCollectSubnetsTask))
	registry.TaskRegistry.MustRegister(TaskCollectFloatingIPs, asynq.HandlerFunc(HandleCollectFloatingIPsTask))
	registry.TaskRegistry.MustRegister(TaskCollectAll, asynq.HandlerFunc(HandleCollectAllTask))
	registry.TaskRegistry.MustRegister(TaskLinkAll, asynq.HandlerFunc(HandleLinkAllTask))
}
