// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/gardener/models"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// GARDENER_COLLECT_SEEDS_TYPE is the type of the task that collects Gardener seeds.
	GARDENER_COLLECT_SEEDS_TYPE = "g:task:collect-seeds"
)

var ErrMissingSeed = errors.New("missing seed name")

// NewGardenerCollectSeedsTask creates a new task for collecting Gardener seeds.
func NewGardenerCollectSeedsTask() *asynq.Task {
	return asynq.NewTask(GARDENER_COLLECT_SEEDS_TYPE, nil)
}

// HandleGardenerCollectSeedsTask is a handler function that collects Gardener seeds.
func HandleGardenerCollectSeedsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting Gardener seeds")
	return collectSeeds(ctx)
}

func collectSeeds(ctx context.Context) error {
	gardenClient := clients.VirtualGardenClient()
	if gardenClient == nil {
		return fmt.Errorf("could not get garden client: %w", asynq.SkipRetry)
	}

	seeds := make([]models.Seed, 0, 100)
	err := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return gardenClient.CoreV1beta1().Seeds().List(ctx, metav1.ListOptions{})
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
		s, ok := obj.(*v1beta1.Seed)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		seed := models.Seed{
			Name:              s.Name,
			KubernetesVersion: strings.StringFromPointer(s.Status.KubernetesVersion),
		}
		seeds = append(seeds, seed)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list seeds: %w", err)
	}

	if len(seeds) == 0 {
		return nil
	}
	_, err = db.DB.NewInsert().
		Model(&seeds).
		On("CONFLICT (name) DO UPDATE").
		Set("kubernetes_version = EXCLUDED.kubernetes_version").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert gardener seeds into db", "err", err)
		return err
	}

	return nil
}
