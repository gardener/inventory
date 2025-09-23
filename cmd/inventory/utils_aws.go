// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/gardener/inventory/pkg/aws/stscreds/kubesatoken"
	"github.com/gardener/inventory/pkg/aws/stscreds/provider"
	"github.com/gardener/inventory/pkg/aws/stscreds/tokenfile"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// errNoAWSRegion is an error which is returned when there was no region or
// default region configured for the AWS client.
var errNoAWSRegion = errors.New("no AWS region specified")

// errNoAWSTokenRetriever is an error, which is returned when there was no token
// retriever name specified.
var errNoAWSTokenRetriever = errors.New("no AWS token retriever specified")

// errUnknownAWSTokenRetriever is an error, which is returned when using an
// unknown/unsupported identity token retriever.
var errUnknownAWSTokenRetriever = errors.New("unknown AWS token retriever specified")

// validateAWSConfig validates the AWS configuration settings.
func validateAWSConfig(conf *config.Config) error {
	// Region or default region must be specified
	if conf.AWS.Region == "" && conf.AWS.DefaultRegion == "" {
		return errNoAWSRegion
	}

	// Make sure that services have configured named credentials
	services := map[string][]string{
		"ec2":     conf.AWS.Services.EC2.UseCredentials,
		"elb":     conf.AWS.Services.ELB.UseCredentials,
		"elbv2":   conf.AWS.Services.ELBv2.UseCredentials,
		"s3":      conf.AWS.Services.S3.UseCredentials,
		"route53": conf.AWS.Services.Route53.UseCredentials,
	}

	for service, namedCredentials := range services {
		// We expect at least one named credential to be present per
		// service
		if len(namedCredentials) == 0 {
			return fmt.Errorf("aws: %w: %s", errNoServiceCredentials, service)
		}

		// Validate that the named credentials used by the services are
		// actually configured.
		for _, nc := range namedCredentials {
			if _, ok := conf.AWS.Credentials[nc]; !ok {
				return fmt.Errorf("aws: %w: service %s refers %s", errUnknownNamedCredentials, service, nc)
			}
		}
	}

	// Each named credential must use a valid token retriever
	supportedTokenRetrievers := []string{
		config.DefaultAWSTokenRetriever,
		kubesatoken.TokenRetrieverName,
		tokenfile.TokenRetrieverName,
	}
	for name, creds := range conf.AWS.Credentials {
		if creds.TokenRetriever == "" {
			return fmt.Errorf("%w: %s", errNoAWSTokenRetriever, name)
		}

		if !slices.Contains(supportedTokenRetrievers, creds.TokenRetriever) {
			return fmt.Errorf("%w: %s", errUnknownAWSTokenRetriever, creds.TokenRetriever)
		}
	}

	return nil
}

// newAWSSTSClient creates a new [sts.Client] based on the provided
// [config.Config] spec.
func newAWSSTSClient(conf *config.Config) *sts.Client {
	awsConf := aws.Config{
		Region: conf.AWS.Region,
		AppID:  conf.AWS.AppID,
	}
	client := sts.NewFromConfig(awsConf)

	return client
}

// newKubeSATokenCredentialsProvider creates a new [aws.CredentialsProvider],
// which uses Kubernetes Service Account Tokens for exchanging them with
// temporary security credentials when accessing AWS resources.  The token
// retriever is configured based on the provided [config.AWSCredentialsConfig].
func newKubeSATokenCredentialsProvider(conf *config.Config, creds config.AWSCredentialsConfig) (aws.CredentialsProvider, error) {
	tokenRetriever, err := kubesatoken.NewTokenRetriever(
		kubesatoken.WithKubeconfig(creds.KubeSATokenRetriever.Kubeconfig),
		kubesatoken.WithServiceAccount(creds.KubeSATokenRetriever.ServiceAccount),
		kubesatoken.WithNamespace(creds.KubeSATokenRetriever.Namespace),
		kubesatoken.WithAudiences(creds.KubeSATokenRetriever.Audiences),
		kubesatoken.WithTokenExpiration(creds.KubeSATokenRetriever.Duration),
	)

	if err != nil {
		return nil, err
	}

	providerSpec := &provider.Spec{
		Client:          newAWSSTSClient(conf),
		RoleARN:         creds.KubeSATokenRetriever.RoleARN,
		RoleSessionName: creds.KubeSATokenRetriever.RoleSessionName,
		Duration:        creds.KubeSATokenRetriever.Duration,
		TokenRetriever:  tokenRetriever,
	}

	return provider.New(providerSpec)
}

// newTokenFileCredentialsProvider creates a new [aws.CredentialsProvider],
// which reads a JWT token from a specified path and exchanges the token for
// temporary security credentials when accessing AWS resources.  The token
// retriever is configured based on the provided [config.AWSCredentialsConfig].
func newTokenFileCredentialsProvider(conf *config.Config, creds config.AWSCredentialsConfig) (aws.CredentialsProvider, error) {
	tokenRetriever, err := tokenfile.NewTokenRetriever(
		tokenfile.WithPath(creds.TokenFileRetriever.Path),
	)

	if err != nil {
		return nil, err
	}

	providerSpec := &provider.Spec{
		Client:          newAWSSTSClient(conf),
		RoleARN:         creds.TokenFileRetriever.RoleARN,
		RoleSessionName: creds.TokenFileRetriever.RoleSessionName,
		Duration:        creds.TokenFileRetriever.Duration,
		TokenRetriever:  tokenRetriever,
	}

	return provider.New(providerSpec)
}

