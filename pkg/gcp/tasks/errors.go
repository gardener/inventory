// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"fmt"
)

// ErrNoProjectID is an error, which is returned when a task expects a project
// id to be sent as part of the payload, but none was provided.
var ErrNoProjectID = errors.New("no project id specified")

// ErrClientNotFound is an error, which is returned when an API client was not
// found in the clientset registries.
var ErrClientNotFound = errors.New("client not found")

// ClientNotFound wraps [ErrClientNotFound] with the given project id.
func ClientNotFound(projectID string) error {
	return fmt.Errorf("%w: project %s", ErrClientNotFound, projectID)
}
