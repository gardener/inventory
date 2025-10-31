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

	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws"
	awsinstall "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/install"
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
	// TaskCollectAWSMachineImages is the name of the task for collecting
	// Machine Images for AWS Cloud Profile type.
	TaskCollectAWSMachineImages = "g:task:collect-aws-machine-images"
)

// HandleCollectAWSMachineImagesTask is the handler for collecting Machine
// Images for AWS Cloud Profile type.
func HandleCollectAWSMachineImagesTask(ctx context.Context, t *asynq.Task) error {
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

	return collectAWSMachineImages(ctx, payload)
}

func collectAWSMachineImages(ctx context.Context, payload CollectCPMachineImagesPayload) error {
	images, err := getAWSMachineImages(payload.ProviderConfig)
	if err != nil {
		return err
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting machine images", "cloud_profile", payload.CloudProfileName)
	items := make([]models.CloudProfileAWSImage, 0)

	for _, image := range images {
		for _, version := range image.Versions {
			for _, region := range version.Regions {
				item := models.CloudProfileAWSImage{
					Name:             image.Name,
					Version:          version.Version,
					RegionName:       region.Name,
					AMI:              region.AMI,
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

	items = deduplicateAWSItemsByKey(items)

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, ami, version, region_name, cloud_profile_name) DO UPDATE").
		Set("architecture = EXCLUDED.architecture").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert gardener aws cloud profile images into db",
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
		"populated gardener aws cloud profile images",
		"cloud_profile", payload.CloudProfileName,
		"count", count,
	)

	return nil
}

func getAWSMachineImages(providerConfig []byte) ([]aws.MachineImages, error) {
	conf, err := decodeAWSProviderConfig(providerConfig)
	if err != nil {
		return nil, err
	}

	return conf.MachineImages, nil
}

func decodeAWSProviderConfig(rawProviderConfig []byte) (*aws.CloudProfileConfig, error) {
	scheme := runtime.NewScheme()
	if err := awsinstall.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("could not reuse aws extension scheme. %v", err)
	}

	// reusing decoding logic from Gardener aws extension
	decoder := serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	providerConfig := &aws.CloudProfileConfig{}

	if err := gutils.Decode(decoder, rawProviderConfig, providerConfig); err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func deduplicateAWSItemsByKey(items []models.CloudProfileAWSImage) []models.CloudProfileAWSImage {
	keyCompareFunc := func(a, b models.CloudProfileAWSImage) int {
		return cmp.Or(
			strings.Compare(a.CloudProfileName, b.CloudProfileName),
			strings.Compare(a.Name, b.Name),
			strings.Compare(a.Version, b.Version),
			strings.Compare(a.RegionName, b.RegionName),
			strings.Compare(a.AMI, b.AMI),
		)
	}

	keyCompactFunc := func(a, b models.CloudProfileAWSImage) bool {
		return keyCompareFunc(a, b) == 0
	}

	slices.SortFunc(items, keyCompareFunc)

	return slices.CompactFunc(items, keyCompactFunc)
}
