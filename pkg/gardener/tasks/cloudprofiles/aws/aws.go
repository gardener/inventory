package aws

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gardener/gardener/extensions/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/gardener/models"
	"github.com/gardener/inventory/pkg/utils/ptr"

	aws "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws"
	awsinstall "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/install"
)

func HandleProviderConfig(ctx context.Context, rawProviderConfig []byte, cloudProfile models.CloudProfile) error {
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
		slog.Error("could not insert gardener aws cloud profile images into db",
			"reason",
			err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated gardener aws cloud profile images", "count", count)

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
