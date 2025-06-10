// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
	"github.com/gardener/inventory/pkg/gardener/models"
	gutils "github.com/gardener/inventory/pkg/gardener/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	stringutils "github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// TaskCollectShoots is the name of the task for collecting Shoots.
	TaskCollectShoots = "g:task:collect-shoots"

	// shootProjectPrefix is the prefix for the shoot project namespace
	shootProjectPrefix = "garden-"
)

// CollectShootsPayload represents the payload, which is used for collecting
// Gardener Shoot clusters.
type CollectShootsPayload struct {
	// ProjectName specifies the name of the project from which to collect
	// shoots.
	ProjectName string `yaml:"project_name" json:"project_name"`

	// ProjectNamespace represents the namespace associated with the
	// project.
	//
	// When creating a new project via the Gardener Dashboard a namespace
	// will be chosen automatically for the user, which follows the
	// `garden-<project_name>' convention.
	//
	// However, if a Project is created via the API and no .spec.namespace
	// is set for the project then the Gardener API will determine a
	// namespace for the user. See the link below for more details.
	//
	// https://github.com/gardener/gardener/blob/2c445773fc3f34681e2b755f5c2c74fbee86933c/pkg/controllermanager/controller/project/project/reconciler.go#L187-L199
	//
	// In order to collect all shoots via the cluster-scoped API an empty
	// project namespace may be used.
	ProjectNamespace string `yaml:"project_namespace" json:"project_namespace"`
}

func getCloudProfileName(s v1beta1.Shoot) (string, error) {
	if s.Spec.CloudProfile != nil {
		return s.Spec.CloudProfile.Name, nil
	}

	if s.Spec.CloudProfileName != nil {
		return *s.Spec.CloudProfileName, nil
	}

	return "", fmt.Errorf("no cloud profile name found for shoot %s", s.Name)
}

// NewCollectShootsTask creates a new [asynq.Task] for collecting
// Gardener shoots, without specifying a payload.
func NewCollectShootsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectShoots, nil)
}

// HandleCollectShootsTask is a handler that collects Gardener Shoots.
func HandleCollectShootsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting shoots from all known projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectShoots(ctx)
	}

	var payload CollectShootsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectName == "" {
		return asynqutils.SkipRetry(ErrNoProjectName)
	}

	return collectShoots(ctx, payload)
}

// enqueueCollectShoots enqueues tasks for collecting Gardener shoots from all
// locally known projects.
func enqueueCollectShoots(ctx context.Context) error {
	projects, err := gutils.GetProjectsFromDB(ctx)
	if err != nil {
		return err
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Create a task for each known project
	for _, p := range projects {
		payload := CollectShootsPayload{
			ProjectName:      p.Name,
			ProjectNamespace: p.Namespace,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Gardener Shoots",
				"project", p.Name,
				"namespace", p.Namespace,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectShoots, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project", p.Name,
				"namespace", p.Namespace,
				"reason", err,
			)

			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"project", p.Name,
			"namespace", p.Namespace,
		)
	}

	return nil
}

// collectShoots collects Gardener shoot clusters from the project specified in
// the payload.
func collectShoots(ctx context.Context, payload CollectShootsPayload) error {
	logger := asynqutils.GetLogger(ctx)
	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")

		return nil
	}

	var count int64
	defer func() {
		shootsMetric.WithLabelValues(payload.ProjectName).Set(float64(count))
	}()

	client := gardenerclient.DefaultClient.GardenClient()
	logger.Info(
		"collecting Gardener shoots",
		"project", payload.ProjectName,
		"namespace", payload.ProjectNamespace,
	)

	shoots := make([]models.Shoot, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().Shoots(payload.ProjectNamespace).List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err := p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		s, ok := obj.(*v1beta1.Shoot)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		projectName, _ := strings.CutPrefix(s.Namespace, shootProjectPrefix)
		// Skip shoots which don't have a technical id yet.
		if s.Status.TechnicalID == "" {
			logger.Warn(
				"skipping shoot",
				"name", s.Name,
				"project", projectName,
				"reason", "missing technical id",
			)

			return nil
		}

		cloudProfileName, err := getCloudProfileName(*s)
		if err != nil {
			logger.Error(
				"cannot extract shoot",
				"reason", err,
			)

			return err
		}

		workerGroups := make([]string, 0)
		workerPrefixes := make([]string, 0)
		for _, group := range s.Spec.Provider.Workers {
			workerGroups = append(workerGroups, group.Name)
			workerPrefixes = append(workerPrefixes, fmt.Sprintf("%s-%s", s.Status.TechnicalID, group.Name))
		}
		item := models.Shoot{
			Name:              s.Name,
			TechnicalID:       s.Status.TechnicalID,
			Namespace:         s.Namespace,
			ProjectName:       projectName,
			CloudProfile:      cloudProfileName,
			Purpose:           stringutils.StringFromPointer((*string)(s.Spec.Purpose)),
			SeedName:          stringutils.StringFromPointer(s.Spec.SeedName),
			Status:            s.Labels["shoot.gardener.cloud/status"],
			IsHibernated:      s.Status.IsHibernated,
			CreatedBy:         s.Annotations["gardener.cloud/created-by"],
			Region:            s.Spec.Region,
			KubernetesVersion: s.Spec.Kubernetes.Version,
			CreationTimestamp: s.CreationTimestamp.Time,
			WorkerGroups:      workerGroups,
			WorkerPrefixes:    workerPrefixes,
		}
		shoots = append(shoots, item)

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list shoots: %w", err)
	}

	if len(shoots) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
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
		Set("region = EXCLUDED.region").
		Set("k8s_version = EXCLUDED.k8s_version").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("worker_groups = EXCLUDED.worker_groups").
		Set("worker_prefixes = EXCLUDED.worker_prefixes").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener shoots into db",
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gardener shoots",
		"count", count,
		"project_name", payload.ProjectName,
		"project_namespace", payload.ProjectNamespace,
	)

	return nil
}
