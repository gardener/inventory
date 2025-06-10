// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package slog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/gardener/inventory/pkg/core/config"
)

// ErrInvalidLogLevel is an error, which is returned when an invalid log level
// has been configured.
var ErrInvalidLogLevel = errors.New("invalid log level")

// ErrInvalidLogFormat is an error, which is returned when an invalid log format
// has been configured.
var ErrInvalidLogFormat = errors.New("invalid log format")

// LogLevel represents the log level.
type LogLevel string

var (
	// LevelInfo specifies INFO log level.
	LevelInfo LogLevel = "info"
	// LevelWarn specifies WARN log level.
	LevelWarn LogLevel = "warn"
	// LevelError specifies ERROR log level.
	LevelError LogLevel = "error"
	// LevelDebug specifies DEBUG log level.
	LevelDebug LogLevel = "debug"
)

// LogFormat represents the format of log events.
type LogFormat string

var (
	// FormatText specifies text log format.
	FormatText LogFormat = "text"
	// FormatJSON specifies JSON log format.
	FormatJSON LogFormat = "json"
)

// NewFromConfig creates a new [slog.Logger] based on the provided
// [config.LoggingConfig] spec. The returned logger outputs to the given
// [io.Writer].
func NewFromConfig(w io.Writer, conf config.LoggingConfig) (*slog.Logger, error) {
	// Defaults, if we don't have any logging settings
	logLevel := LevelInfo
	logFormat := FormatText

	if conf.Level != "" {
		logLevel = LogLevel(conf.Level)
	}

	if conf.Format != "" {
		logFormat = LogFormat(conf.Format)
	}

	// Supported log levels
	levels := map[LogLevel]slog.Level{
		LevelInfo:  slog.LevelInfo,
		LevelWarn:  slog.LevelWarn,
		LevelError: slog.LevelError,
		LevelDebug: slog.LevelDebug,
	}

	level, ok := levels[logLevel]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidLogLevel, logLevel)
	}

	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		AddSource: conf.AddSource,
		Level:     level,
	}

	switch logFormat {
	case FormatText:
		handler = slog.NewTextHandler(w, handlerOpts)
	case FormatJSON:
		handler = slog.NewJSONHandler(w, handlerOpts)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidLogFormat, logFormat)
	}

	// Add default attributes to the logger
	attrs := make([]slog.Attr, 0)
	for k, v := range conf.Attributes {
		attrs = append(attrs, slog.Any(k, v))
	}
	logger := slog.New(handler.WithAttrs(attrs))

	return logger, nil
}
