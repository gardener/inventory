// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"errors"
	"time"

	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/registry"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// HousekeeperTaskType is the name of the task responsible for cleaning
	// up stale records from the database.
	HousekeeperTaskType = "common:task:housekeeper"
	// DeleteArchivedTaskType is the name of the task responsible for deleting
	// archived tasks from a task queue
	DeleteArchivedTaskType = "common:task:delete-archived-tasks"
	// DeleteCompletedTaskType is the name of the task responsible for deleting
	// completed tasks from a task queue
	DeleteCompletedTaskType = "common:task:delete-completed-tasks"
)

// HousekeeperPayload represents the payload of the housekeeper task.
type HousekeeperPayload struct {
	// Retention provides the retention configuration of objects.
	Retention []RetentionConfig `yaml:"retention"`
}

// DeleteQueuePayload represents the payload of a task management task.
type DeleteQueuePayload struct {
	// Name of the queue that holds the tasks.
	Queue string `yaml:"queue"`
}

// RetentionConfig represents the retention configuration for a given model.
type RetentionConfig struct {
	// Name specifies the model name.
	Name string `yaml:"name" json:"name"`

	// Duration specifies the max duration for which an object will be kept,
	// if it hasn't been updated recently.
	//
	// For example:
	//
	// UpdatedAt field for an object is set to: Thu May 30 16:00:00 EEST 2024
	// Duration of the object is configured to: 4 hours
	//
	// If the object is not update anymore by the time the housekeeper runs,
	// after 20:00:00 this object will be considered as stale and removed
	// from the database.
	Duration time.Duration `yaml:"duration" json:"duration"`
}

// HandleHousekeeperTask performs housekeeping activities, such as deleting
// stale records.
func HandleHousekeeperTask(ctx context.Context, task *asynq.Task) error {
	var payload HousekeeperPayload
	if err := asynqutils.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	logger := asynqutils.GetLogger(ctx)
	for _, item := range payload.Retention {
		// Look up the registry for the actual model type
		model, ok := registry.ModelRegistry.Get(item.Name)
		if !ok {
			logger.Warn("model not found in registry", "name", item.Name)
			continue
		}

		now := time.Now()
		past := now.Add(-item.Duration)
		out, err := db.DB.NewDelete().
			Model(model).
			Where("date_part('epoch', updated_at) < ?", past.Unix()).
			Exec(ctx)

		switch err {
		case nil:
			count, err := out.RowsAffected()
			if err != nil {
				logger.Error("failed to get number of deleted rows", "name", item.Name, "reason", err)
				continue
			}
			logger.Info("deleted stale records", "name", item.Name, "count", count)
		default:
			// Simply log the error here and keep going with the
			// rest of the objects to cleanup
			logger.Error("failed to delete stale records", "name", item.Name, "reason", err)
		}
	}

	return nil
}

// HandleDeleteArchivedTask deletes archived tasks.
func HandleDeleteArchivedTask(ctx context.Context, task *asynq.Task) error {
	data := task.Payload()
	var queue string

	if data == nil {
		return asynqutils.SkipRetry(errors.New("queue name is empty"))
	}

	var payload DeleteQueuePayload
	if err := asynqutils.Unmarshal(task.Payload(), &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	queue = payload.Queue
	if queue == "" {
		return asynqutils.SkipRetry(errors.New("queue name is empty"))
	}

	logger := asynqutils.GetLogger(ctx)

	count, err := asynqclient.Inspector.DeleteAllArchivedTasks(queue)
	if err != nil {
		logger.Error("failed to delete archived tasks", "queue", queue, "reason", err)
	}

	logger.Info("deleted archived tasks", "count", count)

	return nil
}

// HandleDeleteCompletedTask deletes completed tasks.
func HandleDeleteCompletedTask(ctx context.Context, task *asynq.Task) error {
	data := task.Payload()
	var queue string

	if data == nil {
		return asynqutils.SkipRetry(errors.New("no queue specified"))
	}

	var payload DeleteQueuePayload
	if err := asynqutils.Unmarshal(task.Payload(), &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	queue = payload.Queue
	if queue == "" {
		return asynqutils.SkipRetry(errors.New("queue name is empty"))
	}

	logger := asynqutils.GetLogger(ctx)

	count, err := asynqclient.Inspector.DeleteAllCompletedTasks(queue)
	if err != nil {
		logger.Error("failed to delete completed tasks", "queue", queue, "reason", err)
	}

	logger.Info("deleted completed tasks", "count", count)

	return nil
}

func init() {
	registry.TaskRegistry.MustRegister(HousekeeperTaskType, asynq.HandlerFunc(HandleHousekeeperTask))
	registry.TaskRegistry.MustRegister(DeleteArchivedTaskType, asynq.HandlerFunc(HandleDeleteArchivedTask))
	registry.TaskRegistry.MustRegister(DeleteCompletedTaskType, asynq.HandlerFunc(HandleDeleteCompletedTask))
}
