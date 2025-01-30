// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/hibiken/asynq"
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
	// TaskCollectMachines is the name of the task for collecting Gardener
	// Machines.
	TaskCollectMachines = "g:task:collect-machines"
)

// CollectMachinesPayload is the payload, which is used for collecting Gardener
// Machines.
type CollectMachinesPayload struct {
	// Seed is the name of the seed cluster from which to collect Gardener
	// Machines.
	Seed string `json:"seed" yaml:"seed"`
}

// NewCollectMachinesTask creates a new [asynq.Task] for collecting Gardener
// Machines, without specifying a payload.
func NewCollectMachinesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectMachines, nil)
}

// HandleCollectMachinesTask is the handler for collecting Gardener Machines.
func HandleCollectMachinesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting Machines from all known Gardener Seed clusters.
	data := t.Payload()
	if data == nil {
		return enqueueCollectMachines(ctx)
	}

	var payload CollectMachinesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Seed == "" {
		return asynqutils.SkipRetry(ErrNoSeedCluster)
	}

	return collectMachines(ctx, payload)
}

// enqueueCollectMachines enqueues tasks for collecting Gardener Machines from
// all known Seed Clusters.
func enqueueCollectMachines(ctx context.Context) error {
	seeds, err := gutils.GetSeedsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get seeds from db: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)

	// Create a task for each known seed cluster
	for _, s := range seeds {
		payload := CollectMachinesPayload{
			Seed: s.Name,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Gardener Machines",
				"seed", s.Name,
				"reason", err,
			)
			continue
		}

		task := asynq.NewTask(TaskCollectMachines, data)
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

// collectMachines collects the Gardener Machines from the Seed Cluster
// specified in the payload.
func collectMachines(ctx context.Context, payload CollectMachinesPayload) error {
	logger := asynqutils.GetLogger(ctx)
	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")
		return nil
	}

	logger.Info("collecting Gardener machines", "seed", payload.Seed)
	client, err := gardenerclient.DefaultClient.MCMClient(ctx, payload.Seed)
	if err != nil {
		if errors.Is(err, gardenerclient.ErrSeedIsExcluded) {
			// Don't treat excluded seeds as errors, in order to
			// avoid accumulating archived tasks
			logger.Warn("seed is excluded", "seed", payload.Seed)
			return nil
		}
		return asynqutils.SkipRetry(fmt.Errorf("cannot get garden client for %q: %s", payload.Seed, err))
	}

	machines := make([]models.Machine, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.MachineV1alpha1().Machines("").List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		m, ok := obj.(*v1alpha1.Machine)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		item := models.Machine{
			Name:              m.Name,
			Namespace:         m.Namespace,
			ProviderId:        m.Spec.ProviderID,
			Status:            string(m.Status.CurrentStatus.Phase),
			Node:              m.Labels["node"],
			SeedName:          payload.Seed,
			CreationTimestamp: m.CreationTimestamp.Time,
		}
		machines = append(machines, item)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list machines for seed %q: %w", payload.Seed, err)
	}

	if len(machines) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&machines).
		On("CONFLICT (name, namespace) DO UPDATE").
		Set("status = EXCLUDED.status").
		Set("node = EXCLUDED.node").
		Set("seed_name = EXCLUDED.seed_name").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener machines into db",
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
		"populated gardener machines",
		"seed", payload.Seed,
		"count", count,
	)

	return nil
}
