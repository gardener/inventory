// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry

import "github.com/hibiken/asynq"

// TaskRegistry is the default registry for tasks.
var TaskRegistry = New[string, asynq.Handler]()

// ScheduledTaskRegistry is the default registry for scheduled tasks.
var ScheduledTaskRegistry = New[string, *asynq.Task]()
