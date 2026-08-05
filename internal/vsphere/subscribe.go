// SPDX-License-Identifier: MIT

package vsphere

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vapi/library"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/tenthirtyam/artifactory-content-library/internal/logging"
)

// SubscribeConfig holds inputs for creating a subscribed library.
type SubscribeConfig struct {
	URL                      string
	Username                 string
	Password                 string
	Name                     string
	Datacenter               string
	Datastore                string
	PublisherSubscriptionURL string
	PublisherSSLThumbprint   string
	PublisherUsername        string
	PublisherPassword        string
	Insecure                 bool
	AutoSync                 bool
	OnDemand                 bool
}

// CreateSubscribedLibrary creates a subscribed content library using govmomi.
func CreateSubscribedLibrary(ctx context.Context, cfg SubscribeConfig) (string, error) {
	if cfg.URL == "" || cfg.Username == "" || cfg.Password == "" {
		return "", fmt.Errorf("vsphere url, username, and password are required")
	}
	if cfg.Name == "" || cfg.Datastore == "" || cfg.PublisherSubscriptionURL == "" {
		return "", fmt.Errorf("library name, datastore, and publisher_subscription_url are required")
	}

	u, err := soap.ParseURL(cfg.URL)
	if err != nil {
		u, err = soap.ParseURL("https://" + cfg.URL)
		if err != nil {
			return "", fmt.Errorf("invalid vsphere url: %w", err)
		}
	}
	u.User = url.UserPassword(cfg.Username, cfg.Password)

	logging.Info("Connecting to vSphere...", "url", cfg.URL)
	client, err := govmomi.NewClient(ctx, u, cfg.Insecure)
	if err != nil {
		return "", fmt.Errorf("connect to vSphere: %w", err)
	}
	defer func() { _ = client.Logout(ctx) }()

	finder := find.NewFinder(client.Client, true)
	dc, err := finder.DatacenterOrDefault(ctx, cfg.Datacenter)
	if err != nil {
		if cfg.Datacenter == "" {
			return "", fmt.Errorf("datacenter required when more than one exists (use --datacenter): %w", err)
		}
		return "", fmt.Errorf("datacenter %q not found: %w", cfg.Datacenter, err)
	}
	finder.SetDatacenter(dc)

	ds, err := finder.Datastore(ctx, cfg.Datastore)
	if err != nil {
		return "", fmt.Errorf("datastore %q not found in datacenter %q: %w", cfg.Datastore, dc.Name(), err)
	}

	restClient := rest.NewClient(client.Client)
	if err := restClient.Login(ctx, u.User); err != nil {
		return "", fmt.Errorf("vapi login: %w", err)
	}
	defer func() { _ = restClient.Logout(ctx) }()

	mgr := library.NewManager(restClient)

	autoSync := cfg.AutoSync
	onDemand := cfg.OnDemand
	sub := &library.Subscription{
		SubscriptionURL:      cfg.PublisherSubscriptionURL,
		AutomaticSyncEnabled: &autoSync,
		OnDemand:             &onDemand,
		AuthenticationMethod: "NONE",
	}
	if cfg.PublisherUsername != "" && cfg.PublisherPassword != "" {
		sub.AuthenticationMethod = "BASIC"
		sub.UserName = cfg.PublisherUsername
		sub.Password = cfg.PublisherPassword
	}
	if cfg.PublisherSSLThumbprint != "" {
		sub.SslThumbprint = cfg.PublisherSSLThumbprint
	}

	lib := library.Library{
		Name: cfg.Name,
		Type: "SUBSCRIBED",
		Storage: []library.StorageBacking{{
			DatastoreID: ds.Reference().Value,
			Type:        "DATASTORE",
		}},
		Subscription: sub,
	}

	logging.Info("Creating subscribed content library...", "name", cfg.Name)
	id, err := mgr.CreateLibrary(ctx, lib)
	if err != nil {
		return "", fmt.Errorf("create subscribed library: %w", err)
	}

	logging.Info("Successfully created subscribed content library",
		"library_id", id,
		"library_name", cfg.Name,
		"publisher_subscription_url", cfg.PublisherSubscriptionURL,
		"auto_sync", cfg.AutoSync,
		"on_demand", cfg.OnDemand,
	)
	return id, nil
}

// normalizeURL strips trailing slashes.
func normalizeURL(u string) string {
	return strings.TrimRight(u, "/")
}
