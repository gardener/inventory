package tasks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/inventory/pkg/clients"
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
	seedList, err := gardenClient.CoreV1beta1().Seeds().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	seeds := make([]models.Seed, 0, len(seedList.Items))
	for _, s := range seedList.Items {
		seed := models.Seed{
			Name:              s.Name,
			KubernetesVersion: strings.StringFromPointer(s.Status.KubernetesVersion),
		}
		seeds = append(seeds, seed)
	}
	if len(seeds) == 0 {
		return nil
	}
	_, err = clients.Db.NewInsert().
		Model(&seeds).
		On("CONFLICT (name) DO UPDATE").
		Set("kubernetes_version = EXCLUDED.kubernetes_version").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert gardener seeds into db", "err", err)
		return err
	}

	return nil
}
