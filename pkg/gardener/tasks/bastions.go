// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
	"github.com/gardener/inventory/pkg/gardener/models"
	gutils "github.com/gardener/inventory/pkg/gardener/utils"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectBastions is the name of the task for collecting Gardener
	// Bastions.
	TaskCollectBastions = "g:task:collect-bastions"
)

// CollectBastionsPayload is the payload, which is used for collecting Gardener
// Bastions.
type CollectBastionsPayload struct {
	// Seed is the name of the seed cluster from which to collect Gardener
	// Bastions.
	Seed string `json:"seed" yaml:"seed"`
}

// NewCollectBastionsTask creates a new [asynq.Task] for collecting Gardener
// Bastions, without specifying a payload.
func NewCollectBastionsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectBastions, nil)
}

// HandleCollectBastionsTask is the handler for collecting Gardener Bastions.
func HandleCollectBastionsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting Bastions from all known Gardener Seed clusters.
	data := t.Payload()
	if data == nil {
		return enqueueCollectBastions(ctx)
	}

	var payload CollectBastionsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Seed == "" {
		return asynqutils.SkipRetry(ErrNoSeedCluster)
	}

	return collectBastions(ctx, payload)
}

// enqueueCollectBastions enqueues tasks for collecting Gardener Bastions from
// all known seed clusters.
func enqueueCollectBastions(ctx context.Context) error {
	seeds, err := gutils.GetSeedsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get seeds from db: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Create a task for each known seed cluster
	for _, s := range seeds {
		payload := CollectBastionsPayload{
			Seed: s.Name,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Gardener Bastions",
				"seed", s.Name,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectBastions, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
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

// collectBastions collects the Gardener Bastions from the seed cluster
// specified in the payload.
func collectBastions(ctx context.Context, payload CollectBastionsPayload) error {
	logger := asynqutils.GetLogger(ctx)

	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")

		return nil
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			bastionsDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.Seed,
		)
		key := metrics.Key(TaskCollectBastions, payload.Seed)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger.Info("collecting Gardener bastions", "seed", payload.Seed)

	restConfig, err := gardenerclient.DefaultClient.SeedRestConfig(ctx, payload.Seed)
	if err != nil {
		if errors.Is(err, gardenerclient.ErrSeedIsExcluded) {
			// Don't treat excluded seeds as errors, in order to
			// avoid accumulating archived tasks
			logger.Warn("seed is excluded", "seed", payload.Seed)

			return nil
		}

		return asynqutils.SkipRetry(fmt.Errorf("cannot get rest config for seed %s: %w", payload.Seed, err))
	}

	scheme := runtime.NewScheme()
	err = extensionsv1alpha1.AddToScheme(scheme)
	if err != nil {
		return asynqutils.SkipRetry(fmt.Errorf("could not add Bastion scheme to client for seed %q: %s", payload.Seed, err))
	}

	client, err := crtclient.New(restConfig, crtclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return asynqutils.SkipRetry(fmt.Errorf("cannot create client for bastion: %s", err))
	}

	p := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		var result extensionsv1alpha1.BastionList

		listOpts := crtclient.ListOptions{
			Limit:    opts.Limit,
			Continue: opts.Continue,
		}
		err := client.List(ctx, &result, &listOpts)

		if err != nil {
			if meta.IsNoMatchError(err) {
				logger.Warn(
					"bastion api not found",
					"seed", payload.Seed,
				)

				return nil, asynqutils.SkipRetry(fmt.Errorf("bastion api not found: %w", err))
			}

			return nil, err
		}

		return &result, nil
	})

	items := make([]models.Bastion, 0)

	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		b, ok := obj.(*extensionsv1alpha1.Bastion)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		var ip net.IP
		var hostname string
		if b.Status.Ingress != nil {
			ip = net.ParseIP(b.Status.Ingress.IP)
			hostname = b.Status.Ingress.Hostname
		}

		item := models.Bastion{
			Name:      b.Name,
			Namespace: b.Namespace,
			SeedName:  payload.Seed,
			IP:        ip,
			Hostname:  hostname,
		}
		items = append(items, item)

		return nil
	})

	if err != nil {
		logger.Error(
			"cannot list bastions",
			"seed", payload.Seed,
			"reason", err,
		)

		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, namespace, seed_name) DO UPDATE").
		Set("ip = EXCLUDED.ip").
		Set("hostname = EXCLUDED.hostname").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener bastions into db",
			"seed", payload.Seed,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gardener bastions",
		"seed", payload.Seed,
		"count", count,
	)

	return nil
}
