// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/gardener/inventory/pkg/core/registry"
)

// S3Clientset provides the registry of S3 clients.
var S3Clientset = registry.New[string, *Client[*s3.Client]]()
