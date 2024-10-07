// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package azure

// Client is a wrapper for an Azure API client, which comes with additional
// metadata such as the named credentials which were used to create the client,
// and the Azure Subscription with which the client is associated with.
type Client[T any] struct {
	// NamedCredentials is the name of the credentials, which were used to
	// create the API client.
	NamedCredentials string

	// SubscriptionID is the id of the subscription.
	SubscriptionID string

	// SubscriptionName is the display name for the subscription.
	SubscriptionName string

	// Client is the client used to make API calls to the Azure API
	// services.
	Client T
}
