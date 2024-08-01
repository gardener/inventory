// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/uptrace/bun"

	"github.com/gardener/inventory/pkg/aws/constants"
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

// LinkImageWithRegion creates links between the Image and Region.
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

// LinkLoadBalancerWithVpc creates links between the LoadBalancer and VPC.
func LinkLoadBalancerWithVpc(ctx context.Context, db *bun.DB) error {
	var lbs []models.LoadBalancer
	err := db.NewSelect().
		Model(&lbs).
		Relation("VPC").
		Where("vpc.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.LoadBalancerToVPC, 0, len(lbs))
	for _, lb := range lbs {
		link := models.LoadBalancerToVPC{
			LoadBalancerID: lb.ID,
			VpcID:          lb.VPC.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (lb_id, vpc_id) DO UPDATE").
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

	slog.Info("linked aws load balancers with VPC", "count", count)

	return nil
}

// LinkLoadBalancerWithRegion creates links between the LoadBalancer and Region.
func LinkLoadBalancerWithRegion(ctx context.Context, db *bun.DB) error {
	var lbs []models.LoadBalancer
	err := db.NewSelect().
		Model(&lbs).
		Relation("Region").
		Where("region.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.LoadBalancerToRegion, 0, len(lbs))
	for _, lb := range lbs {
		link := models.LoadBalancerToRegion{
			LoadBalancerID: lb.ID,
			RegionID:       lb.Region.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (lb_id, region_id) DO UPDATE").
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

	slog.Info("linked aws load balancer with region", "count", count)

	return nil
}

// LinkInstanceWithImage creates links between the Instance and Image.
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

// LinkNetworkInterfaceWithInstance creates links between [models.Instance] and
// [models.NetworkInterface].
func LinkNetworkInterfaceWithInstance(ctx context.Context, db *bun.DB) error {
	var items []models.NetworkInterface
	err := db.NewSelect().
		Model(&items).
		Relation("Instance").
		Where("instance.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.InstanceToNetworkInterface, 0, len(items))
	for _, item := range items {
		link := models.InstanceToNetworkInterface{
			NetworkInterfaceID: item.ID,
			InstanceID:         item.Instance.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, ni_id) DO UPDATE").
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

	slog.Info("linked aws instance with network interface", "count", count)

	return nil
}

// getInterfacesForLoadBalancer retrieves the [models.NetworkInterface]s
// associated with the given [models.LoadBalancer].
func getInterfacesForLoadBalancer(ctx context.Context, db *bun.DB, lb models.LoadBalancer) ([]models.NetworkInterface, error) {
	// In AWS we have a mixture of Load Balancers such as `classic',
	// `network', `application' and `gateway'.
	//
	// In order to determine the Network Interfaces used by these Load
	// Balancers we need to inspect the `description' of each Network
	// Interface, which will be set to a specific value, depending on the
	// type of Load Balancer with which it is associated.
	//
	// Interfaces associated with `classic' Load Balancers have a
	// description set to "ELB <elb-name>".
	//
	// Interfaces associated with Application Load Balancers have a
	// description set to "ELB app/load-balancer-name/load-balancer-id".
	//
	// Interfaces associated with Network Load Balancers have a description
	// set to "ELB net/load-balancer-name/load-balancer-id".
	//
	// NAT Gateways are not covered by this utility function, since we can
	// get the interfaces from the NAT Gateway itself.
	//
	// For more details, please refer to [1].
	//
	// [1]: https://repost.aws/knowledge-center/elb-find-load-balancer-ip

	var description string
	switch lb.Type {
	case constants.LoadBalancerClassicType:
		description = fmt.Sprintf("ELB %s", lb.Name)
	case string(types.LoadBalancerTypeEnumApplication):
		description = fmt.Sprintf("ELB app/%s/%s", lb.Name, lb.LoadBalancerID)
	case string(types.LoadBalancerTypeEnumNetwork):
		description = fmt.Sprintf("ELB net/%s/%s", lb.Name, lb.LoadBalancerID)
	}

	if description == "" {
		return nil, nil
	}

	var items []models.NetworkInterface
	err := db.NewSelect().
		Model(&items).
		Where("description = ?", description).
		Scan(ctx)

	return items, err
}

// LinkNetworkInterfaceWithLoadBalancer creates links between
// [models.NetworkInterface] and [models.LoadBalancer].
func LinkNetworkInterfaceWithLoadBalancer(ctx context.Context, db *bun.DB) error {
	loadBalancers := make([]models.LoadBalancer, 0)
	err := db.NewSelect().
		Model(&loadBalancers).
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.LoadBalancerToNetworkInterface, 0)
	for _, lb := range loadBalancers {
		interfaces, err := getInterfacesForLoadBalancer(ctx, db, lb)
		if err != nil {
			return err
		}

		for _, item := range interfaces {
			link := models.LoadBalancerToNetworkInterface{
				NetworkInterfaceID: item.ID,
				LoadBalancerID:     lb.ID,
			}
			links = append(links, link)
		}
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (lb_id, ni_id) DO UPDATE").
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

	slog.Info("linked aws load balancer with network interface", "count", count)

	return nil
}
