// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"errors"
	"time"

	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/auxiliary/models"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// HousekeeperTaskType is the name of the task responsible for cleaning
	// up stale records from the database.
	HousekeeperTaskType = "aux:task:housekeeper"
)

// HousekeeperPayload represents the payload of the housekeeper task.
type HousekeeperPayload struct {
	// Retention provides the retention configuration of objects.
	Retention []HousekeeperRetentionConfig `yaml:"retention" json:"retention"`
}

// HousekeeperRetentionConfig represents the retention configuration for a given model.
type HousekeeperRetentionConfig struct {
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
		return asynqutils.SkipRetry(err)
	}

	// Record successful models processed by the housekeeper
	hkRuns := make([]models.HousekeeperRun, 0)

	// Capture all errors from all models during a housekeeper run.
	allErrs := make([]error, 0)

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

		allErrs = append(allErrs, err)
		completedAt := time.Now()
		switch err {
		case nil:
			count, err := out.RowsAffected()
			if err != nil {
				logger.Error("failed to get number of deleted rows", "name", item.Name, "reason", err)

				continue
			}
			logger.Info("deleted stale records", "name", item.Name, "count", count)
			hkRun := models.HousekeeperRun{
				ModelName:   item.Name,
				StartedAt:   now,
				CompletedAt: completedAt,
				Count:       count,
			}
			hkRuns = append(hkRuns, hkRun)

			metric := prometheus.MustNewConstMetric(
				hkDeletedRecordsDesc,
				prometheus.GaugeValue,
				float64(count),
				item.Name,
			)
			key := metrics.Key(HousekeeperTaskType, item.Name)
			metrics.DefaultCollector.AddMetric(key, metric)
		default:
			// Simply log the error here and keep going with the
			// rest of the objects to cleanup
			logger.Error("failed to delete stale records", "name", item.Name, "reason", err)
		}
	}

	if len(hkRuns) == 0 {
		return errors.Join(allErrs...)
	}

	_, err := db.DB.NewInsert().
		Model(&hkRuns).
		Returning("id").
		Exec(ctx)

	allErrs = append(allErrs, err)

	return errors.Join(allErrs...)
}

func init() {
	registry.TaskRegistry.MustRegister(HousekeeperTaskType, asynq.HandlerFunc(HandleHousekeeperTask))
}
