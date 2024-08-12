// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/models"
	stringutils "github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// TaskCollectBackupBuckets is the type of the task that collects Gardener BackupBuckets.
	TaskCollectBackupBuckets = "g:task:collect-backup-buckets"
)

// NewGardenerCollectBackupBucketsTask creates a new task for collecting Gardener BackupBuckets.
func NewGardenerCollectBackupBucketsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectBackupBuckets, nil)
}

// HandleGardenerCollectBackupBucketsTask is a handler function that collects Gardener BackupBuckets.
func HandleGardenerCollectBackupBucketsTask(ctx context.Context, t *asynq.Task) error {
	return collectBackupBuckets(ctx)
}

func collectBackupBuckets(ctx context.Context) error {
	slog.Info("Collecting Gardener backup buckets")
	gardenClient, err := gardenerclient.VirtualGardenClient()
	if err != nil {
		return fmt.Errorf("could not get garden client: %w", asynq.SkipRetry)
	}

	backupBuckets := make([]models.BackupBucket, 0)
	err = pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return gardenClient.CoreV1beta1().BackupBuckets().List(ctx, metav1.ListOptions{})
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
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

		backupBucket := models.BackupBucket{
			Name:          b.GetName(),
			SeedName:      stringutils.StringFromPointer(b.Spec.SeedName),
			ProviderType:  b.Spec.Provider.Type,
			RegionName:    b.Spec.Provider.Region,
			State:         state,
			StateProgress: stateProgress,
		}
		backupBuckets = append(backupBuckets, backupBucket)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list backup buckets: %w", err)
	}

	if len(backupBuckets) == 0 {
		return nil
	}
	_, err = db.DB.NewInsert().
		Model(&backupBuckets).
		On("CONFLICT (name) DO UPDATE").
		Set("provider_type = EXCLUDED.provider_type").
		Set("region_name = EXCLUDED.region_name").
		Set("seed_name = EXCLUDED.seed_name").
		Set("state = EXCLUDED.state").
		Set("state_progress = EXCLUDED.state_progress").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert gardener backup buckets into db", "err", err)
		return err
	}

	return nil
}
