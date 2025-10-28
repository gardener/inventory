// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/gardener/gardener-extension-provider-openstack/pkg/apis/openstack"
	openstackinstall "github.com/gardener/gardener-extension-provider-openstack/pkg/apis/openstack/install"
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
	// TaskCollectOpenStackMachineImages is the name of the task for collecting
	// Machine Images for OpenStack Cloud Profile type.
	TaskCollectOpenStackMachineImages = "g:task:collect-openstack-machine-images"
)

// HandleCollectOpenStackMachineImagesTask is the handler for collecting Machine
// Images for OpenStack Cloud Profile type.
func HandleCollectOpenStackMachineImagesTask(ctx context.Context, t *asynq.Task) error {
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

	return collectOpenStackMachineImages(ctx, payload)
}

func collectOpenStackMachineImages(ctx context.Context, payload CollectCPMachineImagesPayload) error {
	images, err := getOpenStackMachineImages(payload.ProviderConfig)
	if err != nil {
		return err
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting machine images", "cloud_profile", payload.CloudProfileName)
	items := make([]models.CloudProfileOpenStackImage, 0)
	for _, image := range images {
		for _, version := range image.Versions {
			for _, region := range version.Regions {
				item := models.CloudProfileOpenStackImage{
					Name:             image.Name,
					Version:          version.Version,
					RegionName:       region.Name,
					ImageID:          region.ID,
					Architecture:     ptr.Value(region.Architecture, ""),
					CloudProfileName: payload.CloudProfileName,
				}

				items = append(items, item)
			}
		}
	}

	if len(items) == 0 {
		return nil
	}

	items = deduplicateOpenStackItemsByKey(items)

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, version, region_name, image_id, cloud_profile_name) DO UPDATE").
		Set("architecture = EXCLUDED.architecture").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener openstack cloud profile images into db",
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
		"populated gardener openstack cloud profile images",
		"cloud_profile", payload.CloudProfileName,
		"count", count,
	)

	return nil
}

func getOpenStackMachineImages(providerConfig []byte) ([]openstack.MachineImages, error) {
	conf, err := decodeOpenStackProviderConfig(providerConfig)
	if err != nil {
		return nil, err
	}

	return conf.MachineImages, nil
}

func decodeOpenStackProviderConfig(rawProviderConfig []byte) (*openstack.CloudProfileConfig, error) {
	scheme := runtime.NewScheme()
	if err := openstackinstall.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("could not reuse openstack extension scheme. %v", err)
	}

	// reusing decoding logic from Gardener OpenStack extension
	decoder := serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	providerConfig := &openstack.CloudProfileConfig{}

	if err := gutils.Decode(decoder, rawProviderConfig, providerConfig); err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func deduplicateOpenStackItemsByKey(items []models.CloudProfileOpenStackImage) []models.CloudProfileOpenStackImage {
	keyCompareFunc := func(a, b models.CloudProfileOpenStackImage) int {
		return cmp.Or(
			strings.Compare(a.CloudProfileName, b.CloudProfileName),
			strings.Compare(a.Name, b.Name),
			strings.Compare(a.Version, b.Version),
			strings.Compare(a.RegionName, b.RegionName),
			strings.Compare(a.ImageID, b.ImageID),
		)
	}

	keyCompactFunc := func(a, b models.CloudProfileOpenStackImage) bool {
		return strings.Compare(a.CloudProfileName, b.CloudProfileName) == 0 &&
			strings.Compare(a.Name, b.Name) == 0 &&
			strings.Compare(a.Version, b.Version) == 0 &&
			strings.Compare(a.RegionName, b.RegionName) == 0 &&
			strings.Compare(a.ImageID, b.ImageID) == 0
	}

	slices.SortFunc(items, keyCompareFunc)

	return slices.CompactFunc(items, keyCompactFunc)
}
