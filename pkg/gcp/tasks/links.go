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

// LinkAddressWithProject creates links between the [models.Address] and
// [models.Project] models.
func LinkAddressWithProject(ctx context.Context, db *bun.DB) error {
	var items []models.Address
	err := db.NewSelect().
		Model(&items).
		Relation("Project").
		Where("project.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.AddressToProject, 0, len(items))
	for _, item := range items {
		link := models.AddressToProject{
			ProjectID: item.Project.ID,
			AddressID: item.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (project_id, address_id) DO UPDATE").
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
	logger.Info("linked gcp address with project", "count", count)

	return nil
}

// LinkInstanceWithNetworkInterface creates links between the
// [models.NetworkInterface] and [models.Instance] models.
func LinkInstanceWithNetworkInterface(ctx context.Context, db *bun.DB) error {
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
			InstanceID:         item.Instance.ID,
			NetworkInterfaceID: item.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, nic_id) DO UPDATE").
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
	logger.Info("linked gcp instance with network interface", "count", count)

	return nil
}

// LinkSubnetWithVPC creates links between the [models.Subnet] and
// [models.VPC] models.
func LinkSubnetWithVPC(ctx context.Context, db *bun.DB) error {
	var items []models.Subnet
	err := db.NewSelect().
		Model(&items).
		Relation("VPC").
		Where("vpc.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.SubnetToVPC, 0, len(items))
	for _, item := range items {
		link := models.SubnetToVPC{
			SubnetID: item.ID,
			VPCID:    item.VPC.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (vpc_id, subnet_id) DO UPDATE").
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
	logger.Info("linked gcp subnet with vpc", "count", count)

	return nil
}

// LinkSubnetWithProject creates links between the [models.Subnet] and
// [models.Project] models.
func LinkSubnetWithProject(ctx context.Context, db *bun.DB) error {
	var items []models.Subnet
	err := db.NewSelect().
		Model(&items).
		Relation("Project").
		Where("project.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.SubnetToProject, 0, len(items))
	for _, item := range items {
		link := models.SubnetToProject{
			SubnetID:  item.ID,
			ProjectID: item.Project.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (project_id, subnet_id) DO UPDATE").
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
	logger.Info("linked gcp subnet with project", "count", count)

	return nil
}

// LinkForwardingRuleWithProject creates links between the
// [models.ForwardingRule] and [models.Project] models.
func LinkForwardingRuleWithProject(ctx context.Context, db *bun.DB) error {
	var items []models.ForwardingRule
	err := db.NewSelect().
		Model(&items).
		Relation("Project").
		Where("project.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.ForwardingRuleToProject, 0, len(items))
	for _, item := range items {
		link := models.ForwardingRuleToProject{
			ProjectID: item.Project.ID,
			RuleID:    item.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (project_id, rule_id) DO UPDATE").
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
	logger.Info("linked gcp forwarding rule with project", "count", count)

	return nil
}

// LinkInstanceWithDisk creates links between the
// [models.Instance] and [models.Disk] models.
func LinkInstanceWithDisk(ctx context.Context, db *bun.DB) error {
	var items []models.AttachedDisk
	err := db.NewSelect().
		Model(&items).
		Relation("Instance").
		Relation("Disk").
		Where("instance.id IS NOT NULL AND disk.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.InstanceToDisk, 0, len(items))
	for _, item := range items {
		link := models.InstanceToDisk{
			InstanceID: item.Instance.ID,
			DiskID:     item.Disk.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (instance_id, disk_id) DO UPDATE").
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
	logger.Info("linked gcp instance with disk", "count", count)

	return nil
}
