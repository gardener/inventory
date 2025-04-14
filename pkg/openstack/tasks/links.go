// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/gardener/inventory/pkg/openstack/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// LinkSubnetsWithNetworks creates links between the OpenStack Subnets and Networks
func LinkSubnetsWithNetworks(ctx context.Context, db *bun.DB) error {
	var subnets []models.Subnet
	err := db.NewSelect().
		Model(&subnets).
		Relation("Network").
		Where("network.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.SubnetToNetwork, 0, len(subnets))
	for _, subnet := range subnets {
		links = append(links, models.SubnetToNetwork{
			SubnetID:  subnet.ID,
			NetworkID: subnet.Network.ID,
		})
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (subnet_id, network_id) DO UPDATE").
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
	logger.Info("linked openstack subnets with networks", "count", count)

	return nil
}

// LinkLoadBalancersWithSubnets creates links between the OpenStack LoadBalancers and Subnets
func LinkLoadBalancersWithSubnets(ctx context.Context, db *bun.DB) error {
	var loadbalancers []models.LoadBalancer
	err := db.NewSelect().
		Model(&loadbalancers).
		Relation("Subnet").
		Where("subnet.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.LoadBalancerToSubnet, 0, len(loadbalancers))
	for _, lb := range loadbalancers {
		links = append(links, models.LoadBalancerToSubnet{
			LoadBalancerID: lb.ID,
			SubnetID:       lb.Subnet.ID,
		})
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (lb_id, subnet_id) DO UPDATE").
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
	logger.Info("linked openstack load balancers with subnets", "count", count)

	return nil
}

// LinkServersWithProjects creates links between the OpenStack Servers and Projects
func LinkServersWithProjects(ctx context.Context, db *bun.DB) error {
	var servers []models.Server
	err := db.NewSelect().
		Model(&servers).
		Relation("Project").
		Where("project.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.ServerToProject, 0, len(servers))
	for _, server := range servers {
		links = append(links, models.ServerToProject{
			ServerID:  server.ID,
			ProjectID: server.Project.ID,
		})
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (server_id, project_id) DO UPDATE").
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
	logger.Info("linked openstack servers with projects", "count", count)

	return nil
}
