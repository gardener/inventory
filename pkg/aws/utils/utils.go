// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/clients/db"
)

const (
	hostedZoneIdPrefix = "/hostedzone/"
)

// FetchTag returns the value of the AWS tag with the key s or an empty string if the tag is not found.
func FetchTag(tags []types.Tag, key string) string {
	for _, t := range tags {
		if t.Key == nil {
			continue
		}
		if strings.Compare(*t.Key, key) == 0 {
			return *t.Value
		}
	}

	return ""
}

// GetRegionsFromDB gets the AWS Regions from the database.
func GetRegionsFromDB(ctx context.Context) ([]models.Region, error) {
	items := make([]models.Region, 0)
	err := db.DB.NewSelect().Model(&items).Scan(ctx)

	return items, err
}

func CutHostedZonePrefix(s string) string {
	// not interested in whether it was actually found
	result, _ := strings.CutPrefix(s, hostedZoneIdPrefix)
	return result
}
