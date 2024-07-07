// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package clients

import "github.com/aws/aws-sdk-go-v2/service/ec2"

var EC2 *ec2.Client

// SetEC2Client sets the AWS EC2 client to be used by workers.
func SetEC2Client(client *ec2.Client) {
	EC2 = client
}
