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

// loggerKey is the key used to store a [slog.Logger] in a [context.Context]
type loggerKey struct{}

// GetLogger returns the [slog.Logger] instance from the provided context, if
// found, or [slog.DefaultLogger] otherwise.
func GetLogger(ctx context.Context) *slog.Logger {
	value := ctx.Value(loggerKey{})
	logger, ok := value.(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return logger
}

// NewLoggerMiddleware returns a new [asynq.MiddlewareFunc], which embeds a
// [slog.Logger] in the context provided to task handlers.
func NewLoggerMiddleware(logger *slog.Logger) asynq.MiddlewareFunc {
	middleware := func(handler asynq.Handler) asynq.Handler {
		mw := func(ctx context.Context, task *asynq.Task) error {
			// Add the task id, queue and task name as default
			// attributes to each log event.
			attrs := make([]slog.Attr, 0)
			taskID, ok := asynq.GetTaskID(ctx)
			if ok {
				attrs = append(attrs, slog.String("task_id", taskID))
			}

			queueName, ok := asynq.GetQueueName(ctx)
			if ok {
				attrs = append(attrs, slog.String("queue", queueName))
			}

			taskName := task.Type()
			attrs = append(attrs, slog.String("task_name", taskName))
			logHandler := logger.Handler().WithAttrs(attrs)
			newLogger := slog.New(logHandler)
			newCtx := context.WithValue(ctx, loggerKey{}, newLogger)
			return handler.ProcessTask(newCtx, task)
		}
		return asynq.HandlerFunc(mw)
	}

	return asynq.MiddlewareFunc(middleware)
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
