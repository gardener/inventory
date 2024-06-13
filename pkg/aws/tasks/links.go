package tasks

import (
	"context"
	"log/slog"

	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/uptrace/bun"
)

// LinkAvailabilityZoneWithRegion creates links between the AWS AZs and Regions
func LinkAvailabilityZoneWithRegion(ctx context.Context, db *bun.DB) error {
	var zones []models.AvailabilityZone
	err := db.NewSelect().
		Model(&zones).
		Relation("Region").
		Where("region.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.RegionToAZ, 0, len(zones))
	for _, zone := range zones {
		link := models.RegionToAZ{
			AvailabilityZoneID: zone.ID,
			RegionID:           zone.Region.ID,
		}
		links = append(links, link)
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (region_id, az_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("linked aws region with az", "count", count)

	return err
}

// LinkRegionWithVPC creates links between the AWS Region and VPC
func LinkRegionWithVPC(ctx context.Context, db *bun.DB) error {
	var vpcs []models.VPC
	err := db.NewSelect().
		Model(&vpcs).
		Relation("Region").
		Where("region.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.RegionToVPC, 0, len(vpcs))
	for _, vpc := range vpcs {
		link := models.RegionToVPC{
			VpcID:    vpc.ID,
			RegionID: vpc.Region.ID,
		}
		links = append(links, link)
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (region_id, vpc_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("linked aws region with vpc", "count", count)

	return err
}
