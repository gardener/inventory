// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/uptrace/bun"

	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// LinkFunction is a function, which establishes relationships between models.
type LinkFunction func(ctx context.Context, db *bun.DB) error

// LinkObjects links objects by using the provided [LinkFunction] items.
func LinkObjects(ctx context.Context, db *bun.DB, items []LinkFunction) error {
	logger := asynqutils.GetLogger(ctx)
	for _, linkFunc := range items {
		if err := linkFunc(ctx, db); err != nil {
			logger.Error("failed to link objects", "reason", err)
			continue
		}
	}

	return nil
}
