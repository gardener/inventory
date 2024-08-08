// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"
	"log/slog"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/models"
)

const (
	// TaskCollectCloudProfiles is the type of the task that collects Gardener CloudProfiles.
	TaskCollectCloudProfiles = "g:task:collect-cloud-profiles"
)

const (
	providerTypeAWS = "aws"
)

// CollectMachineImagesPayload is the payload needed for g:task:collect-<provider>-machine-images.
// It is generic for all providers, since the ProviderConfig is originally presented as a []byte.
type CollectMachineImagesPayload struct {
	ProviderConfig   []byte `json:"providerConfig"`
	CloudProfileName string `json:"cloudProfileName"`
}

// NewGardenerCollectCloudProfilesTask creates a new task for collecting Gardener CloudProfiles.
func NewGardenerCollectCloudProfilesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectCloudProfiles, nil)
}

// HandleGardenerCollectCloudProfilesTask is a handler function that collects Gardener CloudProfiles.
func HandleGardenerCollectCloudProfilesTask(ctx context.Context, t *asynq.Task) error {
	return collectCloudProfiles(ctx)
}

func collectCloudProfiles(ctx context.Context) error {
	slog.Info("Collecting Gardener cloud profiles")
	gardenClient, err := gardenerclient.VirtualGardenClient()

	if err != nil {
		return fmt.Errorf("could not get garden client: %w", asynq.SkipRetry)
	}

	cloudProfiles := make([]models.CloudProfile, 0)
	err = pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return gardenClient.CoreV1beta1().CloudProfiles().List(ctx, metav1.ListOptions{})
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
		cp, ok := obj.(*gardenerv1beta1.CloudProfile)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		providerType := cp.Spec.Type

		providerConfig := cp.Spec.ProviderConfig
		if providerConfig == nil {
			slog.Error("providerConfig not provided for CloudProfile", "CloudProfile name", cp.Name, "type", providerType)
			return nil
		}

		cloudProfile := models.CloudProfile{
			Name: cp.GetName(),
			Type: providerType,
		}
		cloudProfiles = append(cloudProfiles, cloudProfile)

		payload := CollectMachineImagesPayload{
			CloudProfileName: cloudProfile.Name,
			ProviderConfig:   providerConfig.Raw,
		}

		switch providerType {
		case providerTypeAWS:
			enqueueAWSMachineImagesTask(payload)
			//TODO:
		// case "alicloud":
		// case "gcp":
		// case "azure":
		// case "openstack":
		// case "ironcore":
		default:
			slog.Error("received CloudProfile with invalid provider type", "profile", cp.Name, "type", providerType)
			return nil
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list CloudProfile resources: %w", err)
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
		slog.Error("could not insert gardener cloud profiles into db", "err", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated gardener cloud profiles", "count", count)

	return nil
}

func enqueueAWSMachineImagesTask(payload CollectMachineImagesPayload) {
	slog.Info("creating task for collecting gardener machine images for AWS")
	t, err := NewCollectAWSMachineImagesTask(payload)
	if err != nil {
		slog.Error("Could not create task for collecting AWS machine images", "err", err)
		return
	}

	slog.Info("enqueueing task for collecting gardener machine images for AWS")
	info, err := asynqclient.Client.Enqueue(t)
	if err != nil {
		slog.Error(
			"could not enqueue task",
			"type", t.Type(),
			"cloud profile", payload.CloudProfileName,
			"reason", err,
		)
		return
	}

	slog.Info(
		"enqueued task",
		"type", t.Type(),
		"id", info.ID,
		"queue", info.Queue,
		"cloud profile", payload.CloudProfileName,
	)
}
