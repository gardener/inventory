// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/models"
	utils "github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// GARDENER_COLLECT_SHOOTS_TYPE is the type of the task that collects Gardener shoots.
	GARDENER_COLLECT_SHOOTS_TYPE = "g:task:collect-shoots"
)

// NewGardenerCollectShootsTask creates a new task for collecting Gardener shoots.
func NewGardenerCollectShootsTask() *asynq.Task {
	return asynq.NewTask(GARDENER_COLLECT_SHOOTS_TYPE, nil)
}

// HandleGardenerCollectShootsTask is a handler function that collects Gardener shoots.
func HandleGardenerCollectShootsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting Gardener shoots")
	return collectShoots(ctx)
}

func collectShoots(ctx context.Context) error {
	gardenClient := gardenerclient.VirtualGardenClient()
	if gardenClient == nil {
		return fmt.Errorf("could not get garden client: %w", asynq.SkipRetry)
	}

	shoots := make([]models.Shoot, 0, 100)
	err := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return gardenerclient.VirtualGardenClient().CoreV1beta1().Shoots("").List(ctx, metav1.ListOptions{})
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
		s, ok := obj.(*v1beta1.Shoot)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		projectName, _ := strings.CutPrefix(s.Namespace, "garden-")
		shoot := models.Shoot{
			Name:         s.Name,
			TechnicalId:  s.Status.TechnicalID,
			Namespace:    s.Namespace,
			ProjectName:  projectName,
			CloudProfile: s.Spec.CloudProfileName,
			Purpose:      utils.StringFromPointer((*string)(s.Spec.Purpose)),
			SeedName:     utils.StringFromPointer(s.Spec.SeedName),
			Status:       s.Labels["shoot.gardener.cloud/status"],
			IsHibernated: s.Status.IsHibernated,
			CreatedBy:    s.Annotations["gardener.cloud/created-by"],
		}
		shoots = append(shoots, shoot)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list shoots: %w", err)
	}

	if len(shoots) == 0 {
		return nil
	}
	_, err = db.DB.NewInsert().
		Model(&shoots).
		On("CONFLICT (technical_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("namespace = EXCLUDED.namespace").
		Set("project_name = EXCLUDED.project_name").
		Set("cloud_profile = EXCLUDED.cloud_profile").
		Set("purpose = EXCLUDED.purpose").
		Set("seed_name = EXCLUDED.seed_name").
		Set("status = EXCLUDED.status").
		Set("is_hibernated = EXCLUDED.is_hibernated").
		Set("created_by = EXCLUDED.created_by").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert gardener shoots into db", "err", err)
		return err
	}

	return nil
}
