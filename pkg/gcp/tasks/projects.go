// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// TaskCollectProjects is the name of the task for collecting GCP Projects
const TaskCollectProjects = "gcp:task:collect-projects"

// NewCollectProjectsTask creates a new [asynq.Task] for collecting GCP
// Projects, without specifying a payload.
func NewCollectProjectsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectProjects, nil)
}

// HandleCollectProjectsTask is the handler, which collects GCP projects
func HandleCollectProjectsTask(ctx context.Context, t *asynq.Task) error {
	logger := asynqutils.GetLogger(ctx)
	if gcpclients.ProjectsClientset.Length() == 0 {
		logger.Warn("no GCP clients found")
		return nil
	}

	items := make([]models.Project, 0, gcpclients.ProjectsClientset.Length())
	err := gcpclients.ProjectsClientset.Range(func(projectID string, client *gcpclients.Client[*resourcemanager.ProjectsClient]) error {
		logger.Info("collecting GCP project", "project", projectID)
		req := &resourcemanagerpb.GetProjectRequest{
			Name: fmt.Sprintf("projects/%s", projectID),
		}
		p, err := client.Client.GetProject(ctx, req)
		if err != nil {
			logger.Error(
				"failed to get GCP project",
				"project", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}
		item := models.Project{
			Name:              p.Name,
			Parent:            p.Parent,
			State:             p.State.String(),
			ProjectID:         p.ProjectId,
			DisplayName:       p.DisplayName,
			Etag:              p.Etag,
			ProjectCreateTime: p.CreateTime.AsTime(),
			ProjectUpdateTime: p.UpdateTime.AsTime(),
			ProjectDeleteTime: p.DeleteTime.AsTime(),
		}
		items = append(items, item)

		return nil
	})

	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("parent = EXCLUDED.parent").
		Set("state = EXCLUDED.state").
		Set("display_name = EXCLUDED.display_name").
		Set("etag = EXCLUDED.etag").
		Set("project_create_time = EXCLUDED.project_create_time").
		Set("project_update_time = EXCLUDED.project_update_time").
		Set("project_delete_time = EXCLUDED.project_delete_time").
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

	logger.Info("populated gcp projects", "count", count)

	return nil
}
