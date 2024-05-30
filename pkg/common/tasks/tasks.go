package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"

	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/core/registry"
)

const (
	// DeleteStaleRecordsTaskType is the name of the task responsible for
	// cleaning up stale records from the database.
	DeleteStaleRecordsTaskType = "common:task:housekeeper"
)

// NewDeleteStaleRecordsTask creates a new task, which deletes stale records.
func NewDeleteStaleRecordsTask(items []*config.ModelRetentionConfig) (*asynq.Task, error) {
	payload, err := yaml.Marshal(items)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(DeleteStaleRecordsTaskType, payload)

	return task, nil
}

// HandleDeleteStaleRecordsTask deletes records, which have been identified as
// stale.
func HandleDeleteStaleRecordsTask(ctx context.Context, task *asynq.Task) error {
	var items []*config.ModelRetentionConfig

	if err := yaml.Unmarshal(task.Payload(), &items); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if items == nil {
		return nil
	}

	for _, item := range items {
		// Look up the registry for the actual model type
		model, ok := registry.ModelRegistry.Get(item.Name)
		if !ok {
			slog.Warn("model not found in registry", "name", item.Name)
			continue
		}

		now := time.Now()
		past := now.Add(-item.Duration)
		out, err := clients.Db.NewDelete().
			Model(model).
			Where("updated_at < ?", past).
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
		DeleteStaleRecordsTaskType,
		asynq.HandlerFunc(HandleDeleteStaleRecordsTask),
	)
}
