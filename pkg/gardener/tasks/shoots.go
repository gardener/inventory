package tasks

import (
	"context"
	"log/slog"
	"strings"

	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/gardener/models"
	utils "github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// GARDENER_COLLECT_SHOOTS_TYPE is the type of the task that collects Gardener shoots.
	GARDENER_COLLECT_SHOOTS_TYPE = "g:task:collect-shoots"
)

// NewGardenerCollectShootsTask creates a new task for collecting Gardener shoots.
func NewGardenerCollectShootsTask() *asynq.Task {
	return asynq.NewTask(GARDENER_COLLECT_SHOOTS_TYPE, nil)
}

// HandleGardenerCollectShootsTask is a handler function that collects Gardener shoots.
func HandleGardenerCollectShootsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting Gardener shoots")
	return collectShoots(ctx)
}

func collectShoots(ctx context.Context) error {
	shootList, err := clients.VirtualGardenClient.CoreV1beta1().Shoots("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	shoots := make([]models.Shoot, 0, len(shootList.Items))
	for _, s := range shootList.Items {
		projectName, _ := strings.CutPrefix("garden-", s.Namespace)
		shoot := models.Shoot{
			Name:         s.Name,
			TechnicalId:  s.Status.TechnicalID,
			Namespace:    s.Namespace,
			ProjectName:  projectName,
			CloudProfile: s.Spec.CloudProfileName,
			Purpose:      utils.StringFromPointer((*string)(s.Spec.Purpose)),
			SeedName:     utils.StringFromPointer(s.Spec.SeedName),
			Status:       s.Labels["shoot.gardener.cloud/status"],
			IsHibernated: s.Status.IsHibernated,
			CreatedBy:    s.Annotations["garden.sapcloud.io/createdBy"],
		}
		shoots = append(shoots, shoot)
	}
	if len(shoots) == 0 {
		return nil
	}
	_, err = clients.Db.NewInsert().
		Model(&shoots).
		On("CONFLICT (technical_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("namespace = EXCLUDED.namespace").
		Set("project_name = EXCLUDED.project_name").
		Set("cloud_profile = EXCLUDED.cloud_profile").
		Set("purpose = EXCLUDED.purpose").
		Set("seed_name = EXCLUDED.seed_name").
		Set("status = EXCLUDED.status").
		Set("is_hibernated = EXCLUDED.is_hibernated").
		Set("created_by = EXCLUDED.created_by").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert gardener shoots into db", "err", err)
		return err
	}

	return nil
}
