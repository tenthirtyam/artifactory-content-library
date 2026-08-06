// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package config

import (
	"fmt"
	"strings"
)

// ValidateForGenerate checks a config used by the generate command.
func (f *File) ValidateForGenerate() error {
	if f == nil {
		return fmt.Errorf("configuration is empty")
	}
	if err := f.validateLogging(); err != nil {
		return err
	}
	if err := f.validateDefaults(); err != nil {
		return err
	}
	if len(f.Libraries) == 0 {
		return fmt.Errorf("no content libraries defined in configuration file")
	}
	for i, lib := range f.Libraries {
		prefix := fmt.Sprintf("libraries[%d]", i)
		if strings.TrimSpace(lib.Name) == "" {
			return fmt.Errorf("%s: name is required", prefix)
		}
		prefix = fmt.Sprintf("libraries[%d] (%s)", i, lib.Name)
		storageType := strings.TrimSpace(lib.Type)
		if storageType == "" {
			storageType = strings.TrimSpace(f.Defaults.Type)
		}
		if storageType == "" {
			storageType = "artifactory"
		}
		if storageType != "artifactory" {
			return fmt.Errorf("%s: type %q is invalid (want artifactory)", prefix, storageType)
		}
		if lib.Artifactory != nil {
			if err := validateArtifactory(prefix+".artifactory", lib.Artifactory); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateForSubscribe checks a config used by the subscribe command.
func (f *File) ValidateForSubscribe() error {
	if f == nil {
		return fmt.Errorf("configuration is empty")
	}
	return f.validateLogging()
}

func (f *File) validateDefaults() error {
	t := strings.TrimSpace(f.Defaults.Type)
	if t == "" {
		return nil
	}
	if t != "artifactory" {
		return fmt.Errorf("defaults.type %q is invalid (want artifactory)", t)
	}
	return nil
}

func (f *File) validateLogging() error {
	format := strings.TrimSpace(f.Logging.Format)
	if format != "" && !strings.EqualFold(format, "standard") && !strings.EqualFold(format, "structured") {
		return fmt.Errorf("logging.format %q is invalid (want standard or structured)", format)
	}
	level := strings.TrimSpace(f.Logging.Level)
	if level == "" {
		return nil
	}
	switch strings.ToUpper(level) {
	case "DEBUG", "INFO", "WARNING", "WARN", "ERROR":
		return nil
	default:
		return fmt.Errorf("logging.level %q is invalid", level)
	}
}

func validateArtifactory(prefix string, a *Artifactory) error {
	if a.RateLimit < 0 {
		return fmt.Errorf("%s: rate_limit must be >= 0", prefix)
	}
	return validateAuth(prefix+".auth", a.Auth)
}

func validateAuth(prefix string, a Auth) error {
	apiKey := strings.TrimSpace(a.APIKey)
	user := strings.TrimSpace(a.Username)
	pass := strings.TrimSpace(a.Password)
	token := strings.TrimSpace(a.Token)

	basic := user != "" || pass != ""
	if basic && (user == "" || pass == "") {
		return fmt.Errorf("%s: username and password must both be set for basic auth", prefix)
	}

	var methods []string
	if apiKey != "" {
		methods = append(methods, "api_key")
	}
	if basic {
		methods = append(methods, "basic")
	}
	if token != "" {
		methods = append(methods, "token")
	}
	if len(methods) > 1 {
		return fmt.Errorf("%s: multiple authentication methods configured (%s); use only one", prefix, strings.Join(methods, ", "))
	}
	return nil
}

var secretKeys = []string{"api_key", "password", "token", "publisher_password"}

// plaintextSecretWarnings scans raw YAML (before env expansion) for embedded secrets.
func plaintextSecretWarnings(raw string) []string {
	var warnings []string
	for line := range strings.SplitSeq(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, key := range secretKeys {
			prefix := key + ":"
			if !strings.HasPrefix(trimmed, prefix) {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			value = strings.Trim(value, `"'`)
			if value == "" || strings.HasPrefix(value, "${") {
				continue
			}
			warnings = append(warnings, fmt.Sprintf("possible plaintext secret in config: %s (prefer ${ENV} references)", trimmed))
		}
	}
	return warnings
}
