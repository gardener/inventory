// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	iamv1 "cloud.google.com/go/iam/apiv1/iampb"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/models"
	gcputils "github.com/gardener/inventory/pkg/gcp/utils"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectIAMPolicies is the name of the task for collecting GCP IAM Policies.
	TaskCollectIAMPolicies = "gcp:task:collect-iam-policies"

	// ResourceTypeProject represents a resource of type project in the IAMPolicy model.
	// alternatives are 'organisation' and 'folder', which are currently not used.
	ResourceTypeProject = "project"
)

// NewCollectIAMPoliciesTask creates a new [asynq.Task] task for collecting GCP
// IAM Policies without specifying a payload.
func NewCollectIAMPoliciesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectIAMPolicies, nil)
}

// CollectIAMPoliciesPayload is the payload, which is used to collect GCP IAM Policies.
type CollectIAMPoliciesPayload struct {
	// ProjectID specifies the GCP project ID, which is associated with a
	// registered client.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// HandleCollectIAMPoliciesTask is the handler, which collects GCP IAM Policies.
func HandleCollectIAMPoliciesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we will enqueue tasks for
	// collecting IAM Policies for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectIAMPolicies(ctx)
	}

	// Collect IAM Policies using the client associated with the project ID from
	// the payload.
	var payload CollectIAMPoliciesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectIAMPolicies(ctx, payload)
}

// enqueueCollectIAMPolicies enqueues tasks for collecting GCP IAM Policies
// for all collected GCP projects.
func enqueueCollectIAMPolicies(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	queue := asynqutils.GetQueueName(ctx)
	err := gcpclients.ProjectsClientset.Range(func(projectID string, _ *gcpclients.Client[*resourcemanager.ProjectsClient]) error {
		p := &CollectIAMPoliciesPayload{ProjectID: projectID}
		data, err := json.Marshal(p)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP IAM Policies",
				"project", projectID,
				"reason", err,
			)

			return registry.ErrContinue
		}

		task := asynq.NewTask(TaskCollectIAMPolicies, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project", projectID,
				"reason", err,
			)

			return registry.ErrContinue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"project", projectID,
			"id", info.ID,
			"queue", info.Queue,
		)

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// collectIAMPolicies collects GCP IAM Policies for a given project.
func collectIAMPolicies(ctx context.Context, payload CollectIAMPoliciesPayload) error {
	logger := asynqutils.GetLogger(ctx)
	projectID := payload.ProjectID

	client, ok := gcpclients.ProjectsClientset.Get(projectID)
	if !ok {
		logger.Error(
			"cannot find GCP projects client for project",
			"project", projectID,
		)

		return asynqutils.SkipRetry(ClientNotFound(projectID))
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			iamPoliciesDesc,
			prometheus.GaugeValue,
			float64(count),
			projectID,
		)
		key := metrics.Key(TaskCollectIAMPolicies, payload.ProjectID)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger.Info("collecting GCP IAM policies", "project", projectID)

	req := &iamv1.GetIamPolicyRequest{
		Resource: gcputils.ProjectFQN(projectID),
	}

	policy, err := client.Client.GetIamPolicy(ctx, req)
	if err != nil {
		logger.Error(
			"failed to get IAM policy for project",
			"project", projectID,
			"reason", err,
		)

		return registry.ErrContinue
	}

	resourceName := gcputils.ProjectFQN(projectID)

	iamPolicy := models.IAMPolicy{
		ResourceName: resourceName,
		ResourceType: ResourceTypeProject,
		Version:      policy.Version,
	}

	bindings := make([]models.IAMBinding, 0, len(policy.Bindings))
	roleMembers := make([]models.IAMRoleMember, 0)

	for _, binding := range policy.Bindings {
		condition := ""
		if binding.Condition != nil {
			condition = binding.Condition.Expression
		}

		for _, member := range binding.Members {
			iamBindingMember := models.IAMRoleMember{
				ResourceName: resourceName,
				ResourceType: ResourceTypeProject,
				Role:         binding.Role,
				Member:       member,
			}

			roleMembers = append(roleMembers, iamBindingMember)
		}

		iamBinding := models.IAMBinding{
			ResourceName: resourceName,
			ResourceType: ResourceTypeProject,
			Role:         binding.Role,
			Condition:    condition,
		}
		bindings = append(bindings, iamBinding)
	}

	out, err := db.DB.NewInsert().
		Model(&iamPolicy).
		On("CONFLICT (resource_name, resource_type) DO UPDATE").
		Set("version = EXCLUDED.version").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"failed to insert IAM policy",
			"project", projectID,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	if len(bindings) > 0 {
		out, err = db.DB.NewInsert().
			Model(&bindings).
			On("CONFLICT (role, resource_name, resource_type) DO UPDATE").
			Set("condition = EXCLUDED.condition").
			Set("updated_at = EXCLUDED.updated_at").
			Returning("id").
			Exec(ctx)

		if err != nil {
			logger.Error(
				"failed to insert IAM policy bindings",
				"project", projectID,
				"reason", err,
			)

			return err
		}

		count, err = out.RowsAffected()
		if err != nil {
			return err
		}
	}

	if len(roleMembers) > 0 {
		out, err = db.DB.NewInsert().
			Model(&roleMembers).
			On("CONFLICT (member, role, resource_name, resource_type) DO UPDATE").
			Set("updated_at = EXCLUDED.updated_at").
			Returning("id").
			Exec(ctx)

		if err != nil {
			logger.Error(
				"failed to insert IAM policy binding pairs",
				"project", projectID,
				"reason", err,
			)

			return err
		}

		count, err = out.RowsAffected()
		if err != nil {
			return err
		}
	}

	logger.Info(
		"collected IAM policy",
		"project", projectID,
		"bindings", len(bindings),
		"role-members", len(roleMembers),
	)

	return nil
}
