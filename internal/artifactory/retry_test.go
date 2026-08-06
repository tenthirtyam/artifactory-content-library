// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package artifactory

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	rl := newRetryLogic(3, 30)

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"401", errors.New("server response: 401 Unauthorized"), false},
		{"403", errors.New("forbidden 403"), false},
		{"404", errors.New("download failed: 404 Not Found"), false},
		{"500", errors.New("upload failed: 500 Internal Server Error"), true},
		{"timeout", errors.New("context deadline exceeded"), true},
		{"connection", errors.New("connection refused"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rl.shouldRetry(tc.err); got != tc.want {
				t.Fatalf("shouldRetry(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestNewRetryLogicDefaults(t *testing.T) {
	rl := newRetryLogic(0, 0)
	if rl.maxRetries != 3 || rl.timeoutSeconds != 30 {
		t.Fatalf("defaults: %+v", rl)
	}
}

func TestCalculateBackoff(t *testing.T) {
	rl := newRetryLogic(3, 30)
	if got := rl.calculateBackoff(-1); got != time.Second {
		t.Fatalf("neg attempt: %v", got)
	}
	if got := rl.calculateBackoff(0); got != time.Second {
		t.Fatalf("attempt 0: %v", got)
	}
	if got := rl.calculateBackoff(1); got != 2*time.Second {
		t.Fatalf("attempt 1: %v", got)
	}
	if got := rl.calculateBackoff(100); got != 60*time.Second {
		t.Fatalf("cap: %v", got)
	}
}

func TestExecuteWithRetrySuccess(t *testing.T) {
	rl := newRetryLogic(2, 30)
	var calls atomic.Int32
	err := rl.executeWithRetry(context.Background(), func() error {
		calls.Add(1)
		return nil
	}, "op")
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestExecuteWithRetryThenSuccess(t *testing.T) {
	rl := newRetryLogic(1, 30)
	var calls atomic.Int32
	err := rl.executeWithRetry(context.Background(), func() error {
		if calls.Add(1) == 1 {
			return errors.New("upload failed: 500 Internal Server Error")
		}
		return nil
	}, "op")
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestExecuteWithRetryNonRetryable(t *testing.T) {
	rl := newRetryLogic(3, 30)
	var calls atomic.Int32
	err := rl.executeWithRetry(context.Background(), func() error {
		calls.Add(1)
		return errors.New("server response: 401 Unauthorized")
	}, "op")
	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Fatalf("should not retry auth failures; calls=%d", calls.Load())
	}
}

func TestExecuteWithRetryExhausted(t *testing.T) {
	rl := newRetryLogic(1, 30)
	err := rl.executeWithRetry(context.Background(), func() error {
		return errors.New("upload failed: 503 Service Unavailable")
	}, "op")
	if err == nil || !strings.Contains(err.Error(), "failed after") {
		t.Fatalf("expected exhausted error; got %v", err)
	}
}

func TestExecuteWithRetryContextCancel(t *testing.T) {
	rl := newRetryLogic(3, 30)
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	err := rl.executeWithRetry(ctx, func() error {
		if calls.Add(1) == 1 {
			cancel()
			return errors.New("upload failed: 500 Internal Server Error")
		}
		return nil
	}, "op")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled; got %v", err)
	}
}
