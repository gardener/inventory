// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gcp

// Client is a wrapper for a GCP API client, which comes with additional
// metadata such as the named credentials which were used to create the client,
// and the Project ID with which the client is associated with.
type Client[T any] struct {
	// NamedCredentials is the name of the credentials, which were used to
	// create the API client.
	NamedCredentials string

	// ProjectID is the immutable, globally unique GCP Project ID associated
	// with the client.
	ProjectID string

	// Client is the client used to make API calls to the GCP API services.
	Client T
}
