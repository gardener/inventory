// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package asynq

import (
	"github.com/hibiken/asynq"
)

// Client is the [asynq.Client] used by workers during runtime.
var Client *asynq.Client

// Inspector is the [asynq.Inspector] used by workers during runtime.
var Inspector *asynq.Inspector

// SetClient shall be invoked from cli commands to set the asynq client for the workers.
// Workers will have the ability to enqueue tasks.
func SetClient(c *asynq.Client) {
	Client = c
}

// SetInspector shall be invoked from cli commands to set the asynq inspector for the workers.
func SetInspector(i *asynq.Inspector) {
	Inspector = i
}
