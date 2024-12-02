// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
	"github.com/gardener/inventory/pkg/gardener/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	stringutils "github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// TaskCollectBackupBuckets is the name of the task for collecting
	// Gardener BackupBuckets resources.
	TaskCollectBackupBuckets = "g:task:collect-backup-buckets"
)

// NewCollectBackupBucketsTask creates a new [asynq.Task] for collecting
// Gardener BackupBuckets, without specifying a payload.
func NewCollectBackupBucketsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectBackupBuckets, nil)
}

// HandleCollectBackupBucketsTask is the handler for collecting BackupBuckets.
func HandleCollectBackupBucketsTask(ctx context.Context, t *asynq.Task) error {
	client, err := gardenerclient.VirtualGardenClient()
	if err != nil {
		return asynqutils.SkipRetry(ErrNoVirtualGardenClientFound)
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("Collecting Gardener backup buckets")
	buckets := make([]models.BackupBucket, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().BackupBuckets().List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		b, ok := obj.(*v1beta1.BackupBucket)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		var state string
		var stateProgress int
		if b.Status.LastOperation != nil {
			state = string(b.Status.LastOperation.State)
			stateProgress = int(b.Status.LastOperation.Progress)
		}

		item := models.BackupBucket{
			Name:              b.GetName(),
			SeedName:          stringutils.StringFromPointer(b.Spec.SeedName),
			ProviderType:      b.Spec.Provider.Type,
			RegionName:        b.Spec.Provider.Region,
			State:             state,
			StateProgress:     stateProgress,
			CreationTimestamp: b.CreationTimestamp.Time,
		}
		buckets = append(buckets, item)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list backup buckets: %w", err)
	}

	if len(buckets) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&buckets).
		On("CONFLICT (name) DO UPDATE").
		Set("provider_type = EXCLUDED.provider_type").
		Set("region_name = EXCLUDED.region_name").
		Set("seed_name = EXCLUDED.seed_name").
		Set("state = EXCLUDED.state").
		Set("state_progress = EXCLUDED.state_progress").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener backup buckets into db",
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated gardener backup buckets", "count", count)

	return nil
}
