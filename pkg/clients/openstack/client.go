// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

// Client is a wrapper for an OpenStack API client, which comes with additional
// metadata such as the named credentials which were used to create the client,
// the Project ID, Region and Domain which the client is associated with.
type Client[T any] struct {
	// NamedCredentials is the name of the credentials, which were used to
	// create the API client.
	NamedCredentials string

	// ProjectID is the project id associated with the client.
	ProjectID string

	// Region is the region associated with the client.
	Region string

	// Domain is the domain associated with the client.
	Domain string
	// Client is the client used to make API calls to the OpenStack API services.
	Client T
}
