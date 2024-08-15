// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package asynq provides various asynq utilities
package asynq

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"
)

// SkipRetry wraps the provided error with [asynq.SkipRetry] in order to signal
// asynq that the task should not retried.
func SkipRetry(err error) error {
	return fmt.Errorf("%w (%w)", err, asynq.SkipRetry)
}

// Unmarshal unmarshals the given payload data by first attempting to unmarshal
// using [json.Unmarshal], and if not successful then falls back to
// [yaml.Unmarshal].
func Unmarshal(data []byte, v any) error {
	err := json.Unmarshal(data, &v)
	if err == nil {
		return nil
	}

	return yaml.Unmarshal(data, &v)
}
