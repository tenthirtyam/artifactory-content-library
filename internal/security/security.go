// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package security

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxPathLength        = 4096
	maxLibraryNameLength = 255
)

var blockedPathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.\.`),
	regexp.MustCompile(`[\x00-\x1f]`),
	regexp.MustCompile(`[<>:"|?*]`),
	regexp.MustCompile(`^\s+|\s+$`),
}

var dangerousNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)data:`),
	regexp.MustCompile(`(?i)vbscript:`),
	regexp.MustCompile(`(?i)on\w+\s*=`),
}

// validationError represents a security validation failure.
type validationError struct {
	message string
}

// Error implements the error interface.
func (e *validationError) Error() string {
	return fmt.Sprintf("Security validation failed: %s", e.message)
}

// NewError returns a security validation error with the given message.
func NewError(msg string) error {
	return &validationError{message: msg}
}

// ValidatePath validates and normalizes a filesystem path.
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if len(path) > maxPathLength {
		return "", NewError(fmt.Sprintf("Path exceeds maximum length of %d characters", maxPathLength))
	}
	for _, pattern := range blockedPathPatterns {
		if pattern.MatchString(path) {
			return "", NewError(fmt.Sprintf("Path contains forbidden pattern: %s", pattern.String()))
		}
	}
	normalized, err := filepath.Abs(path)
	if err != nil {
		return "", NewError(fmt.Sprintf("Invalid path: %v", err))
	}
	if strings.Contains(normalized, "..") {
		return "", NewError(fmt.Sprintf("Path traversal attempt detected: %s", normalized))
	}
	return normalized, nil
}

// SanitizeLibraryName sanitizes and validates a library name.
func SanitizeLibraryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", NewError("Library name cannot be empty")
	}
	if len(name) > maxLibraryNameLength {
		return "", NewError(fmt.Sprintf("Library name exceeds maximum length of %d characters", maxLibraryNameLength))
	}
	// Skip ".." pattern for names (index 0); check the rest.
	for _, pattern := range blockedPathPatterns[1:] {
		if pattern.MatchString(name) {
			return "", NewError(fmt.Sprintf("Library name contains forbidden characters: %s", pattern.String()))
		}
	}
	for _, pattern := range dangerousNamePatterns {
		if pattern.MatchString(name) {
			return "", NewError(fmt.Sprintf("Library name contains potentially dangerous content: %s", pattern.String()))
		}
	}
	return name, nil
}

// ValidateArtifactoryURL validates and normalizes an Artifactory URL.
func ValidateArtifactoryURL(raw string) (string, error) {
	if raw == "" {
		return "", NewError("Artifactory URL cannot be empty")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", NewError(fmt.Sprintf("Invalid URL format: %v", err))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", NewError(fmt.Sprintf("Unsupported URL scheme: %s", parsed.Scheme))
	}
	if parsed.Hostname() == "" {
		return "", NewError("URL must include hostname")
	}
	clean := fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path)
	return strings.TrimRight(clean, "/"), nil
}

// MaskSensitive masks sensitive data for logging.
func MaskSensitive(data string, visibleChars int) string {
	if visibleChars <= 0 {
		visibleChars = 4
	}
	maskChar := "*"
	if data == "" || len(data) <= visibleChars {
		n := max(len(data), 8)
		return strings.Repeat(maskChar, n)
	}
	if len(data) <= visibleChars*2 {
		return data[:2] + strings.Repeat(maskChar, len(data)-2)
	}
	return data[:visibleChars] + strings.Repeat(maskChar, len(data)-visibleChars*2) + data[len(data)-visibleChars:]
}
