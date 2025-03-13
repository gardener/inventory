// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"errors"
	"os/exec"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/core/registry"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// ErrNoCommand is an error which is returned when the task for executing
// external commands was called without specifying a command as part of the
// payload.
var ErrNoCommand = errors.New("no command specified")

const (
	// CommandTaskType is the name of the task for executing external
	// commands.
	CommandTaskType = "aux:task:command"
)

// CommandPayload represents the payload of the task for executing external
// commands.
type CommandPayload struct {
	// Command specifies the path to the command to be executed
	Command string `yaml:"command" json:"command"`

	// Args specifies any optional arguments to be passed to the command.
	Args []string `yaml:"args" json:"args"`

	// Dir specifies the working directory of the command. If not specified
	// then the external command will be executed in the calling process'
	// current directory.
	Dir string `yaml:"dir" json:"dir"`
}

// HandleCommandTask executes the command specified as part of the payload.
func HandleCommandTask(ctx context.Context, task *asynq.Task) error {
	data := task.Payload()
	var payload CommandPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Command == "" {
		return asynqutils.SkipRetry(ErrNoCommand)
	}

	path, err := exec.LookPath(payload.Command)
	if err != nil {
		return asynqutils.SkipRetry(err)
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"executing command",
		"command", path,
		"args", payload.Args,
		"dir", payload.Dir,
	)

	cmd := exec.CommandContext(ctx, path, payload.Args...)
	cmd.Dir = payload.Dir

	return cmd.Run()
}

func init() {
	registry.TaskRegistry.MustRegister(CommandTaskType, asynq.HandlerFunc(HandleCommandTask))
}
