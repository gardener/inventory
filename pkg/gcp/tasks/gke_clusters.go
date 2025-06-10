// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// TaskCollectGKEClusters is the name of the task for collecting GKE clusters.
const TaskCollectGKEClusters = "gcp:task:collect-gke-clusters"

// CollectGKEClustersPayload is the payload used for collecting GKE Clusters.
type CollectGKEClustersPayload struct {
	// ProjectID specifies the globally unique project id from which to
	// collect.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// NewCollectGKEClustersTask creates a new [asynq.Task] for collecting GKE
// Clusters, without specifying a payload.
func NewCollectGKEClustersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectGKEClusters, nil)
}

// HandleCollectGKEClusters is the handler, which collects GKE Clusters.
func HandleCollectGKEClusters(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting GKE Clusters from all registered projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectGKEClusters(ctx)
	}

	var payload CollectGKEClustersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectGKEClusters(ctx, payload)
}

// enqueueCollectGKEClusters enqueues tasks for collecting GKE Clusters.
func enqueueCollectGKEClusters(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if gcpclients.ClusterManagerClientset.Length() == 0 {
		logger.Warn("no GCP Cluster Manager clients found")

		return nil
	}

	// Enqueue tasks for all registered GCP Projects
	queue := asynqutils.GetQueueName(ctx)
	err := gcpclients.ClusterManagerClientset.Range(func(projectID string, _ *gcpclients.Client[*container.ClusterManagerClient]) error {
		payload := CollectGKEClustersPayload{
			ProjectID: projectID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GKE Clusters",
				"project", projectID,
				"reason", err,
			)

			return registry.ErrContinue
		}
		task := asynq.NewTask(TaskCollectGKEClusters, data)
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
			"id", info.ID,
			"queue", info.Queue,
			"project", projectID,
		)

		return nil
	})

	return err
}

// collectGKEClusters collects the GKE Clusters from the project specified in
// the payload.
func collectGKEClusters(ctx context.Context, payload CollectGKEClustersPayload) error {
	client, ok := gcpclients.ClusterManagerClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting GKE clusters", "project", payload.ProjectID)

	req := &containerpb.ListClustersRequest{
		// Match all zones and regions
		Parent: fmt.Sprintf("projects/%s/locations/-", payload.ProjectID),
	}
	resp, err := client.Client.ListClusters(ctx, req)
	if err != nil {
		return err
	}

	items := make([]models.GKECluster, 0)
	for _, cluster := range resp.Clusters {
		var caData string
		if cluster.MasterAuth != nil {
			caData = cluster.MasterAuth.GetClusterCaCertificate()
		}
		item := models.GKECluster{
			Name:                  cluster.GetName(),
			ClusterID:             cluster.GetId(),
			ProjectID:             payload.ProjectID,
			Location:              cluster.GetLocation(),
			Network:               cluster.GetNetwork(),
			Subnetwork:            cluster.GetSubnetwork(),
			ClusterIPv4CIDR:       cluster.GetClusterIpv4Cidr(),
			ServicesIPv4CIDR:      cluster.GetServicesIpv4Cidr(),
			EnableKubernetesAlpha: cluster.GetEnableKubernetesAlpha(),
			Endpoint:              cluster.GetEndpoint(),
			InitialVersion:        cluster.GetInitialClusterVersion(),
			CurrentMasterVersion:  cluster.GetCurrentMasterVersion(),
			CAData:                caData,
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (project_id, cluster_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("location = EXCLUDED.location").
		Set("network = EXCLUDED.network").
		Set("subnetwork = EXCLUDED.subnetwork").
		Set("cluster_ipv4_cidr = EXCLUDED.cluster_ipv4_cidr").
		Set("services_ipv4_cidr = EXCLUDED.services_ipv4_cidr").
		Set("enable_k8s_alpha = EXCLUDED.enable_k8s_alpha").
		Set("endpoint = EXCLUDED.endpoint").
		Set("initial_version = EXCLUDED.initial_version").
		Set("current_master_version = EXCLUDED.current_master_version").
		Set("ca_data = EXCLUDED.ca_data").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gke clusters",
		"project", payload.ProjectID,
		"count", count,
	)

	return nil
}
