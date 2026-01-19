// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/clients/db"
)

const (
	hostedZoneIDPrefix  = "/hostedzone/"
	route53AsteriskCode = "\\052"
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

// CutHostedZonePrefix removes the 'hosted-zone' prefix from AWS hosted zone IDs
func CutHostedZonePrefix(s string) string {
	// not interested in whether it was actually found
	result, _ := strings.CutPrefix(s, hostedZoneIDPrefix)

	return result
}

// RestoreAsteriskPrefix checks whether the string starts with a \052 prefix
// and swaps it with an asterisk for internal storage.
// ex: \052.inventory.gardener.com becomes *.inventory.gardener.com
func RestoreAsteriskPrefix(route string) string {
	result, found := strings.CutPrefix(route, route53AsteriskCode)
	if found {
		result = "*" + result
	}

	return result
}

// MaybeSkipRetry wraps known AWS errors with [asynq.SkipRetry], so that the
// tasks from which these errors originate from won't be retried.
func MaybeSkipRetry(err error) error {
	// Do not retry tasks where the API call resulted in errors caused by
	// the caller.
	skipRetryCodes := []smithy.ErrorFault{
		smithy.FaultClient,
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if slices.Contains(skipRetryCodes, apiErr.ErrorFault()) {
			return fmt.Errorf("%w (%w)", err, asynq.SkipRetry)
		}
	}

	return err
}
