// SPDX-License-Identifier: MIT
package auth_test

import (
	"testing"

	"github.com/tenthirtyam/artifactory-content-library/internal/auth"
)

func TestResolveAPIKey(t *testing.T) {
	creds, err := auth.Resolve(auth.Config{
		URL: "https://example.com/artifactory", Repo: "repo", APIKey: "key",
	})
	if err != nil || creds.Method != auth.MethodAPIKey {
		t.Fatalf("%v %+v", err, creds)
	}
}

func TestResolveBasic(t *testing.T) {
	creds, err := auth.Resolve(auth.Config{
		URL: "https://example.com/artifactory", Repo: "repo", Username: "u", Password: "p",
	})
	if err != nil || creds.Method != auth.MethodBasic {
		t.Fatalf("%v %+v", err, creds)
	}
}

func TestResolveToken(t *testing.T) {
	creds, err := auth.Resolve(auth.Config{
		URL: "https://example.com/artifactory", Repo: "repo", Token: "tok",
	})
	if err != nil || creds.Method != auth.MethodToken {
		t.Fatalf("%v %+v", err, creds)
	}
}

func TestResolveErrors(t *testing.T) {
	if _, err := auth.Resolve(auth.Config{Repo: "r", APIKey: "k"}); err == nil {
		t.Fatal("missing url")
	}
	if _, err := auth.Resolve(auth.Config{URL: "u", APIKey: "k"}); err == nil {
		t.Fatal("missing repo")
	}
	if _, err := auth.Resolve(auth.Config{URL: "u", Repo: "r"}); err == nil {
		t.Fatal("missing auth")
	}
	if _, err := auth.Resolve(auth.Config{URL: "u", Repo: "r", APIKey: "k", Token: "t"}); err == nil {
		t.Fatal("multiple auth")
	}
}
