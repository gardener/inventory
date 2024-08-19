// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"
	"log/slog"

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
	// TaskCollectProjects is the name of the task for collecting Gardener
	// Projects.
	TaskCollectProjects = "g:task:collect-projects"
)

// NewGardenerCollectProjectsTask creates a new [asynq.Task] for collecting
// Gardener projects, without specifying a payload.
func NewGardenerCollectProjectsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectProjects, nil)
}

// HandleGardenerCollectProjectsTask is a handler function that collects Gardener projects.
func HandleGardenerCollectProjectsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting Gardener projects")
	return collectProjects(ctx)
}

// collectProjects collects the Gardener Projects.
func collectProjects(ctx context.Context) error {
	gardenClient, err := gardenerclient.VirtualGardenClient()
	if err != nil {
		return asynqutils.SkipRetry(ErrNoVirtualGardenClientFound)
	}

	projects := make([]models.Project, 0)
	err = pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return gardenClient.CoreV1beta1().Projects().List(ctx, opts)
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
		p, ok := obj.(*v1beta1.Project)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		item := models.Project{
			Name:      p.Name,
			Namespace: stringutils.StringFromPointer(p.Spec.Namespace),
			Status:    string(p.Status.Phase),
			Purpose:   stringutils.StringFromPointer(p.Spec.Purpose),
			Owner:     p.Spec.Owner.Name,
		}
		projects = append(projects, item)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list projects: %w", err)
	}

	if len(projects) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&projects).
		On("CONFLICT (name) DO UPDATE").
		Set("namespace = EXCLUDED.namespace").
		Set("status = EXCLUDED.status").
		Set("purpose = EXCLUDED.purpose").
		Set("owner = EXCLUDED.owner").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error(
			"could not insert gardener projects into db",
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated gardener projects", "count", count)

	return nil
}
