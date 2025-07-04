// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
	"github.com/gardener/inventory/pkg/gardener/models"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectSeeds is the name of the task for collecting Gardener
	// Seeds.
	TaskCollectSeeds = "g:task:collect-seeds"
)

// NewCollectSeedsTask creates a new [asynq.Task] for collecting
// Gardener Seeds, without specifying a payload.
func NewCollectSeedsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectSeeds, nil)
}

// HandleCollectSeedsTask is the handler for collecting Gardener Seeds.
func HandleCollectSeedsTask(ctx context.Context, _ *asynq.Task) error {
	logger := asynqutils.GetLogger(ctx)
	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")

		return nil
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			seedsDesc,
			prometheus.GaugeValue,
			float64(count),
		)
		metrics.DefaultCollector.AddMetric(TaskCollectSeeds, metric)
	}()

	client := gardenerclient.DefaultClient.GardenClient()
	logger.Info("collecting Gardener seeds")
	seeds := make([]models.Seed, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().Seeds().List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err := p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		s, ok := obj.(*v1beta1.Seed)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		item := models.Seed{
			Name:              s.Name,
			KubernetesVersion: ptr.StringFromPointer(s.Status.KubernetesVersion),
			CreationTimestamp: s.CreationTimestamp.Time,
		}
		seeds = append(seeds, item)

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list seeds: %w", err)
	}

	if len(seeds) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&seeds).
		On("CONFLICT (name) DO UPDATE").
		Set("kubernetes_version = EXCLUDED.kubernetes_version").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener seeds into db",
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated gardener seeds", "count", count)

	return nil
}
