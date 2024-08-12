// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/gardener/models"
	"github.com/gardener/inventory/pkg/utils/ptr"

	"github.com/gardener/gardener/extensions/pkg/util"

	"github.com/hibiken/asynq"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	aws "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws"
	awsinstall "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/install"
)

const (
	// TaskCollectAWSMachineImages is the type of the task that collects Gardener CloudProfile machine images for AWS.
	TaskCollectAWSMachineImages = "g:task:collect-aws-machine-images"
)

// ErrMissingProviderConfig is returned when an expected provider config is missing from the payload.
var ErrMissingProviderConfig = errors.New("missing provider config in payload")

// ErrMissingCloudProfileName is returned when an expected cloud profile name is missing from the payload.
var ErrMissingCloudProfileName = errors.New("missing cloud profile name in payload")

// NewCollectAWSMachineImagesTask creates a new task for collecting Gardener CloudProfile machine images for AWS.
func NewCollectAWSMachineImagesTask(p CollectMachineImagesPayload) (*asynq.Task, error) {
	if len(p.ProviderConfig) == 0 {
		return nil, ErrMissingProviderConfig
	}

	if len(p.CloudProfileName) == 0 {
		return nil, ErrMissingCloudProfileName
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TaskCollectAWSMachineImages, payload), nil
}

// HandleCollectAWSMachineImagesTask is a handler function that collects Gardener CloudProfile AWS machine images.
func HandleCollectAWSMachineImagesTask(ctx context.Context, t *asynq.Task) error {
	var p CollectMachineImagesPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return collectAWSMachineImages(ctx, p)
}

func collectAWSMachineImages(ctx context.Context, p CollectMachineImagesPayload) error {
	slog.Info("Collecting Gardener AWS machine images")

	images, err := getAWSMachineImages(p.ProviderConfig)
	if err != nil {
		return err
	}

	modelImages := make([]models.CloudProfileAWSImage, 0)
	// denormalizing all AWSMachineImage entries
	for _, image := range images {
		for _, version := range image.Versions {
			for _, region := range version.Regions {
				modelImage := models.CloudProfileAWSImage{
					Name:             image.Name,
					Version:          version.Version,
					RegionName:       region.Name,
					AMI:              region.AMI,
					Architecture:     ptr.Value(region.Architecture, ""),
					CloudProfileName: p.CloudProfileName,
				}

				modelImages = append(modelImages, modelImage)
			}
		}
	}

	out, err := db.DB.NewInsert().
		Model(&modelImages).
		On("CONFLICT (name, ami, version, region_name, cloud_profile_name) DO UPDATE").
		Set("architecture = EXCLUDED.architecture").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error("could not insert gardener aws cloud profile images into db", "reason", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated gardener aws cloud profile images", "count", count, "cloudProfile", p.CloudProfileName)

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

	if err := util.Decode(decoder, rawProviderConfig, providerConfig); err != nil {
		return nil, err
	}

	return providerConfig, nil
}
