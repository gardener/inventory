// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/route53"

	"github.com/gardener/inventory/pkg/core/registry"
)

// Route53Clientset provides the registry of Route53 clients.
var Route53Clientset = registry.New[string, *Client[*route53.Client]]()
