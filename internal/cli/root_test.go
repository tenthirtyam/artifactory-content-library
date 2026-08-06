// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/tenthirtyam/artifactory-content-library/internal/auth"
	"github.com/tenthirtyam/artifactory-content-library/internal/config"
	"github.com/tenthirtyam/artifactory-content-library/internal/testharness"
	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	bindEnv()
	t.Cleanup(viper.Reset)
}

func execRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetViper(t)
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestSetVersionAndFlag(t *testing.T) {
	SetVersion("1.2.3", "abc", "2026-01-01T15:04:05Z")
	out, err := execRoot(t, "--version")
	if err != nil {
		t.Fatal(err)
	}
	want := "artifactory-content-library version 1.2.3 (2026-01-01)\n" +
		"https://github.com/tenthirtyam/artifactory-content-library/releases/tag/v1.2.3\n"
	if out != want {
		t.Fatalf("version output:\n got %q\nwant %q", out, want)
	}
}

func TestVersionOutputDevHasNoReleaseURL(t *testing.T) {
	SetVersion("dev", "none", "unknown")
	out, err := execRoot(t, "--version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "https://") {
		t.Fatalf("dev version should not include release URL: %q", out)
	}
	if !strings.Contains(out, "version dev (unknown)") {
		t.Fatalf("version output: %q", out)
	}
}

func TestInitWritesConfigs(t *testing.T) {
	dir := t.TempDir()
	for _, typ := range []string{"artifactory", "subscribe"} {
		out := filepath.Join(dir, typ+".yaml")
		if _, err := execRoot(t, "init", "--output", out, "--type", typ); err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		data, err := os.ReadFile(out)
		if err != nil || len(data) == 0 {
			t.Fatalf("%s missing content: %v", typ, err)
		}
	}
}

func TestInitForceAndInvalidType(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "cfg.yaml")
	if _, err := execRoot(t, "init", "--output", out, "--type", "artifactory"); err != nil {
		t.Fatal(err)
	}
	if _, err := execRoot(t, "init", "--output", out, "--type", "artifactory"); err == nil {
		t.Fatal("expected exists error without --force")
	}
	if _, err := execRoot(t, "init", "--output", out, "--type", "artifactory", "--force"); err != nil {
		t.Fatal(err)
	}
	if _, err := execRoot(t, "init", "--output", out, "--type", "local", "--force"); err == nil {
		t.Fatal("expected invalid type")
	}
	if _, err := execRoot(t, "init", "--output", out, "--type", "nope", "--force"); err == nil {
		t.Fatal("expected invalid type")
	}
}

func TestGenerateRequiresName(t *testing.T) {
	if _, err := execRoot(t, "generate"); err == nil {
		t.Fatal("expected missing name error")
	}
}

func TestTestharnessGenerate(t *testing.T) {
	root := t.TempDir()
	item := filepath.Join(root, "debian-iso")
	if err := os.MkdirAll(item, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(item, "debian.iso"), []byte("iso"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := testharness.Generate("LocalLib", root, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, vcsp.LibFile)); err != nil {
		t.Fatalf("lib.json not written: %v", err)
	}
}

func TestResolveArtifactoryCredsPrecedence(t *testing.T) {
	resetViper(t)
	t.Setenv("ARTIFACTORY_URL", "https://env.example/artifactory")
	t.Setenv("ARTIFACTORY_REPOSITORY", "env-repo")
	t.Setenv("ARTIFACTORY_API_KEY", "env-key")
	t.Setenv("ARTIFACTORY_RATE_LIMIT", "5")
	t.Setenv("ARTIFACTORY_TIMEOUT_SECONDS", "15")
	t.Setenv("ARTIFACTORY_MAX_RETRIES", "2")

	yamlCfg := &config.Artifactory{
		URL:  "https://yaml.example/artifactory",
		Repo: "yaml-repo",
		Auth: config.Auth{APIKey: "yaml-key"},
	}
	creds, err := resolveArtifactoryCreds(yamlCfg)
	if err != nil {
		t.Fatal(err)
	}
	if creds.URL != "https://yaml.example/artifactory" || creds.Repo != "yaml-repo" || creds.APIKey != "yaml-key" {
		t.Fatalf("yaml should beat env: %+v", creds)
	}

	viper.Set("flag.artifactory.url", "https://flag.example/artifactory")
	viper.Set("flag.artifactory.repo", "flag-repo")
	viper.Set("flag.artifactory.auth.api_key", "flag-key")
	creds, err = resolveArtifactoryCreds(yamlCfg)
	if err != nil {
		t.Fatal(err)
	}
	if creds.URL != "https://flag.example/artifactory" || creds.Repo != "flag-repo" || creds.APIKey != "flag-key" {
		t.Fatalf("flags should beat yaml: %+v", creds)
	}
	if creds.Method != auth.MethodAPIKey {
		t.Fatalf("method %s", creds.Method)
	}
}

func TestResolveArtifactoryCredsEnvOnly(t *testing.T) {
	resetViper(t)
	t.Setenv("ARTIFACTORY_URL", "https://env.example/artifactory")
	t.Setenv("ARTIFACTORY_REPOSITORY", "env-repo")
	t.Setenv("ARTIFACTORY_TOKEN", "tok")
	creds, err := resolveArtifactoryCreds(nil)
	if err != nil {
		t.Fatal(err)
	}
	if creds.Method != auth.MethodToken || creds.Token != "tok" || creds.Repo != "env-repo" {
		t.Fatalf("%+v", creds)
	}
}
