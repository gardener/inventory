// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
	"github.com/gardener/inventory/pkg/gardener/models"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectProjects is the name of the task for collecting Gardener
	// Projects.
	TaskCollectProjects = "g:task:collect-projects"
)

// CollectProjectsPayload represents the payload used for collecting Gardener
// Projects.
type CollectProjectsPayload struct {
	// ProjectName specifies name of the Gardener Project to be collected.
	ProjectName string `json:"project_name" yaml:"project_name"`
}

// NewCollectProjectsTask creates a new [asynq.Task] for collecting Gardener
// projects, without specifying a payload.
func NewCollectProjectsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectProjects, nil)
}

// HandleCollectProjectsTask is the handler that collects Gardener projects.
func HandleCollectProjectsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload then we collect all projects.
	data := t.Payload()
	if data == nil {
		return collectAllProjects(ctx)
	}

	var payload CollectProjectsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectName == "" {
		return asynqutils.SkipRetry(ErrNoProjectName)
	}

	return collectProject(ctx, payload)
}

// collectProject collects a single Gardener Project.
func collectProject(ctx context.Context, payload CollectProjectsPayload) error {
	logger := asynqutils.GetLogger(ctx)
	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")

		return nil
	}

	client := gardenerclient.DefaultClient.GardenClient()
	logger.Info("collecting Gardener project", "project", payload.ProjectName)

	result, err := client.CoreV1beta1().Projects().Get(ctx, payload.ProjectName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	projects, members := toProjectModels([]*v1beta1.Project{result})
	if err := persistProjects(ctx, projects); err != nil {
		return err
	}

	return persistProjectMembers(ctx, members)
}

// collectAllProjects collects all projects from Gardener.
func collectAllProjects(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")

		return nil
	}

	client := gardenerclient.DefaultClient.GardenClient()
	logger.Info("collecting Gardener projects")
	items := make([]*v1beta1.Project, 0)

	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().Projects().List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err := p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		item, ok := obj.(*v1beta1.Project)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}
		items = append(items, item)

		return nil
	})

	if err != nil {
		return err
	}

	projects, members := toProjectModels(items)
	if err := persistProjects(ctx, projects); err != nil {
		return err
	}

	return persistProjectMembers(ctx, members)
}

// toProjectModels converts the given slice of [v1beta1.Project] items into
// [models.Projects] and [models.ProjectMember] slices, suitable for persisting
// into the database.
func toProjectModels(items []*v1beta1.Project) ([]models.Project, []models.ProjectMember) {
	projects := make([]models.Project, 0)
	members := make([]models.ProjectMember, 0)

	for _, p := range items {
		// Collect projects
		projectItem := models.Project{
			Name:              p.Name,
			Namespace:         ptr.StringFromPointer(p.Spec.Namespace),
			Status:            string(p.Status.Phase),
			Purpose:           ptr.StringFromPointer(p.Spec.Purpose),
			Owner:             p.Spec.Owner.Name,
			CreationTimestamp: p.CreationTimestamp.Time,
		}
		projects = append(projects, projectItem)

		// Collect project members
		for _, member := range p.Spec.Members {
			name := member.Name
			if member.Kind == rbacv1.ServiceAccountKind {
				name = fmt.Sprintf("system:serviceaccount:%s:%s", member.Namespace, member.Name)
			}
			memberItem := models.ProjectMember{
				Name:        name,
				Kind:        member.Kind,
				Role:        member.Role,
				ProjectName: p.Name,
			}
			members = append(members, memberItem)
		}
	}

	return projects, members
}

// persistProjects persists the provided projects into the database.
func persistProjects(ctx context.Context, items []models.Project) error {
	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			projectsDesc,
			prometheus.GaugeValue,
			float64(count),
		)
		metrics.DefaultCollector.AddMetric(TaskCollectProjects, metric)
	}()

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name) DO UPDATE").
		Set("namespace = EXCLUDED.namespace").
		Set("status = EXCLUDED.status").
		Set("purpose = EXCLUDED.purpose").
		Set("owner = EXCLUDED.owner").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("populated gardener projects", "count", count)

	return nil
}

// persistProjectMembers persists the given project members into the database.
func persistProjectMembers(ctx context.Context, items []models.ProjectMember) error {
	var err error

	// Group members by project and emit metrics
	memberGroups := make(map[string]int)
	for _, member := range items {
		memberGroups[member.ProjectName]++
	}
	defer func() {
		for groupName, count := range memberGroups {
			val := count
			if err != nil {
				val = 0
			}
			metric := prometheus.MustNewConstMetric(
				projectMembersDesc,
				prometheus.GaugeValue,
				float64(val),
				groupName,
			)
			key := metrics.Key(TaskCollectProjects, "members", groupName)
			metrics.DefaultCollector.AddMetric(key, metric)
		}
	}()

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, project_name) DO UPDATE").
		Set("kind = EXCLUDED.kind").
		Set("role = EXCLUDED.role").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("populated gardener project members", "count", count)

	return nil
}
