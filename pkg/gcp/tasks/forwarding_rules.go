// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/hibiken/asynq"
	"google.golang.org/api/iterator"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/constants"
	"github.com/gardener/inventory/pkg/gcp/models"
	gcputils "github.com/gardener/inventory/pkg/gcp/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// TaskCollectForwardingRules is the name of the task for collecting GCP
// Forwarding Rules.
//
// For more information about Forwarding Rules, please refer to the
// [Forwarding Rules overview] documentation.
//
// [Forwarding Rules overview]: https://cloud.google.com/load-balancing/docs/forwarding-rule-concepts
const TaskCollectForwardingRules = "gcp:task:collect-forwarding-rules"

// CollectForwardingRulesPayload is the payload used for collecting GCP Forwarding
// Rules for a given project.
type CollectForwardingRulesPayload struct {
	// ProjectID specifies the globally unique project id from which to
	// collect GCP Forwarding Rules.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// NewCollectForwardingRulesTask creates a new [asynq.Task] for collecting GCP
// Forwarding Rules, without specifying a payload.
func NewCollectForwardingRulesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectForwardingRules, nil)
}

// HandleCollectForwardingRules is the handler, which collects GCP Forwarding
// Rules.
func HandleCollectForwardingRules(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting Forwarding Rules from all registered projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectForwardingRules(ctx)
	}

	var payload CollectForwardingRulesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectForwardingRules(ctx, payload)
}

// enqueueCollectForwardingRules enqueues tasks for collecting GCP Forwarding
// Rules for all known projects.
func enqueueCollectForwardingRules(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if gcpclients.ForwardingRulesClientset.Length() == 0 {
		logger.Warn("no GCP forwarding rules clients found")

		return nil
	}

	// Enqueue tasks for all registered GCP Projects
	queue := asynqutils.GetQueueName(ctx)
	err := gcpclients.ForwardingRulesClientset.Range(func(projectID string, _ *gcpclients.Client[*compute.ForwardingRulesClient]) error {
		payload := CollectForwardingRulesPayload{
			ProjectID: projectID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP Forwarding Rules",
				"project", projectID,
				"reason", err,
			)

			return registry.ErrContinue
		}
		task := asynq.NewTask(TaskCollectForwardingRules, data)
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

// collectForwardingRules collects the GCP Forwarding Rules from the project
// specified in the payload.
func collectForwardingRules(ctx context.Context, payload CollectForwardingRulesPayload) error {
	client, ok := gcpclients.ForwardingRulesClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting GCP forwarding rules", "project", payload.ProjectID)

	pageSize := uint32(constants.PageSize)
	partialSuccess := true
	req := &computepb.AggregatedListForwardingRulesRequest{
		Project:              gcputils.ProjectFQN(payload.ProjectID),
		MaxResults:           &pageSize,
		ReturnPartialSuccess: &partialSuccess,
	}

	items := make([]models.ForwardingRule, 0)
	it := client.Client.AggregatedList(ctx, req)
	for {
		// The iterator returns a k/v pair, where the key represents a
		// specific GCP Region and the value is the slice of forwarding
		// rules in the region. Note that Forwarding Rules are regional
		// and global. The `global' key represents the global forwarding
		// rules returned by the aggregated API call.
		pair, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			logger.Error(
				"failed to get GCP Forwarding Rules",
				"project", payload.ProjectID,
				"reason", err,
			)

			return err
		}

		region := gcputils.UnqualifyRegion(pair.Key)
		for _, fr := range pair.Value.ForwardingRules {
			item := models.ForwardingRule{
				RuleID:              fr.GetId(),
				ProjectID:           payload.ProjectID,
				Name:                fr.GetName(),
				IPAddress:           net.ParseIP(fr.GetIPAddress()),
				IPProtocol:          fr.GetIPProtocol(),
				IPVersion:           fr.GetIpVersion(),
				AllPorts:            fr.GetAllPorts(),
				AllowGlobalAccess:   fr.GetAllowGlobalAccess(),
				BackendService:      gcputils.ResourceNameFromURL(fr.GetBackendService()),
				BaseForwardingRule:  fr.GetBaseForwardingRule(),
				CreationTimestamp:   fr.GetCreationTimestamp(),
				Description:         fr.GetDescription(),
				LoadBalancingScheme: fr.GetLoadBalancingScheme(),
				Network:             gcputils.ResourceNameFromURL(fr.GetNetwork()),
				NetworkTier:         fr.GetNetworkTier(),
				PortRange:           fr.GetPortRange(),
				Ports:               fr.GetPorts(),
				Region:              region,
				ServiceLabel:        fr.GetServiceLabel(),
				ServiceName:         fr.GetServiceName(),
				SourceIPRanges:      fr.GetSourceIpRanges(),
				Subnetwork:          gcputils.ResourceNameFromURL(fr.GetSubnetwork()),
				Target:              fr.GetTarget(),
			}
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (project_id, rule_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("ip_address = EXCLUDED.ip_address").
		Set("ip_protocol = EXCLUDED.ip_protocol").
		Set("ip_version = EXCLUDED.ip_version").
		Set("all_ports = EXCLUDED.all_ports").
		Set("allow_global_access = EXCLUDED.allow_global_access").
		Set("backend_service = EXCLUDED.backend_service").
		Set("base_forwarding_rule = EXCLUDED.base_forwarding_rule").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("description = EXCLUDED.description").
		Set("load_balancing_scheme = EXCLUDED.load_balancing_scheme").
		Set("network = EXCLUDED.network").
		Set("network_tier = EXCLUDED.network_tier").
		Set("port_range = EXCLUDED.port_range").
		Set("ports = EXCLUDED.ports").
		Set("region = EXCLUDED.region").
		Set("service_label = EXCLUDED.service_label").
		Set("service_name = EXCLUDED.service_name").
		Set("source_ip_ranges = EXCLUDED.source_ip_ranges").
		Set("subnetwork = EXCLUDED.subnetwork").
		Set("target = EXCLUDED.target").
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
		"populated gcp forwarding rules",
		"project", payload.ProjectID,
		"count", count,
	)

	return nil
}
