// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/gardener/gardener-extension-provider-azure/pkg/apis/azure"
	azureinstall "github.com/gardener/gardener-extension-provider-azure/pkg/apis/azure/install"
	"github.com/hibiken/asynq"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/gardener/models"
	gutils "github.com/gardener/inventory/pkg/gardener/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectAzureMachineImages is the name of the task for collecting
	// Machine Images for Azure Cloud Profile type.
	TaskCollectAzureMachineImages = "g:task:collect-azure-machine-images"
)

// HandleCollectAzureMachineImagesTask is the handler for collecting Machine
// Images for Azure Cloud Profile type.
func HandleCollectAzureMachineImagesTask(ctx context.Context, t *asynq.Task) error {
	data := t.Payload()
	if data == nil {
		return asynqutils.SkipRetry(ErrNoPayload)
	}

	var payload CollectCPMachineImagesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.CloudProfileName == "" {
		return asynqutils.SkipRetry(ErrNoCloudProfileName)
	}

	if payload.ProviderConfig == nil {
		return asynqutils.SkipRetry(ErrNoProviderConfig)
	}

	return collectAzureMachineImages(ctx, payload)
}

func collectAzureMachineImages(ctx context.Context, payload CollectCPMachineImagesPayload) error {
	images, err := getAzureMachineImages(payload.ProviderConfig)
	if err != nil {
		return err
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting machine images", "cloud_profile", payload.CloudProfileName)
	items := make([]models.CloudProfileAzureImage, 0)

	for _, image := range images {
		for _, version := range image.Versions {
			var imageID string
			imageID = ptr.Value(version.CommunityGalleryImageID, "")
			if imageID == "" {
				imageID = ptr.Value(version.SharedGalleryImageID, "")
			}

			if imageID == "" {
				imageID = ptr.Value(version.URN, "")
			}

			item := models.CloudProfileAzureImage{
				Name:             image.Name,
				Version:          version.Version,
				ImageID:          imageID,
				Architecture:     ptr.Value(version.Architecture, ""),
				CloudProfileName: payload.CloudProfileName,
			}

			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil
	}

	items = deduplicateAzureItemsByKey(items)

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, architecture, version, cloud_profile_name, image_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener azure cloud profile images into db",
			"cloud_profile", payload.CloudProfileName,
			"reason", err,
		)

		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gardener azure cloud profile images",
		"cloud_profile", payload.CloudProfileName,
		"count", count,
	)

	return nil
}

func getAzureMachineImages(providerConfig []byte) ([]azure.MachineImages, error) {
	conf, err := decodeAzureProviderConfig(providerConfig)
	if err != nil {
		return nil, err
	}

	return conf.MachineImages, nil
}

func decodeAzureProviderConfig(rawProviderConfig []byte) (*azure.CloudProfileConfig, error) {
	scheme := runtime.NewScheme()
	if err := azureinstall.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("could not reuse azure extension scheme. %v", err)
	}

	// reusing decoding logic from Gardener Azure extension
	decoder := serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	providerConfig := &azure.CloudProfileConfig{}

	if err := gutils.Decode(decoder, rawProviderConfig, providerConfig); err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func deduplicateAzureItemsByKey(items []models.CloudProfileAzureImage) []models.CloudProfileAzureImage {
	keyCompareFunc := func(a, b models.CloudProfileAzureImage) int {
		return cmp.Or(
			strings.Compare(a.CloudProfileName, b.CloudProfileName),
			strings.Compare(a.Name, b.Name),
			strings.Compare(a.Version, b.Version),
			strings.Compare(a.Architecture, b.Architecture),
			strings.Compare(a.ImageID, b.ImageID),
		)
	}

	keyCompactFunc := func(a, b models.CloudProfileAzureImage) bool {
		return keyCompareFunc(a, b) == 0
	}

	slices.SortFunc(items, keyCompareFunc)

	return slices.CompactFunc(items, keyCompactFunc)
}
