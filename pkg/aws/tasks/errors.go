// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"fmt"

	"github.com/hibiken/asynq"
)

// ErrNoRegion is an error, which is returned when an expected region name is
// missing.
var ErrNoRegion = errors.New("no region name specified")

// ErrNoAccountID is an error which is returned when an AWS task was called
// without having the required Account ID in the payload.
var ErrNoAccountID = errors.New("no account id specified")

// ErrClientNotFound is an error which is returned when an AWS client was not
// found in the clientset registries.
var ErrClientNotFound = errors.New("client not found")

// SkipRetry wraps the provided error with [asynq.SkipRetry] in order to signal
// asynq that the task should not retried.
func SkipRetry(err error) error {
	return fmt.Errorf("%w (%w)", err, asynq.SkipRetry)
}

// ClientNotFound wraps [ErrClientNotFound] with the given name.
func ClientNotFound(name string) error {
	return fmt.Errorf("%w: %s", ErrClientNotFound, name)
}
