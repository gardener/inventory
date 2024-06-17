// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package clients

import "github.com/hibiken/asynq"

var Client *asynq.Client

// SetClient shall be invoked from cli commands to set the asynq client for the workers.
// Workers will have the ability to enqueue tasks.
func SetClient(c *asynq.Client) {
	Client = c
}
