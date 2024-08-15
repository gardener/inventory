// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

// Client is a wrapper for an AWS API client, which comes with additional
// metadata such as the named credentials which were used to create the client,
// and also includes information about the caller identity.
type Client[T any] struct {
	// NamedCredentials is the name of the credentials, which were used to
	// create the client.
	NamedCredentials string

	// Account is the AWS Account ID that owns or contains the calling
	// entity.
	AccountID string

	// ARN is the AWS ARN associated with the calling entity.
	ARN string

	// UserID is the unique identifier of the calling identity.
	UserID string

	// Client is the client used to make API calls to the AWS services.
	Client T
}
