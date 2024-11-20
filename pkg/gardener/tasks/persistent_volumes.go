// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
	"github.com/gardener/inventory/pkg/gardener/models"
	gutils "github.com/gardener/inventory/pkg/gardener/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectPersistentVolumes is the name of the task for collecting Gardener
	// PVs.
	TaskCollectPersistentVolumes = "g:task:collect-persistent-volumes"
)

// ErrDriverNotSupported is an error, which is returned when the
// an unknown CSI driver is parsed.
var ErrDriverNotSupported = errors.New("driver not supported")

// DriverNotSupported wraps [ErrDriverNotSupported] with the given driver name.
func DriverNotSupported(driver string) error {
	return fmt.Errorf("%w: %s", ErrDriverNotSupported, driver)
}

// CollectPersistentVolumesPayload is the payload, which is used for collecting Gardener
// PVs.
type CollectPersistentVolumesPayload struct {
	// Seed is the name of the seed cluster from which to collect Gardener
	// PVs.
	Seed string `json:"seed" yaml:"seed"`
}

// NewCollectPersistentVolumesTask creates a new [asynq.Task] for collecting Gardener
// PVs, without specifying a payload.
func NewCollectPersistentVolumesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectPersistentVolumes, nil)
}

// HandleCollectPersistentVolumesTask is the handler for collecting Gardener PVs.
func HandleCollectPersistentVolumesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting PVs from all known Gardener Seed clusters and the Virtual Garden.
	data := t.Payload()
	if data == nil {
		return enqueueCollectPersistentVolumes(ctx)
	}

	var payload CollectPersistentVolumesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Seed == "" {
		return asynqutils.SkipRetry(ErrNoSeedCluster)
	}

	return collectPersistentVolumes(ctx, payload)
}

// enqueueCollectPersistentVolumes enqueues tasks for collecting Gardener Volumes from
// all known Seed Clusters and the Virtual Garden.
func enqueueCollectPersistentVolumes(ctx context.Context) error {
	seeds, err := gutils.GetSeedsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get seeds from db: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)

	// Create a task for each known seed cluster
	for _, s := range seeds {
		payload := CollectPersistentVolumesPayload{
			Seed: s.Name,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Gardener Volumes",
				"seed", s.Name,
				"reason", err,
			)
			continue
		}

		task := asynq.NewTask(TaskCollectPersistentVolumes, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"seed", s.Name,
				"reason", err,
			)
			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"seed", s.Name,
		)
	}
	return nil
}

// collectPersistentVolumes collects the Gardener Volumes from the Seed Cluster
// specified in the payload.
func collectPersistentVolumes(ctx context.Context, payload CollectPersistentVolumesPayload) error {
	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting Gardener Persistent Volumes", "seed", payload.Seed)
	client, err := gardenerclient.KubeClient(ctx, payload.Seed)
	if err != nil {
		if errors.Is(err, gardenerclient.ErrSeedIsExcluded) {
			// Don't treat excluded seeds as errors, in order to
			// avoid accumulating archived tasks
			logger.Warn("seed is excluded", "seed", payload.Seed)
			return nil
		}
		return asynqutils.SkipRetry(fmt.Errorf("cannot get garden client for %q: %s", payload.Seed, err))
	}

	pvs := make([]models.PersistentVolume, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1().PersistentVolumes().List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		pv, ok := obj.(*corev1.PersistentVolume)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		var diskRef string
		var sourceName string
		source := pv.Spec.PersistentVolumeSource

		switch {
		case source.CSI != nil:
			sourceName = source.CSI.Driver
			diskRef = source.CSI.VolumeHandle
		case source.GCEPersistentDisk != nil:
			sourceName = "gce-persistent-disk"
			diskRef = source.GCEPersistentDisk.PDName
		case source.AWSElasticBlockStore != nil:
			sourceName = "aws-elastic-block-store"
			diskRef = source.AWSElasticBlockStore.VolumeID
		case source.AzureDisk != nil:
			sourceName = "azure-disk"
			diskRef = source.AzureDisk.DiskName
			// TODO: Add the rest of the in-tree drivers
		}

		item := models.PersistentVolume{
			Name:         pv.GetName(),
			SeedName:     payload.Seed,
			Provider:     sourceName,
			DiskRef:      diskRef,
			Status:       string(pv.Status.Phase),
			Capacity:     pv.Spec.Capacity.Storage().String(),
			StorageClass: pv.Spec.StorageClassName,
		}
		pvs = append(pvs, item)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list persistent volumes for seed %q: %w", payload.Seed, err)
	}

	if len(pvs) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&pvs).
		On("CONFLICT (name, seed_name) DO UPDATE").
		Set("provider= EXCLUDED.provider").
		Set("disk_ref= EXCLUDED.disk_ref").
		Set("status = EXCLUDED.status").
		Set("capacity = EXCLUDED.capacity").
		Set("storage_class = EXCLUDED.storage_class").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener persistent volumes into db",
			"seed", payload.Seed,
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gardener persistent volumes",
		"seed", payload.Seed,
		"count", count,
	)

	return nil
}
