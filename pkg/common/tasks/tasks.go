package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"

	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/core/registry"
)

const (
	// HousekeeperTaskType is the name of the task responsible for cleaning
	// up stale records from the database.
	HousekeeperTaskType = "common:task:housekeeper"
)

// HousekeeperPayload represents the payload of the housekeeper task.
type HousekeeperPayload struct {
	// Retention provides the retention configuration of objects.
	Retention []RetentionConfig `yaml:"retention"`
}

// RetentionConfig represents the retention configuration for a given model.
type RetentionConfig struct {
	// Name specifies the model name.
	Name string `yaml:"name"`

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
	Duration time.Duration `yaml:"duration"`
}

// HandleHousekeeperTask performs housekeeping activities, such as deleting
// stale records.
func HandleHousekeeperTask(ctx context.Context, task *asynq.Task) error {
	var payload HousekeeperPayload
	if err := yaml.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, item := range payload.Retention {
		// Look up the registry for the actual model type
		model, ok := registry.ModelRegistry.Get(item.Name)
		if !ok {
			slog.Warn("model not found in registry", "name", item.Name)
			continue
		}

		now := time.Now()
		past := now.Add(-item.Duration)
		out, err := clients.DB.NewDelete().
			Model(model).
			Where("date_part('epoch', updated_at) < ?", past.Unix()).
			Exec(ctx)

		switch err {
		case nil:
			count, err := out.RowsAffected()
			if err != nil {
				slog.Error("failed to get number of deleted rows", "name", item.Name, "reason", err)
				continue
			}
			slog.Info("deleted stale records", "name", item.Name, "count", count)
		default:
			// Simply log the error here and keep going with the
			// rest of the objects to cleanup
			slog.Error("failed to delete stale records", "name", item.Name, "reason", err)
		}
	}

	return nil
}

func init() {
	registry.TaskRegistry.MustRegister(
		HousekeeperTaskType,
		asynq.HandlerFunc(HandleHousekeeperTask),
	)
}
