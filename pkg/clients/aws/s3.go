// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import "github.com/aws/aws-sdk-go-v2/service/s3"

var S3 *s3.Client

// SetS3Client sets the AWS S3 client to be used by workers.
func SetS3Client(client *s3.Client) {
	S3 = client
}
