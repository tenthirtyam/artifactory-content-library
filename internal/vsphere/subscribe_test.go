// SPDX-License-Identifier: MIT

package vsphere_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tenthirtyam/artifactory-content-library/internal/vsphere"
)

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"https://vc01.example.com/": "https://vc01.example.com",
		"vc01.example.com":          "vc01.example.com",
		"https://vc01.example.com":  "https://vc01.example.com",
	}
	for in, want := range cases {
		if got := vsphere.NormalizeURL(in); got != want {
			t.Fatalf("NormalizeURL(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCreateSubscribedLibraryValidation(t *testing.T) {
	ctx := context.Background()

	_, err := vsphere.CreateSubscribedLibrary(ctx, vsphere.SubscribeConfig{})
	if err == nil || !strings.Contains(err.Error(), "url, username, and password") {
		t.Fatalf("expected url/username/password error; got %v", err)
	}

	_, err = vsphere.CreateSubscribedLibrary(ctx, vsphere.SubscribeConfig{
		URL: "vc01.example.com", Username: "u", Password: "p",
	})
	if err == nil || !strings.Contains(err.Error(), "library name, datastore, and publisher_subscription_url") {
		t.Fatalf("expected library fields error; got %v", err)
	}

	// Invalid URL should fail before a durable network dial when ParseURL rejects it.
	_, err = vsphere.CreateSubscribedLibrary(ctx, vsphere.SubscribeConfig{
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