// loadAWSConfig loads the AWS configurations for the given named credentials.
func loadAWSConfig(ctx context.Context, conf *config.Config, namedCredentials string) (aws.Config, error) {
	creds, ok := conf.AWS.Credentials[namedCredentials]
	if !ok {
		return aws.Config{}, fmt.Errorf("%w: %s", errUnknownNamedCredentials, namedCredentials)
	}

	// Default set of options
	opts := []func(o *awsconfig.LoadOptions) error{
		awsconfig.WithRegion(conf.AWS.Region),
		awsconfig.WithDefaultRegion(conf.AWS.DefaultRegion),
		awsconfig.WithAppID(conf.AWS.AppID),
	}

	switch creds.TokenRetriever {
	case config.DefaultAWSTokenRetriever:
		// Load shared credentials config only
		break // nolint: revive
	case kubesatoken.TokenRetrieverName:
		credsProvider, err := newKubeSATokenCredentialsProvider(conf, creds)
		if err != nil {
			return aws.Config{}, err
		}
		opts = append(opts, awsconfig.WithCredentialsProvider(credsProvider))
	case tokenfile.TokenRetrieverName:
		credsProvider, err := newTokenFileCredentialsProvider(conf, creds)
		if err != nil {
			return aws.Config{}, err
		}
		opts = append(opts, awsconfig.WithCredentialsProvider(credsProvider))
	default:
		return aws.Config{}, errUnknownAWSTokenRetriever
	}

	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

// configureEC2Clientset configures the [awsclients.EC2Clientset] registry.
func configureEC2Clientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.EC2.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := ec2.NewFromConfig(awsConf)
		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*ec2.Client]{
			NamedCredentials: namedCreds,
			AccountID:        ptr.StringFromPointer(callerIdentity.Account),
			ARN:              ptr.StringFromPointer(callerIdentity.Arn),
			UserID:           ptr.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.EC2Clientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "ec2",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

// configureELBClientset configures the [awsclients.ELBClientset] registry.
func configureELBClientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.ELB.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := elb.NewFromConfig(awsConf)
		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*elb.Client]{
			NamedCredentials: namedCreds,
			AccountID:        ptr.StringFromPointer(callerIdentity.Account),
			ARN:              ptr.StringFromPointer(callerIdentity.Arn),
			UserID:           ptr.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.ELBClientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "elb",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

// configureELBv2Clientset configures the [awsclients.ELBv2Clientset] registry.
func configureELBv2Clientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.ELBv2.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := elbv2.NewFromConfig(awsConf)
		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*elbv2.Client]{
			NamedCredentials: namedCreds,
			AccountID:        ptr.StringFromPointer(callerIdentity.Account),
			ARN:              ptr.StringFromPointer(callerIdentity.Arn),
			UserID:           ptr.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.ELBv2Clientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "elbv2",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

// configureS3Clientset configures the [awsclients.S3Clientset] registry.
func configureS3Clientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.S3.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := s3.NewFromConfig(awsConf)
		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*s3.Client]{
			NamedCredentials: namedCreds,
			AccountID:        ptr.StringFromPointer(callerIdentity.Account),
			ARN:              ptr.StringFromPointer(callerIdentity.Arn),
			UserID:           ptr.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.S3Clientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "s3",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

type delayer struct{}

func (delayer) BackoffDelay(attempt int, err error) (time.Duration, error) {
	return time.Second, nil
}

// configureRoute53Clientset configures the [awsclients.Route53Clientset] registry.
func configureRoute53Clientset(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.AWS.Services.Route53.UseCredentials {
		awsConf, err := loadAWSConfig(ctx, conf, namedCreds)
		if err != nil {
			return err
		}

		var d delayer

		// configure a custom retryer per client instance, so they don't share
		// the same bucket
		retryer := retry.NewStandard(func(o *retry.StandardOptions) {
			o.MaxAttempts = 5
			o.Backoff = d
		})

		// Get the caller identity information associated with the named
		// credentials which were used to create the client and register
		// it.
		awsClient := route53.NewFromConfig(awsConf, func(o *route53.Options) {
			o.Retryer = retryer
		})

		stsClient := sts.NewFromConfig(awsConf)
		callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		client := &awsclients.Client[*route53.Client]{
			NamedCredentials: namedCreds,
			AccountID:        ptr.StringFromPointer(callerIdentity.Account),
			ARN:              ptr.StringFromPointer(callerIdentity.Arn),
			UserID:           ptr.StringFromPointer(callerIdentity.UserId),
			Client:           awsClient,
		}
		awsclients.Route53Clientset.Overwrite(client.AccountID, client)
		slog.Info(
			"configured AWS client",
			"service", "route53",
			"credentials", client.NamedCredentials,
			"account_id", client.AccountID,
			"arn", client.ARN,
			"user_id", client.UserID,
		)
	}

	return nil
}

// configureAWSClients creates the AWS clients for the supported by Inventory
// AWS services and registers them.
func configureAWSClients(ctx context.Context, conf *config.Config) error {
	if !conf.AWS.IsEnabled {
		slog.Warn("AWS is not enabled, will not create API clients")

		return nil
	}

	slog.Info("configuring AWS clients")
	if err := validateAWSConfig(conf); err != nil {
		return err
	}

	configFuncs := map[string]func(ctx context.Context, conf *config.Config) error{
		"ec2":     configureEC2Clientset,
		"elb":     configureELBClientset,
		"elbv2":   configureELBv2Clientset,
		"s3":      configureS3Clientset,
		"route53": configureRoute53Clientset,
	}

	for svc, configFunc := range configFuncs {
		if err := configFunc(ctx, conf); err != nil {
			return fmt.Errorf("unable to configure AWS clients for %s: %w", svc, err)
		}
	}

	return nil
}
