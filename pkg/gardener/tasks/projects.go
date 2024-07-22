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
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// GARDENER_COLLECT_PROJECTS_TYPE is the type of the task that collects Gardener projects.
	GARDENER_COLLECT_PROJECTS_TYPE = "g:task:collect-projects"
)

// NewGardenerCollectProjectsTask creates a new task for collecting Gardener projects.
func NewGardenerCollectProjectsTask() *asynq.Task {
	return asynq.NewTask(GARDENER_COLLECT_PROJECTS_TYPE, nil)
}

// HandleGardenerCollectProjectsTask is a handler function that collects Gardener projects.
func HandleGardenerCollectProjectsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting Gardener projects")
	return collectProjects(ctx)
}

func collectProjects(ctx context.Context) error {
	gardenClient := gardenerclient.VirtualGardenClient()
	if gardenClient == nil {
		return fmt.Errorf("could not get garden client: %w", asynq.SkipRetry)
	}

	projects := make([]models.Project, 0, 100)
	err := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return gardenClient.CoreV1beta1().Projects().List(ctx, opts)
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
		p, ok := obj.(*v1beta1.Project)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		project := models.Project{
			Name:      p.Name,
			Namespace: strings.StringFromPointer(p.Spec.Namespace),
			Status:    string(p.Status.Phase),
			Purpose:   strings.StringFromPointer(p.Spec.Purpose),
			Owner:     p.Spec.Owner.Name,
		}
		projects = append(projects, project)
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list projects: %w", err)
	}

	if len(projects) == 0 {
		return nil
	}
	_, err = db.DB.NewInsert().
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
		slog.Error("could not insert gardener projects into db", "err", err)
		return err
	}

	return nil
}
