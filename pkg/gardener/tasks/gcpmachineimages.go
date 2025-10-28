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

	"github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp"
	gcpinstall "github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp/install"
	"github.com/hibiken/asynq"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/gardener/models"
	gutils "github.com/gardener/inventory/pkg/gardener/utils"
	"github.com/gardener/inventory/pkg/gcp/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectGCPMachineImages is the name of the task for collecting
	// Machine Images for GCP Cloud Profile type.
	TaskCollectGCPMachineImages = "g:task:collect-gcp-machine-images"
)

// HandleCollectGCPMachineImagesTask is the handler for collecting Machine
// Images for GCP Cloud Profile type.
func HandleCollectGCPMachineImagesTask(ctx context.Context, t *asynq.Task) error {
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

	return collectGCPMachineImages(ctx, payload)
}

func collectGCPMachineImages(ctx context.Context, payload CollectCPMachineImagesPayload) error {
	images, err := getGCPMachineImages(payload.ProviderConfig)
	if err != nil {
		return err
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting machine images", "cloud_profile", payload.CloudProfileName)
	items := make([]models.CloudProfileGCPImage, 0)

	for _, image := range images {
		for _, version := range image.Versions {
			item := models.CloudProfileGCPImage{
				Name:             image.Name,
				Version:          version.Version,
				Image:            utils.ResourceNameFromURL(version.Image),
				Architecture:     ptr.Value(version.Architecture, ""),
				CloudProfileName: payload.CloudProfileName,
			}

			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil
	}

	items = deduplicateGCPItemsByKey(items)

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, image, version, cloud_profile_name) DO UPDATE").
		Set("architecture = EXCLUDED.architecture").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener gcp cloud profile images into db",
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
		"populated gardener gcp cloud profile images",
		"cloud_profile", payload.CloudProfileName,
		"count", count,
	)

	return nil
}

func getGCPMachineImages(providerConfig []byte) ([]gcp.MachineImages, error) {
	conf, err := decodeGCPProviderConfig(providerConfig)
	if err != nil {
		return nil, err
	}

	return conf.MachineImages, nil
}

func decodeGCPProviderConfig(rawProviderConfig []byte) (*gcp.CloudProfileConfig, error) {
	scheme := runtime.NewScheme()
	if err := gcpinstall.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("could not reuse gcp extension scheme. %v", err)
	}

	// reusing decoding logic from Gardener GCP extension
	decoder := serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	providerConfig := &gcp.CloudProfileConfig{}

	if err := gutils.Decode(decoder, rawProviderConfig, providerConfig); err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func deduplicateGCPItemsByKey(items []models.CloudProfileGCPImage) []models.CloudProfileGCPImage {
	keyCompareFunc := func(a, b models.CloudProfileGCPImage) int {
		return cmp.Or(
			strings.Compare(a.CloudProfileName, b.CloudProfileName),
			strings.Compare(a.Name, b.Name),
			strings.Compare(a.Version, b.Version),
			strings.Compare(a.Image, b.Image),
		)
	}

	keyCompactFunc := func(a, b models.CloudProfileGCPImage) bool {
		return strings.Compare(a.CloudProfileName, b.CloudProfileName) == 0 &&
			strings.Compare(a.Name, b.Name) == 0 &&
			strings.Compare(a.Version, b.Version) == 0 &&
			strings.Compare(a.Image, b.Image) == 0
	}

	slices.SortFunc(items, keyCompareFunc)

	return slices.CompactFunc(items, keyCompactFunc)
}
