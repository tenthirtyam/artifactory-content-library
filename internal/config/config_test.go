// SPDX-License-Identifier: MIT
package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tenthirtyam/artifactory-content-library/internal/config"
)

func TestWriteAndLoadExample(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := config.WriteExample(path, "artifactory", false); err != nil {
		t.Fatal(err)
	}
	cfg, warnings, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings=%v", warnings)
	}
	if err := cfg.ValidateForGenerate(); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Libraries) != 1 || cfg.Libraries[0].Type != "artifactory" {
		t.Fatalf("%+v", cfg)
	}
}

func TestEnvExpansion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	_ = os.Setenv("TEST_VCSP_TOKEN", "secret-token")
	defer os.Unsetenv("TEST_VCSP_TOKEN")
	content := "libraries:\n  - name: x\n    type: artifactory\n    artifactory:\n      auth:\n        token: ${TEST_VCSP_TOKEN}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Libraries[0].Artifactory.Auth.Token != "secret-token" {
		t.Fatalf("%q", cfg.Libraries[0].Artifactory.Auth.Token)
	}
}

func TestWriteExampleRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := config.WriteExample(path, "artifactory", false); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteExample(path, "artifactory", false); err == nil {
		t.Fatal("expected error when file exists")
	}
	if err := config.WriteExample(path, "artifactory", true); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	content := "libraries:\n  - name: x\n    type: artifactory\n    path: /tmp\n    typo: true\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.Load(path); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestValidateRejectsLocalType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	content := "libraries:\n  - name: x\n    type: local\n    path: /tmp\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	err = cfg.ValidateForGenerate()
	if err == nil || !strings.Contains(err.Error(), "want artifactory") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateRejectsMultipleAuthMethods(t *testing.T) {
	cfg := &config.File{
		Libraries: []config.Library{{
			Name: "x",
			Type: "artifactory",
			Artifactory: &config.Artifactory{
				Auth: config.Auth{APIKey: "k", Token: "t"},
			},
		}},
	}
	err := cfg.ValidateForGenerate()
	if err == nil || !strings.Contains(err.Error(), "multiple authentication methods") {
		t.Fatalf("got %v", err)
	}
}

func TestPlaintextSecretWarnings(t *testing.T) {
	raw := "libraries:\n  - name: x\n    artifactory:\n      auth:\n        password: hunter2\n        token: ${ARTIFACTORY_TOKEN}\n"
	warnings := config.PlaintextSecretWarnings(raw)
	if len(warnings) != 1 {
		t.Fatalf("warnings=%v", warnings)
	}
}

func TestWriteSubscribeExample(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub.yaml")
	if err := config.WriteExample(path, "subscribe", false); err != nil {
		t.Fatal(err)
	}
	cfg, warnings, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings=%v", warnings)
	}
	if err := cfg.ValidateForSubscribe(); err != nil {
		t.Fatal(err)
	}
}
