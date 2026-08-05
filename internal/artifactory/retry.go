// SPDX-License-Identifier: MIT

package artifactory

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/tenthirtyam/artifactory-content-library/internal/logging"
)

type retryableOperation func() error

type retryLogic struct {
	maxRetries     int
	timeoutSeconds int
}

func newRetryLogic(maxRetries, timeoutSeconds int) *retryLogic {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	return &retryLogic{
		maxRetries:     maxRetries,
		timeoutSeconds: timeoutSeconds,
	}
}

func (rl *retryLogic) executeWithRetry(ctx context.Context, operation retryableOperation, operationName string) error {
	var lastErr error

	for attempt := 0; attempt <= rl.maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		lastErr = err

		if !rl.shouldRetry(err) {
			return err
		}
		if attempt == rl.maxRetries {
			break
		}

		delay := rl.calculateBackoff(attempt)
		logging.Warn(fmt.Sprintf("%s failed (attempt %d/%d): %s. Retrying in %v...",
			operationName, attempt+1, rl.maxRetries+1, err, delay))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("%s failed after %d attempts: %w", operationName, rl.maxRetries+1, lastErr)
}

func (rl *retryLogic) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Do not retry authentication / authorization failures.
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "forbidden") {
		return false
	}

	if isNetworkError(err) {
		return true
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return true
	}
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "connect") {
		return true
	}

	// Retry 5xx responses.
	if strings.Contains(errStr, "500") || strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") || strings.Contains(errStr, "504") {
		return true
	}

	// Do not retry other 4xx client errors.
	if strings.Contains(errStr, "400") || strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "409") || strings.Contains(errStr, "422") {
		return false
	}

	return true
}

func (rl *retryLogic) calculateBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt > 10 {
		attempt = 10
	}
	return min(time.Duration(1<<attempt)*time.Second, 60*time.Second)
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr)
}
