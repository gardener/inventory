// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"log/slog"

	"github.com/uptrace/bun"

	"github.com/gardener/inventory/pkg/aws/models"
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

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (region_id, az_id) DO UPDATE").
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

	slog.Info("linked aws region with az", "count", count)

	return nil
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

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (region_id, vpc_id) DO UPDATE").
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

	slog.Info("linked aws region with vpc", "count", count)

	return nil
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

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (subnet_id, vpc_id) DO UPDATE").
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

	slog.Info("linked aws subnet with vpc", "count", count)

	return nil
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

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, vpc_id) DO UPDATE").
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

	slog.Info("linked aws instance with vpc", "count", count)

	return nil
}

// LinkSubnetWithAZ creates links between the AZ and Subnets.
func LinkSubnetWithAZ(ctx context.Context, db *bun.DB) error {
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

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (subnet_id, az_id) DO UPDATE").
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

	slog.Info("linked aws subnet with az", "count", count)

	return nil
}

// LinkInstanceWithSubnet creates links between the Instance and Subnet.
func LinkInstanceWithSubnet(ctx context.Context, db *bun.DB) error {
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

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, subnet_id) DO UPDATE").
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

	slog.Info("linked aws instance with subnet", "count", count)

	return nil
}

// LinkInstanceWithRegion creates links between the Instance and Region.
func LinkInstanceWithRegion(ctx context.Context, db *bun.DB) error {
	var instances []models.Instance
	err := db.NewSelect().
		Model(&instances).
		Relation("Region").
		Where("region.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.InstanceToRegion, 0, len(instances))
	for _, instance := range instances {
		link := models.InstanceToRegion{
			InstanceID: instance.ID,
			RegionID:   instance.Region.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, region_id) DO UPDATE").
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

	slog.Info("linked aws instance with region", "count", count)

	return nil
}

func LinkImageWithRegion(ctx context.Context, db *bun.DB) error {
	var images []models.Image
	err := db.NewSelect().
		Model(&images).
		Relation("Region").
		Where("region.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.ImageToRegion, 0, len(images))
	for _, image := range images {
		link := models.ImageToRegion{
			ImageID:  image.ID,
			RegionID: image.Region.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (image_id, region_id) DO UPDATE").
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

	slog.Info("Linked AWS images (AMIs) with region", "count", count)

	return nil
}

func LinkInstanceWithImage(ctx context.Context, db *bun.DB) error {
	var instances []models.Instance
	err := db.NewSelect().
		Model(&instances).
		Relation("Image").
		Where("image.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.InstanceToImage, 0, len(instances))
	for _, instance := range instances {
		link := models.InstanceToImage{
			InstanceID: instance.ID,
			ImageID:    instance.Image.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, image_id) DO UPDATE").
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

	slog.Info("linked aws instance with image", "count", count)

	return nil
}
