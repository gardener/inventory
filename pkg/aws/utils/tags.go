// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// FetchTag returns the value of the AWS tag with the key s or an empty string if the tag is not found.
func FetchTag(tags []types.Tag, key string) string {
	for _, t := range tags {
		if strings.Compare(*t.Key, key) == 0 {
			return *t.Value
		}
	}
	return ""
}
