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
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	stringutils "github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// TaskCollectShoots is the name of the task for collecting Shoots.
	TaskCollectShoots = "g:task:collect-shoots"

	// shootProjectPrefix is the prefix for the shoot project namespace
	shootProjectPrefix = "garden-"
)

// NewCollectShootsTask creates a new [asynq.Task] for collecting
// Gardener shoots, without specifying a payload.
func NewCollectShootsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectShoots, nil)
}

// HandleCollectShootsTask is a handler that collects Gardener Shoots.
func HandleCollectShootsTask(ctx context.Context, t *asynq.Task) error {
	client, err := gardenerclient.VirtualGardenClient()
	if err != nil {
		return asynqutils.SkipRetry(ErrNoVirtualGardenClientFound)
	}

	slog.Info("collecting Gardener shoots")
	shoots := make([]models.Shoot, 0)
	err = pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().Shoots("").List(ctx, metav1.ListOptions{})
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
		s, ok := obj.(*v1beta1.Shoot)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		projectName, _ := strings.CutPrefix(s.Namespace, shootProjectPrefix)
		item := models.Shoot{
			Name:         s.Name,
			TechnicalId:  s.Status.TechnicalID,
			Namespace:    s.Namespace,
			ProjectName:  projectName,
			CloudProfile: s.Spec.CloudProfileName,
			Purpose:      stringutils.StringFromPointer((*string)(s.Spec.Purpose)),
			SeedName:     stringutils.StringFromPointer(s.Spec.SeedName),
			Status:       s.Labels["shoot.gardener.cloud/status"],
			IsHibernated: s.Status.IsHibernated,
			CreatedBy:    s.Annotations["gardener.cloud/created-by"],
		}
		shoots = append(shoots, item)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list shoots: %w", err)
	}

	if len(shoots) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
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
		slog.Error(
			"could not insert gardener shoots into db",
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated gardener shoots", "count", count)

	return nil
}
