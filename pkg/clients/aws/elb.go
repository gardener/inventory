// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ELBClientset provides the registry of ELB v1 clients.
var ELBClientset = registry.New[string, *Client[*elb.Client]]()
