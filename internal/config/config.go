// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// File is the top-level YAML configuration.
type File struct {
	Libraries []Library `yaml:"libraries"`
	Defaults  defaults  `yaml:"defaults"`
	Logging   logging   `yaml:"logging"`
	VSphere   vSphere   `yaml:"vsphere"`
	Library   subscribe `yaml:"library"`
}

// Library describes one content library generation target.
type Library struct {
	Name        string       `yaml:"name"`
	Type        string       `yaml:"type"`
	Path        string       `yaml:"path"`
	SkipCert    *bool        `yaml:"skip_cert"`
	Artifactory *Artifactory `yaml:"artifactory"`
}

// Artifactory holds Artifactory connection settings in YAML.
type Artifactory struct {
	URL            string `yaml:"url"`
	Repo           string `yaml:"repo"`
	Auth           Auth   `yaml:"auth"`
	SSLVerify      *bool  `yaml:"ssl_verify"`
	RateLimit      int    `yaml:"rate_limit"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxRetries     int    `yaml:"max_retries"`
}

// Auth holds auth fields (exactly one method).
type Auth struct {
	APIKey   string `yaml:"api_key"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`
}

// defaults are shared library defaults.
type defaults struct {
	Type     string `yaml:"type"`
	SkipCert *bool  `yaml:"skip_cert"`
}

// logging configures log output.
type logging struct {
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
}

// vSphere holds vCenter connection settings.
type vSphere struct {
	URL       string `yaml:"url"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	SSLVerify *bool  `yaml:"ssl_verify"`
}

// subscribe holds subscribed library settings.
type subscribe struct {
	Name                     string `yaml:"name"`
	Datacenter               string `yaml:"datacenter"`
	Datastore                string `yaml:"datastore"`
	AutoSync                 bool   `yaml:"auto_sync"`
	OnDemand                 bool   `yaml:"on_demand"`
	PublisherSubscriptionURL string `yaml:"publisher_subscription_url"`
	PublisherSSLThumbprint   string `yaml:"publisher_ssl_thumbprint"`
	PublisherUsername        string `yaml:"publisher_username"`
	PublisherPassword        string `yaml:"publisher_password"`
}

var envPattern = regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

const schemaHint = `# yaml-language-server: $schema=https://raw.githubusercontent.com/tenthirtyam/artifactory-content-library/main/schema/config.schema.json`

// Load reads, expands, and strictly decodes a YAML config file.
// warnings are non-fatal plaintext-secret notices from the raw file.
func Load(path string) (*File, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("configuration file not found: %s", path)
	}
	raw := string(data)
	warnings := plaintextSecretWarnings(raw)
	expanded := expandEnv(raw)

	var cfg File
	dec := yaml.NewDecoder(strings.NewReader(expanded))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, warnings, fmt.Errorf("invalid YAML in configuration file: %w", err)
	}
	return &cfg, warnings, nil
}

func expandEnv(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := envPattern.FindStringSubmatch(m)
		name := sub[1]
		if name == "" {
			name = sub[2]
		}
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		return m
	})
}

// WriteExample writes an example config file.
// storageType is artifactory or subscribe.
// If force is false and path already exists, it returns an error.
func WriteExample(path, storageType string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	var content string
	var nextCmd string
	switch storageType {
	case "artifactory":
		content = schemaHint + `
---
libraries:
  - name: example
    type: artifactory
    path: path/to/content
    artifactory:
      url: https://packages.example.com/artifactory
      repo: example
      ssl_verify: true
      rate_limit: 10
      timeout_seconds: 30
      max_retries: 3
      auth:
        # Use Basic Authentication:
        # username: ${ARTIFACTORY_USERNAME}
        # password: ${ARTIFACTORY_PASSWORD}
        # Or Use API Key:
        # api_key: ${ARTIFACTORY_API_KEY}
        # Or Use Access Token:
        # token: ${ARTIFACTORY_TOKEN}

logging:
  format: standard
  level: info
`
		nextCmd = "generate"
	case "subscribe":
		content = schemaHint + `
---
vsphere:
  url: ${VSPHERE_URL}
  username: ${VSPHERE_USERNAME}
  password: ${VSPHERE_PASSWORD}
  ssl_verify: true

library:
  name: Artifactory Subscribed Library
  datacenter: Datacenter
  datastore: nfs
  auto_sync: true
  on_demand: false
  publisher_subscription_url: https://packages.example.com/artifactory/.../lib.json
  publisher_ssl_thumbprint: ""
  publisher_username: ${VSPHERE_PUBLISHER_USERNAME}
  publisher_password: ${VSPHERE_PUBLISHER_PASSWORD}

logging:
  format: standard
  level: info
`
		nextCmd = "subscribe"
	default:
		return fmt.Errorf("unsupported example type %q (want artifactory or subscribe)", storageType)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return err
	}
	fmt.Printf("Generated example %s configuration file: %s\n", storageTypeLabel(storageType), path)
	fmt.Println("\nEdit this file and run:")
	fmt.Printf("  artifactory-content-library %s --config %s\n", nextCmd, path)
	return nil
}

func storageTypeLabel(t string) string {
	switch t {
	case "artifactory":
		return "Artifactory"
	case "subscribe":
		return "subscribe"
	default:
		return t
	}
}

// BoolVal returns pointer value or default.
func BoolVal(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}
