package tasks

import (
	"context"

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

	_, err = db.NewInsert().
		Model(&links).
		On("CONFLICT (region_id, az_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	return err
}
