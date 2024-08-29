// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/gardener/inventory/pkg/gcp/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// LinkInstanceWithProject creates links between the [models.Instance] and
// [models.Project] models.
func LinkInstanceWithProject(ctx context.Context, db *bun.DB) error {
	var items []models.Instance
	err := db.NewSelect().
		Model(&items).
		Relation("Project").
		Where("project.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.InstanceToProject, 0, len(items))
	for _, item := range items {
		link := models.InstanceToProject{
			ProjectID:  item.Project.ID,
			InstanceID: item.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (project_id, instance_id) DO UPDATE").
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

	logger := asynqutils.GetLogger(ctx)
	logger.Info("linked gcp instance with project", "count", count)

	return nil
}

// LinkVPCWithProject creates links between the [models.VPC] and
// [models.Project] models.
func LinkVPCWithProject(ctx context.Context, db *bun.DB) error {
	var items []models.VPC
	err := db.NewSelect().
		Model(&items).
		Relation("Project").
		Where("project.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.VPCToProject, 0, len(items))
	for _, item := range items {
		link := models.VPCToProject{
			ProjectID: item.Project.ID,
			VPCID:     item.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (project_id, vpc_id) DO UPDATE").
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

	logger := asynqutils.GetLogger(ctx)
	logger.Info("linked gcp vpc with project", "count", count)

	return nil
}
