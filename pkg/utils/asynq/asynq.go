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

	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"

	"github.com/gardener/inventory/pkg/core/config"
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

// NewDefaultErrorHandler returns an [asynq.ErrorHandlerFunc], which logs the
// task and the reason why it has failed.
func NewDefaultErrorHandler() asynq.ErrorHandlerFunc {
	handler := func(ctx context.Context, task *asynq.Task, err error) {
		// The context we get for the error handler will *not* contain
		// our embedded logger, since it goes through a different path
		// than the one used when enqueuing the task. That's why we need
		// to extract the task id, queue, etc. from it.
		logger := GetLogger(ctx)
		taskID, _ := asynq.GetTaskID(ctx)
		taskName := task.Type()
		queueName, _ := asynq.GetQueueName(ctx)
		retried, _ := asynq.GetRetryCount(ctx)
		logger.Error(
			"task failed",
			"task_id", taskID,
			"task_queue", queueName,
			"task_name", taskName,
			"retry", retried,
			"reason", err,
		)
	}

	return asynq.ErrorHandlerFunc(handler)
}

// GetQueueName returns the queue name from the specified context, if present.
// Otherwise it returns [config.DefaultQueueName].
func GetQueueName(ctx context.Context) string {
	queue, ok := asynq.GetQueueName(ctx)
	if ok {
		return queue
	}

	return config.DefaultQueueName
}

// NewRedisClientOptFromConfig returns an [asynq.RedisClientOpt] from the
// provided [config.RedisConfig] configuration.
func NewRedisClientOptFromConfig(conf config.RedisConfig) asynq.RedisClientOpt {
	// TODO: Handle authentication, TLS, etc.
	opts := asynq.RedisClientOpt{
		Addr: conf.Endpoint,
	}

	return opts
}
