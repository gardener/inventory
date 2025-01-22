// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"

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
	// TaskCollectProjects is the name of the task for collecting Gardener
	// Projects.
	TaskCollectProjects = "g:task:collect-projects"
)

// NewCollectProjectsTask creates a new [asynq.Task] for collecting
// Gardener projects, without specifying a payload.
func NewCollectProjectsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectProjects, nil)
}

// HandleCollectProjectsTask is the handler that collects Gardener
// projects.
func HandleCollectProjectsTask(ctx context.Context, t *asynq.Task) error {
	client, err := gardenerclient.VirtualGardenClient()
	if err != nil {
		return asynqutils.SkipRetry(ErrNoVirtualGardenClientFound)
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting Gardener projects")
	projects := make([]models.Project, 0)
	members := make([]models.ProjectMember, 0)

	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().Projects().List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		p, ok := obj.(*v1beta1.Project)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		// Collect projects
		projectItem := models.Project{
			Name:              p.Name,
			Namespace:         stringutils.StringFromPointer(p.Spec.Namespace),
			Status:            string(p.Status.Phase),
			Purpose:           stringutils.StringFromPointer(p.Spec.Purpose),
			Owner:             p.Spec.Owner.Name,
			CreationTimestamp: p.CreationTimestamp.Time,
		}
		projects = append(projects, projectItem)

		// Collect project members
		for _, member := range p.Spec.Members {
			memberItem := models.ProjectMember{
				Name:        member.Name,
				Kind:        member.Kind,
				Role:        member.Role,
				ProjectName: p.Name,
			}
			members = append(members, memberItem)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list projects: %w", err)
	}

	// Persist projects
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
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated gardener projects", "count", count)

	// Persist project members
	if len(members) == 0 {
		return nil
	}

	out, err = db.DB.NewInsert().
		Model(&members).
		On("CONFLICT (name, project_name) DO UPDATE").
		Set("kind = EXCLUDED.kind").
		Set("role = EXCLUDED.role").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated gardener project members", "count", count)

	return nil
}
