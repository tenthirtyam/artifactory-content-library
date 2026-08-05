// SPDX-License-Identifier: MIT

package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"DEBUG":   slog.LevelDebug,
		"debug":   slog.LevelDebug,
		"INFO":    slog.LevelInfo,
		"":        slog.LevelInfo,
		"WARN":    slog.LevelWarn,
		"WARNING": slog.LevelWarn,
		"ERROR":   slog.LevelError,
		"other":   slog.LevelInfo,
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Fatalf("parseLevel(%q)=%v want %v", in, got, want)
		}
	}
}

func TestConfigureStructuredAndStandard(t *testing.T) {
	// Configure should not panic for either format and should install a default logger.
	Configure("structured", "DEBUG")
	Audit("audit-event", "k", "v")
	Info("info-event")
	Warn("warn-event")
	Error("error-event")
	Debug("debug-event")

	Configure("standard", "INFO")
	Info("still-works")
}

func TestConfigureLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: parseLevel("ERROR")})
	slog.SetDefault(slog.New(handler))

	Info("should-not-appear")
	Error("should-appear")

	out := buf.String()
	if strings.Contains(out, "should-not-appear") {
		t.Fatalf("info logged at error level: %q", out)
	}
	if !strings.Contains(out, "should-appear") {
		t.Fatalf("error missing: %q", out)
	}
}
