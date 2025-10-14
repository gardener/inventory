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
	// TaskCollectAll is a meta task, which enqueues all relevant GCP tasks.
	TaskCollectAll = "gcp:task:collect-all"

	// TaskLinkAll is a task, which establishes links between GCP models.
	TaskLinkAll = "gcp:task:link-all"
)

// HandleCollectAllTask is a handler, which enqueues tasks for collecting all
// GCP objects.
func HandleCollectAllTask(ctx context.Context, _ *asynq.Task) error {
	queue := asynqutils.GetQueueName(ctx)

	// Task constructors
	taskFns := []asynqutils.TaskConstructor{
		NewCollectProjectsTask,
		NewCollectInstancesTask,
		NewCollectVPCsTask,
		NewCollectAddressesTask,
		NewCollectSubnetsTask,
		NewCollectBucketsTask,
		NewCollectForwardingRulesTask,
		NewCollectDisksTask,
		NewCollectGKEClustersTask,
		NewCollectTargetPoolsTask,
		NewCollectIAMPoliciesTask,
	}

	return asynqutils.Enqueue(ctx, taskFns, asynq.Queue(queue))
}

// HandleLinkAllTask is a handler, which establishes links between the various
// GCP models.
func HandleLinkAllTask(ctx context.Context, _ *asynq.Task) error {
	linkFns := []dbutils.LinkFunction{
		LinkInstanceWithProject,
		LinkVPCWithProject,
		LinkAddressWithProject,
		LinkInstanceWithNetworkInterface,
		LinkSubnetWithVPC,
		LinkSubnetWithProject,
		LinkForwardingRuleWithProject,
		LinkInstanceWithDisk,
		LinkGKEClusterWithProject,
		LinkTargetPoolWithInstance,
		LinkTargetPoolWithProject,
	}

	return dbutils.LinkObjects(ctx, db.DB, linkFns)
}

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister(TaskCollectAll, asynq.HandlerFunc(HandleCollectAllTask))
	registry.TaskRegistry.MustRegister(TaskLinkAll, asynq.HandlerFunc(HandleLinkAllTask))
	registry.TaskRegistry.MustRegister(TaskCollectProjects, asynq.HandlerFunc(HandleCollectProjectsTask))
	registry.TaskRegistry.MustRegister(TaskCollectInstances, asynq.HandlerFunc(HandleCollectInstancesTask))
	registry.TaskRegistry.MustRegister(TaskCollectVPCs, asynq.HandlerFunc(HandleCollectVPCsTask))
	registry.TaskRegistry.MustRegister(TaskCollectAddresses, asynq.HandlerFunc(HandleCollectAddressesTask))
	registry.TaskRegistry.MustRegister(TaskCollectSubnets, asynq.HandlerFunc(HandleCollectSubnetsTask))
	registry.TaskRegistry.MustRegister(TaskCollectBuckets, asynq.HandlerFunc(HandleCollectBucketsTask))
	registry.TaskRegistry.MustRegister(TaskCollectForwardingRules, asynq.HandlerFunc(HandleCollectForwardingRules))
	registry.TaskRegistry.MustRegister(TaskCollectDisks, asynq.HandlerFunc(HandleCollectDisksTask))
	registry.TaskRegistry.MustRegister(TaskCollectGKEClusters, asynq.HandlerFunc(HandleCollectGKEClusters))
	registry.TaskRegistry.MustRegister(TaskCollectTargetPools, asynq.HandlerFunc(HandleCollectTargetPools))
	registry.TaskRegistry.MustRegister(TaskCollectIAMPolicies, asynq.HandlerFunc(HandleCollectIAMPoliciesTask))
}
