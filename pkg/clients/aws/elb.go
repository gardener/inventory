// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ELBClientset provides the registry of ELB v1 clients.
var ELBClientset = registry.New[string, *Client[*elb.Client]]()
