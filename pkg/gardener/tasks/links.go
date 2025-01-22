// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/gardener/inventory/pkg/gardener/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
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

	logger := asynqutils.GetLogger(ctx)
	logger.Info("linked gardener shoot with project", "count", count)

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

	logger := asynqutils.GetLogger(ctx)
	logger.Info("linked gardener shoot with seed", "count", count)

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

	logger := asynqutils.GetLogger(ctx)
	logger.Info("linked gardener machine with shoot", "count", count)

	return nil
}

// LinkAWSImageWithCloudProfile creates the relationship between the CloudProfileAWSImage and CloudProfile
func LinkAWSImageWithCloudProfile(ctx context.Context, db *bun.DB) error {
	var awsImages []models.CloudProfileAWSImage
	err := db.NewSelect().
		Model(&awsImages).
		Relation("CloudProfile").
		Where("cloud_profile.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.AWSImageToCloudProfile, 0, len(awsImages))
	for _, image := range awsImages {
		link := models.AWSImageToCloudProfile{
			AWSImageID:     image.ID,
			CloudProfileID: image.CloudProfile.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (aws_image_id, cloud_profile_id) DO UPDATE").
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
	logger.Info("linked gardener cloud profile aws image with cloud profile", "count", count)

	return nil
}

// LinkGCPImageWithCloudProfile creates the relationship between the CloudProfileGCPImage and CloudProfile
func LinkGCPImageWithCloudProfile(ctx context.Context, db *bun.DB) error {
	var gcpImages []models.CloudProfileGCPImage
	err := db.NewSelect().
		Model(&gcpImages).
		Relation("CloudProfile").
		Where("cloud_profile.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.GCPImageToCloudProfile, 0, len(gcpImages))
	for _, image := range gcpImages {
		link := models.GCPImageToCloudProfile{
			GCPImageID:     image.ID,
			CloudProfileID: image.CloudProfile.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (gcp_image_id, cloud_profile_id) DO UPDATE").
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
	logger.Info("linked gardener cloud profile gcp image with cloud profile", "count", count)

	return nil
}

// LinkAzureImageWithCloudProfile creates the relationship between the CloudProfileAzureImage and CloudProfile
func LinkAzureImageWithCloudProfile(ctx context.Context, db *bun.DB) error {
	var azureImages []models.CloudProfileAzureImage
	err := db.NewSelect().
		Model(&azureImages).
		Relation("CloudProfile").
		Where("cloud_profile.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.AzureImageToCloudProfile, 0, len(azureImages))
	for _, image := range azureImages {
		link := models.AzureImageToCloudProfile{
			AzureImageID:   image.ID,
			CloudProfileID: image.CloudProfile.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (azure_image_id, cloud_profile_id) DO UPDATE").
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
	logger.Info("linked gardener cloud profile azure image with cloud profile", "count", count)

	return nil
}

// LinkProjectWithMember creates the relationship between the [models.Project]
// and [models.ProjectMember] models.
func LinkProjectWithMember(ctx context.Context, db *bun.DB) error {
	var items []models.ProjectMember
	err := db.NewSelect().
		Model(&items).
		Relation("Project").
		Where("project.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.ProjectToMember, 0, len(items))
	for _, item := range items {
		link := models.ProjectToMember{
			MemberID:  item.ID,
			ProjectID: item.Project.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (project_id, member_id) DO UPDATE").
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
	logger.Info("linked gardener project with member", "count", count)

	return nil
}
