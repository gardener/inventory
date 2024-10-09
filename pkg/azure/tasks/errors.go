// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"fmt"
)

// ErrNoSubscriptionID is an error, which is returned when a task expects an
// Azure Subscription ID, but none was provided.
var ErrNoSubscriptionID = errors.New("no subscription id specified")

// ErrClientNotFound is an error, which is returned when an API client was not
// found in the clientset registries.
var ErrClientNotFound = errors.New("client not found")

// ClientNotFound wraps [ErrClientNotFound] with the given subscription id.
func ClientNotFound(subscriptionID string) error {
	return fmt.Errorf("%w: subscription id %s", ErrClientNotFound, subscriptionID)
}
