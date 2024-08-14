// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ELBv2Clients provides the registry of [*elbv2.Client] clients.
var ELBv2Clients = registry.New[string, *elbv2.Client]()
