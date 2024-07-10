// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
//
// Package provider implements utilities for creating Web Identity based
// implementations of [aws.CredentialsProvider].
//
// Please refer to the following documentation for more details.
//
// https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata
// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html
// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html
// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-idp_oidc.html
// https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html

package provider

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ErrNoSTSClient is an error, which is returned when creating a new credentials
// provider without the required AWS STS client.
var ErrNoSTSClient = errors.New("no STS client specified")

// ErrNoRoleARN is an error, which is returned when creating a new credentials
// provider without specifying a IAM Role ARN to be assumed.
var ErrNoRoleARN = errors.New("no IAM Role ARN specified")

// ErrNoTokenRetriever is an error, which is returned when creating a new web
// identity credentials provider, without specifying a
// [stscreds.IdentityTokenRetriever] implementation.
var ErrNoTokenRetriever = errors.New("no token retriever specified")

// Spec provides the configuration settings for the Web Identity Credentials
// Provider.
type Spec struct {
	// Client is the API client used to make API calls to the AWS STS.
	Client *sts.Client

	// RoleARN is the IAM Role ARN to assume.
	RoleARN string

	// RoleSessionName is the name of the session, which uniquely identifies it
	RoleSessionName string

	// Duration specifies the expiry duration of the STS credentials.
	Duration time.Duration

	// TokenRetriever is the identity token retriever implementation to use.
	TokenRetriever stscreds.IdentityTokenRetriever
}

// New creates a new Web Identity implementation of [aws.CredentialsProvider]
// based on the provided spec.
func New(spec *Spec) (aws.CredentialsProvider, error) {
	if spec.Client == nil {
		return nil, ErrNoSTSClient
	}

	if spec.RoleARN == "" {
		return nil, ErrNoRoleARN
	}

	if spec.TokenRetriever == nil {
		return nil, ErrNoTokenRetriever
	}

	opts := []func(o *stscreds.WebIdentityRoleOptions){
		func(o *stscreds.WebIdentityRoleOptions) {
			o.Duration = spec.Duration
		},
		func(o *stscreds.WebIdentityRoleOptions) {
			o.RoleSessionName = spec.RoleSessionName
		},
	}

	provider := stscreds.NewWebIdentityRoleProvider(
		spec.Client,
		spec.RoleARN,
		spec.TokenRetriever,
		opts...,
	)

	return provider, nil
}
