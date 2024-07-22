// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

var ELB *elb.Client

// SetELBClient sets the AWS ELB client to be used by workers.
func SetELBClient(client *elb.Client) {
	ELB = client
}
