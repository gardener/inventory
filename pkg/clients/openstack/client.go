// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

// ClientScope uniquely identifies the scope of the credentials used with an OpenStack
// client
type ClientScope struct {
	// NamedCredentials is the name of the credentials, which were used to
	// create the API client.
	NamedCredentials string

	// Project is the project associated with the client.
	Project string

	// Domain is the domain associated with the client.
	Domain string

	// Region is the region associated with the client.
	Region string
}

// Client is a wrapper for an OpenStack API client, which comes with additional
// metadata such as the named credentials which were used to create the client,
// the Project ID, Region and Domain which the client is associated with.
type Client[T any] struct {
	ClientScope

	// Client is the client used to make API calls to the OpenStack API services.
	Client T
}
