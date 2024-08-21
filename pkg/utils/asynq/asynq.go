// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package asynq provides various asynq utilities
package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"
)

// SkipRetry wraps the provided error with [asynq.SkipRetry] in order to signal
// asynq that the task should not retried.
func SkipRetry(err error) error {
	return fmt.Errorf("%w (%w)", err, asynq.SkipRetry)
}

// Unmarshal unmarshals the given payload data by first attempting to unmarshal
// using [json.Unmarshal], and if not successful then falls back to
// [yaml.Unmarshal].
func Unmarshal(data []byte, v any) error {
	err := json.Unmarshal(data, v)
	if err == nil {
		return nil
	}

	return yaml.Unmarshal(data, v)
}

// NewMeasuringMiddleware returns a new [asynq.MiddlewareFunc] which measures
// the execution of tasks.
func NewMeasuringMiddleware() asynq.MiddlewareFunc {
	middleware := func(handler asynq.Handler) asynq.Handler {
		mw := func(ctx context.Context, task *asynq.Task) error {
			taskID, _ := asynq.GetTaskID(ctx)
			queueName, _ := asynq.GetQueueName(ctx)
			taskName := task.Type()
			slog.Info(
				"received task",
				"id", taskID,
				"queue", queueName,
				"name", taskName,
			)
			start := time.Now()
			err := handler.ProcessTask(ctx, task)
			elapsed := time.Since(start)
			slog.Info(
				"task finished",
				"id", taskID,
				"queue", queueName,
				"name", taskName,
				"duration", elapsed,
			)
			return err
		}

		return asynq.HandlerFunc(mw)
	}

	return asynq.MiddlewareFunc(middleware)
}
