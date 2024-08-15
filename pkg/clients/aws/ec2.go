// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// EC2Clientset provides the registry of EC2 clients.
var EC2Clientset = registry.New[string, *Client[*ec2.Client]]()
