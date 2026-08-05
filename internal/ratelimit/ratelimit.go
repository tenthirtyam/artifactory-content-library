// SPDX-License-Identifier: MIT

package ratelimit

import (
	"sync"
	"time"
)

// Limiter is a thread-safe sliding-window rate limiter.
type Limiter struct {
	maxCalls int
	window   time.Duration
	calls    []time.Time
	mu       sync.Mutex
}

// New creates a rate limiter allowing maxCalls per window.
func New(maxCalls int, window time.Duration) *Limiter {
	if maxCalls <= 0 {
		maxCalls = 10
	}
	if window <= 0 {
		window = time.Second
	}
	return &Limiter{maxCalls: maxCalls, window: window}
}

// Acquire blocks until a call is permitted.
func (l *Limiter) Acquire() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for {
		now := time.Now()
		cutoff := now.Add(-l.window)
		i := 0
		for i < len(l.calls) && l.calls[i].Before(cutoff) {
			i++
		}
		l.calls = l.calls[i:]

		if len(l.calls) < l.maxCalls {
			l.calls = append(l.calls, now)
			return
		}
		sleepFor := l.window - now.Sub(l.calls[0])
		if sleepFor > 0 {
			l.mu.Unlock()
			time.Sleep(sleepFor)
			l.mu.Lock()
		}
	}
}
