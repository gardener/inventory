// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"fmt"
)

// ErrClientNotFound is an error which is returned when an OpenStack client was not
// found in the clientset registries.
var ErrClientNotFound = errors.New("client not found")

// ClientNotFound wraps [ErrClientNotFound] with the given name.
func ClientNotFound(name string) error {
	return fmt.Errorf("%w: %s", ErrClientNotFound, name)
}
