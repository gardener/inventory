// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/uptrace/bun"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// TaskConstructor is a function which creates and returns a new [asynq.Task].
type TaskConstructor func() *asynq.Task

// Enqueue enqueues the tasks produced by the given task constructors.
func Enqueue(ctx context.Context, items []TaskConstructor) error {
	logger := asynqutils.GetLogger(ctx)
	for _, fn := range items {
		task := fn()
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"reason", err,
			)
			return err
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
		)
	}

	return nil
}

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
