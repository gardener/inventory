// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
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

func getCloudProfileName(s v1beta1.Shoot) (string, error) {
	if s.Spec.CloudProfile != nil {
		return s.Spec.CloudProfile.Name, nil
	}

	if s.Spec.CloudProfileName != nil {
		return *s.Spec.CloudProfileName, nil
	}

	return "", fmt.Errorf("No cloud profile name found for shoot %s", s.Name)
}

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

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting Gardener shoots")
	shoots := make([]models.Shoot, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().Shoots("").List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		s, ok := obj.(*v1beta1.Shoot)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		projectName, _ := strings.CutPrefix(s.Namespace, shootProjectPrefix)
		cloudProfileName, err := getCloudProfileName(*s)
		if err != nil {
			logger.Error(
				"cannot extract shoot",
				"reason", err,
			)
			return err
		}

		item := models.Shoot{
			Name:              s.Name,
			TechnicalId:       s.Status.TechnicalID,
			Namespace:         s.Namespace,
			ProjectName:       projectName,
			CloudProfile:      cloudProfileName,
			Purpose:           stringutils.StringFromPointer((*string)(s.Spec.Purpose)),
			SeedName:          stringutils.StringFromPointer(s.Spec.SeedName),
			Status:            s.Labels["shoot.gardener.cloud/status"],
			IsHibernated:      s.Status.IsHibernated,
			CreatedBy:         s.Annotations["gardener.cloud/created-by"],
			Region:            s.Spec.Region,
			KubernetesVersion: s.Spec.Kubernetes.Version,
			CreationTimestamp: s.CreationTimestamp.Time,
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
		Set("region = EXCLUDED.region").
		Set("k8s_version = EXCLUDED.k8s_version").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener shoots into db",
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated gardener shoots", "count", count)

	return nil
}
