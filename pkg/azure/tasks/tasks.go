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
	// TaskCollectAll is a meta task, which enqueues all relevant Azure tasks.
	TaskCollectAll = "az:task:collect-all"

	// TaskLinkAll is a task, which establishes links between Azure models.
	TaskLinkAll = "az:task:link-all"
)

// HandleCollectAllTask is a handler, which enqueues tasks for collecting all
// Azure objects.
func HandleCollectAllTask(ctx context.Context, t *asynq.Task) error {
	// Task constructors
	taskFns := []utils.TaskConstructor{
		NewCollectSubscriptionsTasks,
		NewCollectResourceGroupsTask,
		NewCollectVirtualMachinesTask,
		NewCollectPublicAddressesTask,
		NewCollectLoadBalancersTask,
		NewCollectVPCsTask,
		NewCollectSubnetsTask,
	}

	return utils.Enqueue(ctx, taskFns)
}

// HandleLinkAllTask is a handler, which establishes links between the various
// Azure models.
func HandleLinkAllTask(ctx context.Context, t *asynq.Task) error {
	linkFns := []utils.LinkFunction{
		LinkResourceGroupWithSubscription,
		LinkVirtualMachineWithResourceGroup,
		LinkPublicAddressWithResourceGroup,
		LinkLoadBalancerWithResourceGroup,
	}

	return utils.LinkObjects(ctx, db.DB, linkFns)
}

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(TaskCollectAll, asynq.HandlerFunc(HandleCollectAllTask))
	registry.TaskRegistry.MustRegister(TaskLinkAll, asynq.HandlerFunc(HandleLinkAllTask))
	registry.TaskRegistry.MustRegister(TaskCollectSubscriptions, asynq.HandlerFunc(HandleCollectSubscriptionsTask))
	registry.TaskRegistry.MustRegister(TaskCollectResourceGroups, asynq.HandlerFunc(HandleCollectResourceGroupsTask))
	registry.TaskRegistry.MustRegister(TaskCollectVirtualMachines, asynq.HandlerFunc(HandleCollectVirtualMachinesTask))
	registry.TaskRegistry.MustRegister(TaskCollectPublicAddresses, asynq.HandlerFunc(HandleCollectPublicAddressesTask))
	registry.TaskRegistry.MustRegister(TaskCollectLoadBalancers, asynq.HandlerFunc(HandleCollectLoadBalancersTask))
	registry.TaskRegistry.MustRegister(TaskCollectVPCs, asynq.HandlerFunc(HandleCollectVPCsTask))
	registry.TaskRegistry.MustRegister(TaskCollectSubnets, asynq.HandlerFunc(HandleCollectSubnetsTask))
}
