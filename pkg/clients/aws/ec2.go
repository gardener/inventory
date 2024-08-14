// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// EC2Clients provides the registry of [*ec2.Client] clients.
var EC2Clients = registry.New[string, *ec2.Client]()
