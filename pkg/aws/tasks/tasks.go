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
	// TaskCollectAll is a meta task, which enqueues all relevant AWS
	// tasks.
	TaskCollectAll = "aws:task:collect-all"

	// TaskLinkAll is a task, which creates links between the AWS
	// models.
	TaskLinkAll = "aws:task:link-all"
)

// HandleCollectAllTask is a handler, which enqueues tasks for collecting all
// AWS objects.
func HandleCollectAllTask(ctx context.Context, t *asynq.Task) error {
	// Task constructors
	taskFns := []utils.TaskConstructor{
		NewCollectRegionsTask,
		NewCollectAzsTask,
		NewCollectVpcsTask,
		NewCollectSubnetsTask,
		NewCollectInstancesTask,
		NewCollectLoadBalancersTask,
		NewCollectBucketsTask,
		NewCollectNetworkInterfacesTask,
	}

	return utils.Enqueue(taskFns)
}

// HandleLinkAllTask is a handler, which establishes links between the various
// AWS models.
func HandleLinkAllTask(ctx context.Context, t *asynq.Task) error {
	linkFns := []utils.LinkFunction{
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

	return utils.LinkObjects(ctx, db.DB, linkFns)
}

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(TaskCollectRegions, asynq.HandlerFunc(HandleAwsCollectRegionsTask))
	registry.TaskRegistry.MustRegister(TaskCollectAvailabilityZones, asynq.HandlerFunc(HandleCollectAzsTask))
	registry.TaskRegistry.MustRegister(TaskCollectAvailabilityZonesForRegion, asynq.HandlerFunc(HandleCollectAzsForRegionTask))
	registry.TaskRegistry.MustRegister(TaskCollectVPCs, asynq.HandlerFunc(HandleCollectVpcsTask))
	registry.TaskRegistry.MustRegister(TaskCollectVPCsForRegion, asynq.HandlerFunc(HandleCollectVpcsForRegionTask))
	registry.TaskRegistry.MustRegister(TaskCollectSubnets, asynq.HandlerFunc(HandleCollectSubnetsTask))
	registry.TaskRegistry.MustRegister(TaskCollectSubnetsForRegion, asynq.HandlerFunc(HandleCollectSubnetsForRegionTask))
	registry.TaskRegistry.MustRegister(TaskCollectInstances, asynq.HandlerFunc(HandleCollectInstancesTask))
	registry.TaskRegistry.MustRegister(TaskCollectImages, asynq.HandlerFunc(HandleCollectImagesTask))
	registry.TaskRegistry.MustRegister(TaskCollectInstancesForRegion, asynq.HandlerFunc(HandleCollectInstancesForRegionTask))
	registry.TaskRegistry.MustRegister(TaskCollectImagesForRegion, asynq.HandlerFunc(HandleCollectImagesForRegionTask))
	registry.TaskRegistry.MustRegister(TaskCollectLoadBalancersForRegion, asynq.HandlerFunc(HandleCollectLoadBalancersForRegionTask))
	registry.TaskRegistry.MustRegister(TaskCollectLoadBalancers, asynq.HandlerFunc(HandleCollectLoadBalancersTask))
	registry.TaskRegistry.MustRegister(TaskCollectBuckets, asynq.HandlerFunc(HandleCollectBucketsTask))
	registry.TaskRegistry.MustRegister(TaskCollectNetworkInterfaces, asynq.HandlerFunc(HandleCollectNetworkInterfacesTask))
	registry.TaskRegistry.MustRegister(TaskCollectNetworkInterfacesForRegion, asynq.HandlerFunc(HandleCollectNetworkInterfacesForRegionTask))
	registry.TaskRegistry.MustRegister(TaskCollectAll, asynq.HandlerFunc(HandleCollectAllTask))
	registry.TaskRegistry.MustRegister(TaskLinkAll, asynq.HandlerFunc(HandleLinkAllTask))
}
