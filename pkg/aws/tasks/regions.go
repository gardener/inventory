// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// TaskCollectRegions is the name of the task for collecting AWS
	// regions.
	TaskCollectRegions = "aws:task:collect-regions"
)

// NewAwsCollectRegionsTask creates a new [asynq.Task] task for collecting AWS
// regions.
func NewCollectRegionsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectRegions, nil)
}

// CollectRegionsPayload is the payload, which is used to collect AWS regions.
type CollectRegionsPayload struct {
	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// HandleCollectRegionsTask is the handler, which collects AWS Regions.
func HandleCollectRegionsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting AWS regions")

	// If we were called without a payload, then we will enqueue tasks for
	// collecting regions for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectRegionsForAllClients()
	}

	// Collect regions using the client associated with the Account ID from
	// the payload.
	var payload CollectRegionsPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if payload.AccountID == "" {
		return fmt.Errorf("%w: %w", ErrNoAccountID, asynq.SkipRetry)
	}

	return collectRegions(ctx, payload)
}

// enqueueCollectRegionsForAllClients enqueues tasks for collecting AWS Regions
// for all configured AWS EC2 clients.
func enqueueCollectRegionsForAllClients() error {
	awsclients.EC2Clientset.Range(func(accountID string, _ *awsclients.Client[*ec2.Client]) error {
		p := &CollectRegionsPayload{AccountID: accountID}
		data, err := json.Marshal(p)
		if err != nil {
			slog.Error(
				"failed to marshal payload for AWS regions",
				"account_id", accountID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		task := asynq.NewTask(TaskCollectRegions, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			slog.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"account_id", accountID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		slog.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"account_id", accountID,
		)
		return nil
	})

	return nil
}

// collectRegions collects the AWS regions using the client configuration
// specified in the payload.
func collectRegions(ctx context.Context, payload CollectRegionsPayload) error {
	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return fmt.Errorf("%w: %s (%w)", ErrClientNotFound, payload.AccountID, asynq.SkipRetry)
	}

	result, err := client.Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		slog.Error("could not describe regions", "account_id", client.AccountID, "reason", err)
		return err
	}

	regions := make([]models.Region, 0, len(result.Regions))
	for _, region := range result.Regions {
		item := models.Region{
			Name:        strings.StringFromPointer(region.RegionName),
			AccountID:   client.AccountID,
			Endpoint:    strings.StringFromPointer(region.Endpoint),
			OptInStatus: strings.StringFromPointer(region.OptInStatus),
		}
		regions = append(regions, item)
	}

	if len(regions) == 0 {
		return nil
	}

	// Bulk insert regions into db
	out, err := db.DB.NewInsert().
		Model(&regions).
		On("CONFLICT (name, account_id) DO UPDATE").
		Set("endpoint = EXCLUDED.endpoint").
		Set("opt_in_status = EXCLUDED.opt_in_status").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error("could not insert regions into db", "account_id", client.AccountID, "reason", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws regions", "account_id", client.AccountID, "count", count)

	return nil
}
