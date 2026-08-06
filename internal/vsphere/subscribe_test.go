// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package vsphere

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"https://vc01.example.com/": "https://vc01.example.com",
		"vc01.example.com":          "vc01.example.com",
		"https://vc01.example.com":  "https://vc01.example.com",
	}
	for in, want := range cases {
		if got := normalizeURL(in); got != want {
			t.Fatalf("normalizeURL(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCreateSubscribedLibraryValidation(t *testing.T) {
	ctx := context.Background()

	_, err := CreateSubscribedLibrary(ctx, SubscribeConfig{})
	if err == nil || !strings.Contains(err.Error(), "url, username, and password") {
		t.Fatalf("expected url/username/password error; got %v", err)
	}

	_, err = CreateSubscribedLibrary(ctx, SubscribeConfig{
		URL: "vc01.example.com", Username: "u", Password: "p",
	})
	if err == nil || !strings.Contains(err.Error(), "library name, datastore, and publisher_subscription_url") {
		t.Fatalf("expected library fields error; got %v", err)
	}

	_, err = CreateSubscribedLibrary(ctx, SubscribeConfig{
		URL:                      "://bad",
		Username:                 "u",
		Password:                 "p",
		Name:                     "lib",
		Datastore:                "ds1",
		PublisherSubscriptionURL: "https://example.com/lib.json",
	})
	if err == nil {
		t.Fatal("expected invalid url error")
	}
}
