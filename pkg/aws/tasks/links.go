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

// LinkSubnetWithVPC creates links between the AWS Subnet and VPC
func LinkSubnetWithVPC(ctx context.Context, db *bun.DB) error {
	var subnets []models.Subnet
	err := db.NewSelect().
		Model(&subnets).
		Relation("VPC").
		Where("vpc.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.VPCToSubnet, 0, len(subnets))
	for _, subnet := range subnets {
		link := models.VPCToSubnet{
			SubnetID: subnet.ID,
			VpcID:    subnet.VPC.ID,
		}
		links = append(links, link)
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (subnet_id, vpc_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("linked aws subnet with vpc", "count", count)

	return err
}

// LinkInstanceWithVPC creates links between the AWS VPC and Instance.
func LinkInstanceWithVPC(ctx context.Context, db *bun.DB) error {
	var instances []models.Instance
	err := db.NewSelect().
		Model(&instances).
		Relation("VPC").
		Where("vpc.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.VPCToInstance, 0, len(instances))
	for _, instance := range instances {
		link := models.VPCToInstance{
			InstanceID: instance.ID,
			VpcID:      instance.VPC.ID,
		}
		links = append(links, link)
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, vpc_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("linked aws instance with vpc", "count", count)

	return err
}

// LinkSubnetToAZ creates links between the AZ and Subnets.
func LinkSubnetToAZ(ctx context.Context, db *bun.DB) error {
	var subnets []models.Subnet
	err := db.NewSelect().
		Model(&subnets).
		Relation("AvailabilityZone").
		Where("availability_zone.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.SubnetToAZ, 0, len(subnets))
	for _, subnet := range subnets {
		link := models.SubnetToAZ{
			SubnetID:           subnet.ID,
			AvailabilityZoneID: subnet.AvailabilityZone.ID,
		}
		links = append(links, link)
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (subnet_id, az_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("linked aws subnet with az", "count", count)

	return err
}

// LinkInstanceToSubnet creates links between the Instance and Subnet.
func LinkInstanceToSubnet(ctx context.Context, db *bun.DB) error {
	var instances []models.Instance
	err := db.NewSelect().
		Model(&instances).
		Relation("Subnet").
		Where("subnet.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.InstanceToSubnet, 0, len(instances))
	for _, instance := range instances {
		link := models.InstanceToSubnet{
			InstanceID: instance.ID,
			SubnetID:   instance.Subnet.ID,
		}
		links = append(links, link)
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, subnet_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("linked aws instance with subnet", "count", count)

	return err
}
