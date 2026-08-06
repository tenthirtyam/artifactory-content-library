// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package ratelimit_test

import (
	"testing"
	"time"

	"github.com/tenthirtyam/artifactory-content-library/internal/ratelimit"
)

func TestRateLimiter(t *testing.T) {
	l := ratelimit.New(2, 50*time.Millisecond)
	start := time.Now()
	l.Acquire()
	l.Acquire()
	l.Acquire()
	if time.Since(start) < 40*time.Millisecond {
		t.Fatal("expected rate limit delay")
	}
}
