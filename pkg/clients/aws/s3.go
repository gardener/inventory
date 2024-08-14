// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/gardener/inventory/pkg/core/registry"
)

// S3Clients provides the registry of [*s3.Client] clients.
var S3Clients = registry.New[string, *s3.Client]()
