package tasks

import (
	"context"
	"log/slog"

	"github.com/gardener/inventory/pkg/gardener/models"
	"github.com/uptrace/bun"
)

// LinkShootWithProject creates the relationship between the Gardener Shoot and
// Project.
func LinkShootWithProject(ctx context.Context, db *bun.DB) error {
	var shoots []models.Shoot
	err := db.NewSelect().
		Model(&shoots).
		Relation("Project").
		Where("project.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.ShootToProject, 0, len(shoots))
	for _, shoot := range shoots {
		link := models.ShootToProject{
			ShootID:   shoot.ID,
			ProjectID: shoot.Project.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (shoot_id, project_id) DO UPDATE").
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

	slog.Info("linked gardener shoot with project", "count", count)

	return nil
}
