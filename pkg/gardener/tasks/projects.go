package tasks

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/gardener/models"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// GARDENER_COLLECT_PROJECTS_TYPE is the type of the task that collects Gardener projects.
	GARDENER_COLLECT_PROJECTS_TYPE = "g:task:collect-projects"
)

// NewGardenerCollectProjects creates a new task for collecting Gardener projects.
func NewGardenerCollectProjects() *asynq.Task {
	return asynq.NewTask(GARDENER_COLLECT_PROJECTS_TYPE, nil)
}

// HandleGardenerCollectProjectsTask is a handler function that collects Gardener projects.
func HandleGardenerCollectProjectsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting Gardener projects")
	err := collectProjects(ctx)
	if err != nil {
		return err
	}
	return nil
}

func collectProjects(ctx context.Context) error {

	projectList, err := clients.VirtualGardenClient.CoreV1beta1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	projects := make([]models.Project, 0, len(projectList.Items))
	for _, p := range projectList.Items {
		projectModel := models.Project{
			Name:      p.Name,
			Namespace: p.Namespace,
			Status:    string(p.Status.Phase),
			Purpose:   strings.StringFromPointer(p.Spec.Purpose),
			Owner:     p.Spec.Owner.Name,
		}
		projects = append(projects, projectModel)
	}
	if len(projects) == 0 {
		return nil
	}
	_, err = clients.Db.NewInsert().
		Model(&projects).
		On("CONFLICT (name) DO UPDATE").
		Set("namespace = EXCLUDED.namespace").
		Set("status = EXCLUDED.status").
		Set("purpose = EXCLUDED.purpose").
		Set("owner = EXCLUDED.owner").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert gardener projects into db", "err", err)
		return err
	}
	return nil

}
