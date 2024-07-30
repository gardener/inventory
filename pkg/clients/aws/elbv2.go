// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

var ELBV2 *elbv2.Client

// SetELBClient sets the AWS ELBV2 client to be used by workers.
func SetELBV2Client(client *elbv2.Client) {
	ELBV2 = client
}
