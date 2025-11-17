// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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
	// TaskCollectAll is a meta task, which enqueues all relevant Azure tasks.
	TaskCollectAll = "az:task:collect-all"

	// TaskLinkAll is a task, which establishes links between Azure models.
	TaskLinkAll = "az:task:link-all"
)

// HandleCollectAllTask is a handler, which enqueues tasks for collecting all
// Azure objects.
func HandleCollectAllTask(ctx context.Context, _ *asynq.Task) error {
	queue := asynqutils.GetQueueName(ctx)

	// Task constructors
	taskFns := []asynqutils.TaskConstructor{
		NewCollectSubscriptionsTask,
		NewCollectResourceGroupsTask,
		NewCollectVirtualMachinesTask,
		NewCollectPublicAddressesTask,
		NewCollectLoadBalancersTask,
		NewCollectVPCsTask,
		NewCollectSubnetsTask,
		NewCollectStorageAccountsTask,
		NewCollectBlobContainersTask,
		NewCollectNetworkInterfacesTask,
	}

	return asynqutils.Enqueue(ctx, taskFns, asynq.Queue(queue))
}

// HandleLinkAllTask is a handler, which establishes links between the various
// Azure models.
func HandleLinkAllTask(ctx context.Context, _ *asynq.Task) error {
	linkFns := []dbutils.LinkFunction{
		LinkResourceGroupWithSubscription,
		LinkVirtualMachineWithResourceGroup,
		LinkPublicAddressWithResourceGroup,
		LinkLoadBalancerWithResourceGroup,
		LinkVPCWithResourceGroup,
		LinkSubnetWithVPC,
		LinkBlobContainerWithResourceGroup,
	}

	return dbutils.LinkObjects(ctx, db.DB, linkFns)
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
	registry.TaskRegistry.MustRegister(TaskCollectStorageAccounts, asynq.HandlerFunc(HandleCollectStorageAccountsTask))
	registry.TaskRegistry.MustRegister(TaskCollectBlobContainers, asynq.HandlerFunc(HandleCollectBlobContainersTask))
	registry.TaskRegistry.MustRegister(TaskCollectUsers, asynq.HandlerFunc(HandleCollectUsersTask))
	registry.TaskRegistry.MustRegister(TaskCollectNetworkInterfaces, asynq.HandlerFunc(HandleCollectNetworkInterfacesTask))
}
