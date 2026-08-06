// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Configure sets up slog logging for standard or structured (JSON) output.
func Configure(format, level string) {
	lvl := parseLevel(level)
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: lvl}
	w := io.Writer(os.Stdout)
	if strings.EqualFold(format, "structured") {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func parseLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARNING", "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Audit logs an audit-style event with structured fields.
func Audit(msg string, attrs ...any) {
	slog.Info(msg, append([]any{"logger", "acl.audit"}, attrs...)...)
}

// Info is a convenience wrapper.
func Info(msg string, attrs ...any) { slog.Info(msg, attrs...) }

// Warn is a convenience wrapper.
func Warn(msg string, attrs ...any) { slog.Warn(msg, attrs...) }

// Error is a convenience wrapper.
func Error(msg string, attrs ...any) { slog.Error(msg, attrs...) }

// Debug is a convenience wrapper.
func Debug(msg string, attrs ...any) { slog.Debug(msg, attrs...) }
