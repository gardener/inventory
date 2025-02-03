// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/registry"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	stringutils "github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// TaskCollectRegions is the name of the task for collecting AWS
	// regions.
	TaskCollectRegions = "aws:task:collect-regions"
)

// NewAwsCollectRegionsTask creates a new [asynq.Task] task for collecting AWS
// regions without specifying a payload.
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
	// If we were called without a payload, then we will enqueue tasks for
	// collecting regions for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectRegions(ctx)
	}

	// Collect regions using the client associated with the Account ID from
	// the payload.
	var payload CollectRegionsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	return collectRegions(ctx, payload)
}

// enqueueCollectRegions enqueues tasks for collecting AWS Regions
// for all configured AWS EC2 clients.
func enqueueCollectRegions(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if awsclients.EC2Clientset.Length() == 0 {
		logger.Warn("no AWS clients found")
		return nil
	}

	queue := asynqutils.GetQueueName(ctx)
	err := awsclients.EC2Clientset.Range(func(accountID string, _ *awsclients.Client[*ec2.Client]) error {
		p := &CollectRegionsPayload{AccountID: accountID}
		data, err := json.Marshal(p)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS regions",
				"account_id", accountID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		task := asynq.NewTask(TaskCollectRegions, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"account_id", accountID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"account_id", accountID,
		)
		return nil
	})

	return err
}

// collectRegions collects the AWS regions using the client configuration
// specified in the payload.
func collectRegions(ctx context.Context, payload CollectRegionsPayload) error {
	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	logger := asynqutils.GetLogger(ctx)

	logger.Info("collecting AWS regions", "account_id", payload.AccountID)
	result, err := client.Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})

	if err != nil {
		logger.Error(
			"could not describe regions",
			"account_id", payload.AccountID,
			"reason", err,
		)
		return err
	}

	regions := make([]models.Region, 0, len(result.Regions))
	for _, region := range result.Regions {
		item := models.Region{
			Name:        stringutils.StringFromPointer(region.RegionName),
			AccountID:   payload.AccountID,
			Endpoint:    stringutils.StringFromPointer(region.Endpoint),
			OptInStatus: stringutils.StringFromPointer(region.OptInStatus),
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
		logger.Error(
			"could not insert regions into db",
			"account_id", payload.AccountID,
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated aws regions",
		"account_id", payload.AccountID,
		"count", count,
	)

	return nil
}
