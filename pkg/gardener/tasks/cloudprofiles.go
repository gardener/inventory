// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
	"github.com/gardener/inventory/pkg/gardener/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectCloudProfiles is the name of the task for collecting
	// Gardener Cloud Profiles.
	TaskCollectCloudProfiles = "g:task:collect-cloud-profiles"

	// providerTypeAWS is the name of the provider for AWS Cloud Profile
	cpProviderTypeAWS = "aws"
	// providerTypeGCP is the name of the provider for GCP Cloud Profile
	cpProviderTypeGCP = "gcp"
)

// CollectCPMachineImagesPayload is the payload for collecting the Machine
// Images for a given Cloud Profile.
type CollectCPMachineImagesPayload struct {
	// ProviderConfig is the raw config, which is specific for each Cloud
	// Profile.
	ProviderConfig []byte `json:"provider_config" yaml:"provider_config"`

	// CloudProfileName is the name of the Cloud Profile.
	CloudProfileName string `json:"cloud_profile_name" yaml:"cloud_profile_name"`
}

// NewCollectCloudProfilesTask creates a new [asynq.Task] for collecting
// Gardener Cloud Profiles., without specifying a payload.
func NewCollectCloudProfilesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectCloudProfiles, nil)
}

// HandleCollectCloudProfilesTask is the handler for collecting Gardener Cloud
// Profiles. This handler will also enqueue tasks for collecting and persisting
// the machine images for each supported Cloud Profile type.
func HandleCollectCloudProfilesTask(ctx context.Context, t *asynq.Task) error {
	// After collecting the Cloud Profiles we will enqueue a separate task
	// for persisting the Machine Images for each supported Cloud Profile
	// type. The following is the mapping between the Cloud Profile type,
	// and the task name responsible for decoding and persisting the Machine
	// Images.
	providerTypeToTask := map[string]string{
		cpProviderTypeAWS: TaskCollectAWSMachineImages,
		cpProviderTypeGCP: TaskCollectGCPMachineImages,
	}

	client, err := gardenerclient.VirtualGardenClient()
	if err != nil {
		return asynqutils.SkipRetry(ErrNoVirtualGardenClientFound)
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting Gardener cloud profiles")
	cloudProfiles := make([]models.CloudProfile, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().CloudProfiles().List(ctx, opts)
		}),
	)
	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		cp, ok := obj.(*gardenerv1beta1.CloudProfile)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		providerType := cp.Spec.Type
		providerConfig := cp.Spec.ProviderConfig
		item := models.CloudProfile{
			Name: cp.Name,
			Type: providerType,
		}
		cloudProfiles = append(cloudProfiles, item)

		// Enqueue a task for persisting the Cloud Profile Machine
		// Images, only if we have any provider data.
		if providerConfig == nil {
			logger.Error(
				"no provider config data found",
				"cloud_profile", cp.Name,
				"provider_type", providerType,
			)
			return nil
		}

		miTaskName, ok := providerTypeToTask[providerType]
		if !ok {
			logger.Warn(
				"will not collect machine images for unsupported cloud profile",
				"cloud_profile", cp.Name,
				"provider_type", providerType,
			)
			return nil
		}

		payload := CollectCPMachineImagesPayload{
			CloudProfileName: cp.Name,
			ProviderConfig:   providerConfig.Raw,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for machine images",
				"cloud_profile", cp.Name,
				"provider_type", providerType,
				"reason", err,
			)
			return nil
		}

		task := asynq.NewTask(miTaskName, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"cloud_profile", cp.Name,
				"provider_type", providerType,
				"reason", err,
			)
			return nil
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"cloud_profile", cp.Name,
			"provider_type", providerType,
		)

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list Cloud Profile resources: %w", err)
	}

	if len(cloudProfiles) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&cloudProfiles).
		On("CONFLICT (name) DO UPDATE").
		Set("type = EXCLUDED.type").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener cloud profiles into db",
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated gardener cloud profiles", "count", count)

	return nil
}
