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
	// TaskCollectAll is a meta task, which enqueues all relevant AWS
	// tasks.
	TaskCollectAll = "aws:task:collect-all"

	// TaskLinkAll is a task, which creates links between the AWS
	// models.
	TaskLinkAll = "aws:task:link-all"
)

// HandleCollectAllTask is a handler, which enqueues tasks for collecting all
// AWS objects.
func HandleCollectAllTask(ctx context.Context, _ *asynq.Task) error {
	queue := asynqutils.GetQueueName(ctx)

	// Task constructors
	taskFns := []asynqutils.TaskConstructor{
		NewCollectRegionsTask,
		NewCollectAvailabilityZonesTask,
		NewCollectVPCsTask,
		NewCollectSubnetsTask,
		NewCollectInstancesTask,
		NewCollectImagesTask,
		NewCollectLoadBalancersTask,
		NewCollectBucketsTask,
		NewCollectNetworkInterfacesTask,
		NewCollectDHCPOptionSetsTask,
	}

	return asynqutils.Enqueue(ctx, taskFns, asynq.Queue(queue))
}

// HandleLinkAllTask is a handler, which establishes links between the various
// AWS models.
func HandleLinkAllTask(ctx context.Context, _ *asynq.Task) error {
	linkFns := []dbutils.LinkFunction{
		LinkAvailabilityZoneWithRegion,
		LinkInstanceWithRegion,
		LinkInstanceWithSubnet,
		LinkInstanceWithVPC,
		LinkInstanceWithImage,
		LinkRegionWithVPC,
		LinkSubnetWithAZ,
		LinkSubnetWithVPC,
		LinkImageWithRegion,
		LinkLoadBalancerWithVpc,
		LinkLoadBalancerWithRegion,
		LinkNetworkInterfaceWithInstance,
		LinkNetworkInterfaceWithLoadBalancer,
	}

	return dbutils.LinkObjects(ctx, db.DB, linkFns)
}

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(TaskCollectRegions, asynq.HandlerFunc(HandleCollectRegionsTask))
	registry.TaskRegistry.MustRegister(TaskCollectAvailabilityZones, asynq.HandlerFunc(HandleCollectAvailabilityZonesTask))
	registry.TaskRegistry.MustRegister(TaskCollectVPCs, asynq.HandlerFunc(HandleCollectVPCsTask))
	registry.TaskRegistry.MustRegister(TaskCollectSubnets, asynq.HandlerFunc(HandleCollectSubnetsTask))
	registry.TaskRegistry.MustRegister(TaskCollectInstances, asynq.HandlerFunc(HandleCollectInstancesTask))
	registry.TaskRegistry.MustRegister(TaskCollectImages, asynq.HandlerFunc(HandleCollectImagesTask))
	registry.TaskRegistry.MustRegister(TaskCollectLoadBalancers, asynq.HandlerFunc(HandleCollectLoadBalancersTask))
	registry.TaskRegistry.MustRegister(TaskCollectBuckets, asynq.HandlerFunc(HandleCollectBucketsTask))
	registry.TaskRegistry.MustRegister(TaskCollectNetworkInterfaces, asynq.HandlerFunc(HandleCollectNetworkInterfacesTask))
	registry.TaskRegistry.MustRegister(TaskCollectDHCPOptionSets, asynq.HandlerFunc(HandleCollectDHCPOptionSetsTask))
	registry.TaskRegistry.MustRegister(TaskCollectAll, asynq.HandlerFunc(HandleCollectAllTask))
	registry.TaskRegistry.MustRegister(TaskLinkAll, asynq.HandlerFunc(HandleLinkAllTask))
}
