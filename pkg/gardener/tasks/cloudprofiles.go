// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/gardener/gardener/extensions/pkg/util"
	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/hibiken/asynq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/models"
	"github.com/gardener/inventory/pkg/utils/ptr"

	aws "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws"
	awsinstall "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/install"
)

const (
	// TaskCollectCloudProfiles is the type of the task that collects Gardener CloudProfiles.
	TaskCollectCloudProfiles = "g:task:collect-cloud-profiles"
)

// cannot use const on slices/arrays, so this is just package private
var allowedProviderTypes = []string{"aws", "alicloud", "gcp", "azure", "openstack", "ironcore"}

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
		if !slices.Contains(allowedProviderTypes, providerType) {
			return fmt.Errorf("received CloudProfile with invalid profile type: profile: %v, type: %v", cp.Name, providerType)
		}

		providerConfig := cp.Spec.ProviderConfig
		if providerConfig == nil {
			return fmt.Errorf("providerConfig not provided for CloudProfile %v", cp.Name)
		}

		cloudProfile := models.CloudProfile{
			Name: cp.GetName(),
			Type: providerType,
		}
		cloudProfiles = append(cloudProfiles, cloudProfile)

		switch providerType {
		case "aws":
			if err := handleAWSProviderConfig(ctx, providerConfig.Raw, cloudProfile); err != nil {
				return err
			}
		case "alicloud":
		case "gcp":
		case "azure":
		case "openstack":
		case "ironcore":
		default:
			return fmt.Errorf("received CloudProfile with invalid provider type: profile: %v, type: %v", cp.Name, providerType)
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

func handleAWSProviderConfig(ctx context.Context, rawProviderConfig []byte, cloudProfile models.CloudProfile) error {
	images, err := getAWSMachineImages(rawProviderConfig)
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
					CloudProfileName: cloudProfile.Name,
				}

				modelImages = append(modelImages, modelImage)
			}
		}
	}

	out, err := db.DB.NewInsert().
		Model(&modelImages).
		On("CONFLICT (name, ami, version, region_name) DO UPDATE").
		Set("architecture = EXCLUDED.architecture").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error("could not insert aws cloud profile images into db",
			"reason",
			err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws cloud profile images", "count", count)

	return nil
}

func getAWSMachineImages(rawProviderConfig []byte) ([]aws.MachineImages, error) {
	conf, err := decodeAWSProviderConfig(rawProviderConfig)
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
