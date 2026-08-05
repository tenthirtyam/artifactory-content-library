// SPDX-License-Identifier: MIT

package artifactory

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tenthirtyam/artifactory-content-library/internal/auth"
	"github.com/tenthirtyam/artifactory-content-library/internal/ratelimit"
)

func testHTTPClient(t *testing.T, handler http.HandlerFunc, creds *auth.Credentials) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	if creds == nil {
		creds = &auth.Credentials{
			URL:      srv.URL,
			Repo:     "repo",
			Method:   auth.MethodBasic,
			Username: "user",
			Password: "pass",
		}
	}
	c := &Client{
		baseURL:    strings.TrimRight(srv.URL, "/"),
		repo:       creds.Repo,
		httpClient: srv.Client(),
		limiter:    ratelimit.New(100, time.Second),
		creds:      creds,
		retry:      newRetryLogic(0, 5), // no retries for fast unit tests
	}
	return c, srv
}

func TestNewClientValidation(t *testing.T) {
	cases := []struct {
		name string
		cfg  *auth.Credentials
	}{
		{"empty repo", &auth.Credentials{URL: "https://example.com/artifactory", Method: auth.MethodAPIKey, APIKey: "k"}},
		{"bad url", &auth.Credentials{URL: "not-a-url", Repo: "r", Method: auth.MethodAPIKey, APIKey: "k"}},
		{"api key missing", &auth.Credentials{URL: "https://example.com/artifactory", Repo: "r", Method: auth.MethodAPIKey}},
		{"basic missing", &auth.Credentials{URL: "https://example.com/artifactory", Repo: "r", Method: auth.MethodBasic, Username: "u"}},
		{"token missing", &auth.Credentials{URL: "https://example.com/artifactory", Repo: "r", Method: auth.MethodToken}},
		{"unsupported method", &auth.Credentials{URL: "https://example.com/artifactory", Repo: "r", Method: auth.Method("other")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewClient(tc.cfg); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDownloadSuccessAndError(t *testing.T) {
	var gotAuth string
	c, _ := testHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if strings.HasSuffix(r.URL.Path, "/missing") {
			http.Error(w, "nope", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte("payload"))
	}, &auth.Credentials{
		Repo: "repo", Method: auth.MethodToken, Token: "tok",
	})

	data, err := c.Download(context.Background(), "path/file.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("got %q", data)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Fatalf("auth header %q", gotAuth)
	}

	if _, err := c.Download(context.Background(), "missing"); err == nil {
		t.Fatal("expected download error")
	}
}

func TestDeleteSuccessAndError(t *testing.T) {
	c, _ := testHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method %s", r.Method)
		}
		if strings.HasSuffix(r.URL.Path, "/bad") {
			http.Error(w, "nope", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}, nil)

	if err := c.Delete(context.Background(), "path/file.json"); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(context.Background(), "bad"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestSetProperties(t *testing.T) {
	var gotURL string
	c, _ := testHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		if r.Method != http.MethodPut {
			t.Fatalf("method %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}, nil)

	if err := c.setProperties(context.Background(), "dir/item", map[string]string{"k": "v", "a": "b"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotURL, "/api/metadata/repo/dir/item") || !strings.Contains(gotURL, "properties=") {
		t.Fatalf("unexpected url %s", gotURL)
	}
}

func TestDoAuthHeaders(t *testing.T) {
	cases := []struct {
		name  string
		creds *auth.Credentials
		check func(t *testing.T, r *http.Request)
	}{
		{
			name:  "api_key",
			creds: &auth.Credentials{Repo: "repo", Method: auth.MethodAPIKey, APIKey: "key123"},
			check: func(t *testing.T, r *http.Request) {
				if r.Header.Get("X-JFrog-Art-Api") != "key123" {
					t.Fatalf("api key header %q", r.Header.Get("X-JFrog-Art-Api"))
				}
			},
		},
		{
			name:  "basic",
			creds: &auth.Credentials{Repo: "repo", Method: auth.MethodBasic, Username: "u", Password: "p"},
			check: func(t *testing.T, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != "u" || pass != "p" {
					t.Fatalf("basic auth %q %q ok=%v", user, pass, ok)
				}
			},
		},
		{
			name:  "token",
			creds: &auth.Credentials{Repo: "repo", Method: auth.MethodToken, Token: "abc"},
			check: func(t *testing.T, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer abc" {
					t.Fatalf("auth %q", r.Header.Get("Authorization"))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := testHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
				tc.check(t, r)
				_, _ = io.WriteString(w, "ok")
			}, tc.creds)
			if _, err := c.Download(context.Background(), "f"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRepoPathAndAccessors(t *testing.T) {
	c := &Client{baseURL: "https://example.com/artifactory", repo: "my-repo"}
	if c.repo != "my-repo" || c.baseURL != "https://example.com/artifactory" {
		t.Fatalf("fields: %q %q", c.repo, c.baseURL)
	}
	if got := c.repoPath(""); got != "my-repo" {
		t.Fatalf("empty: %q", got)
	}
	if got := c.repoPath("/a/b"); got != "my-repo/a/b" {
		t.Fatalf("rel: %q", got)
	}
}
