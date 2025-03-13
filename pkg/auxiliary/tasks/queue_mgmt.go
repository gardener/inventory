// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"errors"

	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/core/registry"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// DeleteArchivedTaskType is the name of the task responsible for deleting
	// archived tasks from a task queue
	DeleteArchivedTaskType = "aux:task:delete-archived-tasks"
	// DeleteCompletedTaskType is the name of the task responsible for deleting
	// completed tasks from a task queue
	DeleteCompletedTaskType = "aux:task:delete-completed-tasks"
)

// DeleteQueuePayload represents the payload of a task management task.
type DeleteQueuePayload struct {
	// Name of the queue that holds the tasks.
	Queue string `yaml:"queue" json:"queue"`
}

// HandleDeleteArchivedTask deletes archived tasks.
func HandleDeleteArchivedTask(ctx context.Context, task *asynq.Task) error {
	var payload DeleteQueuePayload
	if err := asynqutils.Unmarshal(task.Payload(), &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Queue == "" {
		return asynqutils.SkipRetry(errors.New("queue name is empty"))
	}

	logger := asynqutils.GetLogger(ctx)

	count, err := asynqclient.Inspector.DeleteAllArchivedTasks(payload.Queue)
	if err != nil {
		return err
	}

	logger.Info("deleted archived tasks", "count", count)

	return nil
}

// HandleDeleteCompletedTask deletes completed tasks.
func HandleDeleteCompletedTask(ctx context.Context, task *asynq.Task) error {
	var payload DeleteQueuePayload
	if err := asynqutils.Unmarshal(task.Payload(), &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Queue == "" {
		return asynqutils.SkipRetry(errors.New("queue name is empty"))
	}

	logger := asynqutils.GetLogger(ctx)

	count, err := asynqclient.Inspector.DeleteAllCompletedTasks(payload.Queue)
	if err != nil {
		return err
	}

	logger.Info("deleted completed tasks", "count", count)

	return nil
}

func init() {
	registry.TaskRegistry.MustRegister(DeleteArchivedTaskType, asynq.HandlerFunc(HandleDeleteArchivedTask))
	registry.TaskRegistry.MustRegister(DeleteCompletedTaskType, asynq.HandlerFunc(HandleDeleteCompletedTask))
}
