// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package asynq provides various asynq utilities
package asynq

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/metrics"
)

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
				attrs = append(attrs, slog.String("task_queue", queueName))
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
			logger := GetLogger(ctx)
			logger.Info("received task")
			start := time.Now()
			err := handler.ProcessTask(ctx, task)
			elapsed := time.Since(start)
			logger.Info("task finished", "duration", elapsed)

			return err
		}

		return asynq.HandlerFunc(mw)
	}

	return asynq.MiddlewareFunc(middleware)
}

// NewMetricsMiddleware returns a new [asynq.MiddlewareFunc] which provides
// metrics about task handlers.
func NewMetricsMiddleware() asynq.MiddlewareFunc {
	middleware := func(handler asynq.Handler) asynq.Handler {
		mw := func(ctx context.Context, task *asynq.Task) error {
			taskName := task.Type()
			queueName := GetQueueName(ctx)

			start := time.Now()
			err := handler.ProcessTask(ctx, task)
			elapsed := time.Since(start)

			switch {
			case err == nil:
				// OK
				metrics.TaskSuccessfulTotal.WithLabelValues(taskName, queueName).Inc()
				metrics.TaskDurationSeconds.WithLabelValues(taskName, queueName).Observe(elapsed.Seconds())
			case errors.Is(err, asynq.SkipRetry):
				// Skipped
				metrics.TaskSkippedTotal.WithLabelValues(taskName, queueName).Inc()
			default:
				// Failed
				metrics.TaskFailedTotal.WithLabelValues(taskName, queueName).Inc()
			}

			return err
		}

		return asynq.HandlerFunc(mw)
	}

	return asynq.MiddlewareFunc(middleware)
}
