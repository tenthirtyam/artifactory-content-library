// SPDX-License-Identifier: MIT

package auth

import (
	"fmt"
	"strconv"
	"strings"
)

// Method is an Artifactory authentication method.
type Method string

const (
	// MethodAPIKey authenticates with an Artifactory API key.
	MethodAPIKey Method = "api_key"
	// MethodBasic authenticates with username and password.
	MethodBasic Method = "basic"
	// MethodToken authenticates with an access token.
	MethodToken Method = "token"

	// DefaultRateLimit is the default Artifactory API calls allowed per second.
	DefaultRateLimit = 10
	// DefaultTimeoutSeconds is the default Artifactory HTTP timeout.
	DefaultTimeoutSeconds = 30
	// DefaultMaxRetries is the default number of Artifactory request retries.
	DefaultMaxRetries = 3
)

// Credentials holds Artifactory connection settings.
type Credentials struct {
	URL            string
	Repo           string
	Method         Method
	APIKey         string
	Username       string
	Password       string
	Token          string
	SSLVerify      bool
	RateLimit      int
	TimeoutSeconds int
	MaxRetries     int
}

// Config is the raw auth inputs before resolution.
type Config struct {
	URL            string
	Repo           string
	APIKey         string
	Username       string
	Password       string
	Token          string
	SSLVerify      *bool
	RateLimit      int
	TimeoutSeconds int
	MaxRetries     int
}

// Resolve picks exactly one auth method and validates required fields.
func Resolve(cfg Config) (*Credentials, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("missing required Artifactory URL (set --url, YAML artifactory.url, or ARTIFACTORY_URL)")
	}
	if strings.TrimSpace(cfg.Repo) == "" {
		return nil, fmt.Errorf("missing required Artifactory repository (set --repo, YAML artifactory.repo, or ARTIFACTORY_REPOSITORY)")
	}

	apiKey := strings.TrimSpace(cfg.APIKey)
	user := strings.TrimSpace(cfg.Username)
	pass := strings.TrimSpace(cfg.Password)
	token := strings.TrimSpace(cfg.Token)

	type cand struct {
		method Method
		ok     bool
	}
	cands := []cand{
		{MethodAPIKey, apiKey != ""},
		{MethodBasic, user != "" && pass != ""},
		{MethodToken, token != ""},
	}
	var available []Method
	for _, c := range cands {
		if c.ok {
			available = append(available, c.method)
		}
	}
	if len(available) == 0 {
		return nil, fmt.Errorf(
			"no authentication method configured; set one of API key, username+password, or token via flags, YAML, or environment",
		)
	}
	if len(available) > 1 {
		names := make([]string, len(available))
		for i, m := range available {
			names[i] = string(m)
		}
		return nil, fmt.Errorf("multiple authentication methods configured: %v; use only one", names)
	}

	ssl := true
	if cfg.SSLVerify != nil {
		ssl = *cfg.SSLVerify
	}
	rate := cfg.RateLimit
	if rate <= 0 {
		rate = DefaultRateLimit
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = DefaultTimeoutSeconds
	}
	retries := cfg.MaxRetries
	if retries <= 0 {
		retries = DefaultMaxRetries
	}

	creds := &Credentials{
		URL:            strings.TrimSpace(cfg.URL),
		Repo:           strings.TrimSpace(cfg.Repo),
		Method:         available[0],
		APIKey:         apiKey,
		Username:       user,
		Password:       pass,
		Token:          token,
		SSLVerify:      ssl,
		RateLimit:      rate,
		TimeoutSeconds: timeout,
		MaxRetries:     retries,
	}
	return creds, nil
}

// ParseBoolEnv parses common truthy/falsey strings.
func ParseBoolEnv(s string, defaultVal bool) bool {
	if s == "" {
		return defaultVal
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultVal
	}
}

// ParseIntEnv parses an int with default.
func ParseIntEnv(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}
