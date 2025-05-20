// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"fmt"
)

// ErrNoPayload is an error, which is returned by task handlers, which expect a
// payload, but none was provided.
var ErrNoPayload = errors.New("no payload specified")

// ErrNoUserPrincipalName is an error, which is returned by task handlers, which
// expect a user principal name to be specified, but none was provided.
var ErrNoUserPrincipalName = errors.New("no user principal name specified")

// ErrNoTenantID is an error, which is returned when a task expects an
// Azure Tenant ID, but none was provided.
var ErrNoTenantID = errors.New("no tenant id specified")

// ErrNoSubscriptionID is an error, which is returned when a task expects an
// Azure Subscription ID, but none was provided.
var ErrNoSubscriptionID = errors.New("no subscription id specified")

// ErrNoResourceGroup is an error, which is returned when a task expects an
// Azure Resource Group name, but none was provided.
var ErrNoResourceGroup = errors.New("no resource group specified")

// ErrNoVPC is an error, which is returned when a task expects an
// Azure VPC name, but none was provided.
var ErrNoVPC = errors.New("no vpc specified")

// ErrNoStorageAccount is an error, which is returned when a task expects an
// Azure Storage Account name, but none was provided.
var ErrNoStorageAccount = errors.New("no storage account specified")

// ErrClientNotFound is an error, which is returned when an API client was not
// found in the clientset registries.
var ErrClientNotFound = errors.New("client not found")

// ClientNotFound wraps [ErrClientNotFound] with the given subscription id.
func ClientNotFound(subscriptionID string) error {
	return fmt.Errorf("%w: subscription id %s", ErrClientNotFound, subscriptionID)
}
