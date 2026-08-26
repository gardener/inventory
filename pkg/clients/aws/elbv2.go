// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ELBv2Clientset provides the registry of ELB v2 clients.
var ELBv2Clientset = registry.New[string, *Client[*elbv2.Client]]()
