// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "github.com/gardener/inventory/pkg/aws/models"
	_ "github.com/gardener/inventory/pkg/aws/tasks"
	_ "github.com/gardener/inventory/pkg/common/tasks"
	_ "github.com/gardener/inventory/pkg/gardener/models"
	_ "github.com/gardener/inventory/pkg/gardener/tasks"
)
