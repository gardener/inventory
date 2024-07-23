// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/models"
)

const (
	// GARDENER_COLLECT_MACHINES_TYPE is the type of the task that collects Gardener machines.
	GARDENER_COLLECT_MACHINES_TYPE = "g:task:collect-machines"

	// GARDENER_COLLECT_MACHINES_SEED_TYPE is the type of the task that collects Gardener machines for a given seed.
	GARDENER_COLLECT_MACHINES_SEED_TYPE = "g:task:collect-machines-seed"
)

// CollectMachinesPayload is the payload for collecting Machines for a given Gardener seed.
type CollectMachinesPayload struct {
	Seed string `json:"seed"`
}

// NewGardenerCollectMachinesTask creates a new task for collecting Gardener machines.
func NewGardenerCollectMachinesTask() *asynq.Task {
	return asynq.NewTask(GARDENER_COLLECT_MACHINES_TYPE, nil)
}

// HandleGardenerCollectMachinesTask is a handler function that collects Gardener machines.
func HandleGardenerCollectMachinesTask(ctx context.Context, t *asynq.Task) error {
	return collectMachines(ctx)
}

func collectMachines(ctx context.Context) error {
	slog.Info("Collecting Gardener Machines")
	seeds := make([]models.Seed, 0)
	err := db.DB.NewSelect().Model(&seeds).Scan(ctx)
	if err != nil {
		slog.Error("could not select seeds from db", "err", err)
		return err
	}
	for _, s := range seeds {
		// Trigger Asynq task for each region
		machineTask, err := NewGardenerCollectMachinesForSeed(s.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := asynqclient.Client.Enqueue(machineTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", machineTask.Type(), "reason", err)
			continue
		}

		slog.Info("enqueued task", "type", machineTask.Type(), "id", info.ID, "queue", info.Queue)
	}
	return nil
}

// NewGardenerCollectMachinesForSeed creates a new task for collecting Gardener machines for a given seed.
func NewGardenerCollectMachinesForSeed(seed string) (*asynq.Task, error) {
	if seed == "" {
		return nil, ErrMissingSeed
	}

	payload, err := json.Marshal(CollectMachinesPayload{Seed: seed})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(GARDENER_COLLECT_MACHINES_SEED_TYPE, payload), nil
}

// HandleGardenerCollectMachinesForSeedTask is a handler function that collects Gardener machines for a given seed.
func HandleGardenerCollectMachinesForSeedTask(ctx context.Context, t *asynq.Task) error {
	var p CollectMachinesPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return collectMachinesForSeed(ctx, p.Seed)
}

func collectMachinesForSeed(ctx context.Context, seed string) error {
	slog.Info("Collecting Gardener machines for seed", "seed", seed)

	gardenClient, err := gardenerclient.MCMClient(seed)
	if err != nil {
		return fmt.Errorf("could not get garden client for seed %q: %s: %w", seed, err, asynq.SkipRetry)
	}

	machines := make([]models.Machine, 0)
	err = pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return gardenClient.MachineV1alpha1().Machines("").List(ctx, opts)
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
		m, ok := obj.(*v1alpha1.Machine)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		machine := models.Machine{
			Name:       m.Name,
			Namespace:  m.Namespace,
			ProviderId: m.Spec.ProviderID,
			Status:     string(m.Status.CurrentStatus.Phase),
		}
		machines = append(machines, machine)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list machines for seed %q: %w", seed, err)
	}
	if len(machines) == 0 {
		return nil
	}
	_, err = db.DB.NewInsert().
		Model(&machines).
		On("CONFLICT (name, namespace) DO UPDATE").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert gardener machines into db", "err", err)
		return err
	}

	return nil
}
