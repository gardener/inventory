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

// LinkShootWithSeed creates the relationship between the Shoot and Seed
func LinkShootWithSeed(ctx context.Context, db *bun.DB) error {
	var shoots []models.Shoot
	err := db.NewSelect().
		Model(&shoots).
		Relation("Seed").
		Where("seed.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.ShootToSeed, 0, len(shoots))
	for _, shoot := range shoots {
		link := models.ShootToSeed{
			ShootID: shoot.ID,
			SeedID:  shoot.Seed.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (shoot_id, seed_id) DO UPDATE").
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

	slog.Info("linked gardener shoot with seed", "count", count)

	return nil
}

// LinkMachineWithShoot creates the relationship between the Machine and Shoot
func LinkMachineWithShoot(ctx context.Context, db *bun.DB) error {
	var machines []models.Machine
	err := db.NewSelect().
		Model(&machines).
		Relation("Shoot").
		Where("shoot.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.MachineToShoot, 0, len(machines))
	for _, machine := range machines {
		link := models.MachineToShoot{
			MachineID: machine.ID,
			ShootID:   machine.Shoot.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (machine_id, shoot_id) DO UPDATE").
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

	slog.Info("linked gardener machine with shoot", "count", count)

	return nil
}
