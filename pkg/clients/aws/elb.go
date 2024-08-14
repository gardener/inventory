// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ELBClients provides the registry of [*elb.Client] clients.
var ELBClients = registry.New[string, *elb.Client]()
