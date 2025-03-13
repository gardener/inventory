// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "github.com/gardener/inventory/pkg/auxiliary/tasks"
	_ "github.com/gardener/inventory/pkg/aws/models"
	_ "github.com/gardener/inventory/pkg/aws/tasks"
	_ "github.com/gardener/inventory/pkg/azure/models"
	_ "github.com/gardener/inventory/pkg/azure/tasks"
	_ "github.com/gardener/inventory/pkg/gardener/models"
	_ "github.com/gardener/inventory/pkg/gardener/tasks"
	_ "github.com/gardener/inventory/pkg/gcp/models"
	_ "github.com/gardener/inventory/pkg/gcp/tasks"
	_ "github.com/gardener/inventory/pkg/openstack/models"
	_ "github.com/gardener/inventory/pkg/openstack/tasks"
)
