// SPDX-License-Identifier: MIT

package artifactory_test

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/tenthirtyam/artifactory-content-library/internal/artifactory"
	"github.com/tenthirtyam/artifactory-content-library/internal/auth"
	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

const (
	integrationBasePath          = "test-integration-library"
	integrationISONestedBasePath = "test-integration-iso-nested"
	integrationISOFlatBasePath   = "test-integration-iso-flat"
	integrationOVFFlatBasePath   = "test-integration-ovf-flat"
	integrationCombinedBasePath  = "test-integration-combined"
	integrationAuthSmokeBasePath = "test-integration-auth-smoke"
)

var integrationISONames = []string{
	"ubuntu-26.04-amd64",
	"ubuntu-26.04-arm64",
	"ubuntu-24.04-amd64",
	"ubuntu-24.04-arm64",
	"rhel-10-amd64",
	"rhel-10-arm64",
	"rhel-9-amd64",
	"rhel-9-arm64",
	"debian-13-amd64",
	"debian-13-arm64",
	"debian-12-amd64",
	"debian-12-arm64",
}

var integrationISONestedPaths = []string{
	"iso/ubuntu/ubuntu-26.04/amd64/ubuntu-26.04-amd64.iso",
	"iso/ubuntu/ubuntu-26.04/arm64/ubuntu-26.04-arm64.iso",
	"iso/ubuntu/ubuntu-24.04/amd64/ubuntu-24.04-amd64.iso",
	"iso/ubuntu/ubuntu-24.04/arm64/ubuntu-24.04-arm64.iso",
	"iso/rhel/rhel-10/amd64/rhel-10-amd64.iso",
	"iso/rhel/rhel-10/arm64/rhel-10-arm64.iso",
	"iso/rhel/rhel-9/amd64/rhel-9-amd64.iso",
	"iso/rhel/rhel-9/arm64/rhel-9-arm64.iso",
	"iso/debian/debian-13/amd64/debian-13-amd64.iso",
	"iso/debian/debian-13/arm64/debian-13-arm64.iso",
	"iso/debian/debian-12/amd64/debian-12-amd64.iso",
	"iso/debian/debian-12/arm64/debian-12-arm64.iso",
}

// Docs OVF/OVA layout (folder → files). Used by flat and combined tests.
var integrationOVFFlatUploads = map[string][]string{
	"ubuntu-26.04-amd64": {
		"ubuntu-26.04-amd64.ovf",
		"ubuntu-26.04-amd64-disk-0.vmdk",
		"ubuntu-26.04-amd64.mf",
	},
	"ubuntu-24.04-amd64": {
		"ubuntu-24.04-amd64.ova",
	},
	"rhel-10-amd64": {
		"rhel-10-amd64.ovf",
		"rhel-10-amd64-disk-0.vmdk",
		"rhel-10-amd64.mf",
	},
	"rhel-9-amd64": {
		"rhel-9-amd64.ova",
	},
}

func integrationEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration tests; set INTEGRATION_TESTS=1 to enable")
	}
}

func integrationCreds(t *testing.T) *auth.Credentials {
	t.Helper()
	url := envOr("ARTIFACTORY_URL", "http://localhost:8081/artifactory")
	repo := envOr("ARTIFACTORY_REPOSITORY", "example-repo-local")
	user := envOr("ARTIFACTORY_USERNAME", "admin")
	pass := envOr("ARTIFACTORY_PASSWORD", "password")

	creds, err := auth.Resolve(auth.Config{
		URL:            url,
		Repo:           repo,
		Username:       user,
		Password:       pass,
		TimeoutSeconds: 60,
		MaxRetries:     3,
		RateLimit:      20,
	})
	if err != nil {
		t.Fatalf("resolve creds: %v", err)
	}
	return creds
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newIntegrationClient(t *testing.T) *artifactory.Client {
	t.Helper()
	client, err := artifactory.NewClient(integrationCreds(t))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

func downloadJSON[T any](ctx context.Context, t *testing.T, client artifactory.StorageClient, relPath string) T {
	t.Helper()
	data, err := client.Download(ctx, relPath)
	if err != nil {
		t.Fatalf("download %s: %v", relPath, err)
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal %s: %v\n%s", relPath, err, string(data))
	}
	return out
}

func uploadBytes(ctx context.Context, t *testing.T, client artifactory.StorageClient, relPath string, payload []byte) {
	t.Helper()
	if err := client.Upload(ctx, relPath, payload, "application/octet-stream"); err != nil {
		t.Fatalf("upload %s: %v", relPath, err)
	}
}

// uploadIntegrationLibrary uploads the mental-model layout (ISO + OVA + OVF package)
// under integrationBasePath for generate / idempotency / version-bump tests.
func uploadIntegrationLibrary(ctx context.Context, t *testing.T, client artifactory.StorageClient) {
	t.Helper()
	base := integrationBasePath
	uploads := map[string][]byte{
		base + "/debian-iso/debian.iso":         []byte("integration-debian.iso\n"),
		base + "/debian-ova/debian.ova":         []byte("integration-debian.ova\n"),
		base + "/debian-ovf/debian.ovf":         []byte("integration-debian.ovf\n"),
		base + "/debian-ovf/debian-disk-0.vmdk": []byte("integration-debian.vmdk\n"),
		base + "/debian-ovf/debian.iso":         []byte("integration-debian-sidecar.iso\n"),
		base + "/debian-ovf/debian.mf":          []byte("integration-debian.mf\n"),
		base + "/debian-ovf/debian.nvram":       []byte("integration-debian.nvram\n"),
		base + "/debian-ovf/debian.cert":        []byte("integration-debian.cert\n"),
	}
	for rel, payload := range uploads {
		uploadBytes(ctx, t, client, rel, payload)
	}
}

func TestIntegrationGenerate(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	uploadIntegrationLibrary(ctx, t, client)

	if err := artifactory.Generate(ctx, client, "integration-lib", integrationBasePath, true); err != nil {
		t.Fatalf("generate: %v", err)
	}

	libPath := integrationBasePath + "/" + vcsp.LibFile
	itemsPath := integrationBasePath + "/" + vcsp.ItemsFile

	ok, err := client.FileExists(ctx, libPath)
	if err != nil || !ok {
		t.Fatalf("lib.json missing: ok=%v err=%v", ok, err)
	}
	ok, err = client.FileExists(ctx, itemsPath)
	if err != nil || !ok {
		t.Fatalf("items.json missing: ok=%v err=%v", ok, err)
	}

	lib := downloadJSON[vcsp.Library](ctx, t, client, libPath)
	if lib.Name != "integration-lib" {
		t.Fatalf("library name = %q", lib.Name)
	}
	if lib.Version == "" {
		t.Fatal("library version empty")
	}

	coll := downloadJSON[vcsp.ItemsCollection](ctx, t, client, itemsPath)
	wantTypes := map[string]vcsp.Type{
		"debian-iso": vcsp.TypeISO,
		"debian-ova": vcsp.TypeOVF,
		"debian-ovf": vcsp.TypeOVF,
	}
	byName := map[string]vcsp.Item{}
	for _, item := range coll.Items {
		byName[item.Name] = item
	}
	for name, typ := range wantTypes {
		it, ok := byName[name]
		if !ok {
			t.Fatalf("missing item %q; got %v", name, itemNames(byName))
		}
		if it.Type != typ {
			t.Fatalf("%s type = %q, want %q", name, it.Type, typ)
		}
		itemPath := integrationBasePath + "/" + it.SelfHref
		ok, err := client.FileExists(ctx, itemPath)
		if err != nil || !ok {
			t.Fatalf("item.json missing for %s (%s): ok=%v err=%v", name, itemPath, ok, err)
		}
	}
	// skip_cert=true: OVF package keeps ovf + vmdk + iso sidecar + mf + nvram (not .cert).
	if ovf := byName["debian-ovf"]; len(ovf.Files) != 5 {
		t.Fatalf("debian-ovf: got %d files, want 5 (cert skipped): %+v", len(ovf.Files), ovf.Files)
	}
}

func itemNames(byName map[string]vcsp.Item) []string {
	out := make([]string, 0, len(byName))
	for name := range byName {
		out = append(out, name)
	}
	return out
}

func assertISOItems(ctx context.Context, t *testing.T, client artifactory.StorageClient, basePath string, wantNames []string) {
	t.Helper()
	coll := downloadJSON[vcsp.ItemsCollection](ctx, t, client, basePath+"/"+vcsp.ItemsFile)
	if len(coll.Items) != len(wantNames) {
		t.Fatalf("expected %d ISO items, got %d: %+v", len(wantNames), len(coll.Items), coll.Items)
	}
	byName := map[string]vcsp.Item{}
	for _, it := range coll.Items {
		byName[it.Name] = it
	}
	for _, want := range wantNames {
		it, ok := byName[want]
		if !ok {
			t.Fatalf("missing item %q; got names %v", want, itemNames(byName))
		}
		if it.Type != vcsp.TypeISO || len(it.Files) != 1 {
			t.Fatalf("%s: %+v", want, it)
		}
		itemPath := basePath + "/" + it.SelfHref
		ok, err := client.FileExists(ctx, itemPath)
		if err != nil || !ok {
			t.Fatalf("item.json missing for %s (%s): ok=%v err=%v", want, itemPath, ok, err)
		}
	}
}

func TestIntegrationISONested(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	// Nested container tree: leaf folders with one ISO each become library items.
	for _, rel := range integrationISONestedPaths {
		objectPath := integrationISONestedBasePath + "/" + rel
		if err := client.Upload(ctx, objectPath, []byte("integration-iso-nested-"+rel+"\n"), "application/octet-stream"); err != nil {
			t.Fatalf("upload %s: %v", objectPath, err)
		}
	}

	if err := artifactory.Generate(ctx, client, "integration-iso-nested", integrationISONestedBasePath, true); err != nil {
		t.Fatalf("generate: %v", err)
	}

	assertISOItems(ctx, t, client, integrationISONestedBasePath, integrationISONames)

	for _, container := range []string{
		"iso/item.json",
		"iso/ubuntu/item.json",
		"iso/rhel/item.json",
		"iso/debian/item.json",
	} {
		ok, err := client.FileExists(ctx, integrationISONestedBasePath+"/"+container)
		if err != nil {
			t.Fatalf("FileExists %s: %v", container, err)
		}
		if ok {
			t.Fatalf("container must not get item.json: %s", container)
		}
	}
}

func TestIntegrationISOFlat(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	// Flat: one top-level folder per ISO (folder name = display name).
	for _, name := range integrationISONames {
		objectPath := integrationISOFlatBasePath + "/" + name + "/" + name + ".iso"
		if err := client.Upload(ctx, objectPath, []byte("integration-iso-flat-"+name+"\n"), "application/octet-stream"); err != nil {
			t.Fatalf("upload %s: %v", objectPath, err)
		}
	}

	if err := artifactory.Generate(ctx, client, "integration-iso-flat", integrationISOFlatBasePath, true); err != nil {
		t.Fatalf("generate: %v", err)
	}

	assertISOItems(ctx, t, client, integrationISOFlatBasePath, integrationISONames)
}

func TestIntegrationOVFFlat(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	// Docs layout: OVF packages (descriptor + disk + mf) and single-file OVAs.
	for folder, files := range integrationOVFFlatUploads {
		for _, file := range files {
			objectPath := integrationOVFFlatBasePath + "/" + folder + "/" + file
			if err := client.Upload(ctx, objectPath, []byte("integration-ovf-flat-"+file+"\n"), "application/octet-stream"); err != nil {
				t.Fatalf("upload %s: %v", objectPath, err)
			}
		}
	}

	if err := artifactory.Generate(ctx, client, "integration-ovf-flat", integrationOVFFlatBasePath, true); err != nil {
		t.Fatalf("generate: %v", err)
	}

	wantFiles := map[string]int{
		"ubuntu-26.04-amd64": 3,
		"ubuntu-24.04-amd64": 1,
		"rhel-10-amd64":      3,
		"rhel-9-amd64":       1,
	}
	coll := downloadJSON[vcsp.ItemsCollection](ctx, t, client, integrationOVFFlatBasePath+"/"+vcsp.ItemsFile)
	if len(coll.Items) != len(wantFiles) {
		t.Fatalf("expected %d OVF/OVA items, got %d: %+v", len(wantFiles), len(coll.Items), coll.Items)
	}
	byName := map[string]vcsp.Item{}
	for _, it := range coll.Items {
		byName[it.Name] = it
	}
	for name, fileCount := range wantFiles {
		it, ok := byName[name]
		if !ok {
			t.Fatalf("missing item %q; got names %v", name, itemNames(byName))
		}
		if it.Type != vcsp.TypeOVF {
			t.Fatalf("%s type = %q, want %q", name, it.Type, vcsp.TypeOVF)
		}
		if len(it.Files) != fileCount {
			t.Fatalf("%s: got %d files, want %d: %+v", name, len(it.Files), fileCount, it.Files)
		}
		itemPath := integrationOVFFlatBasePath + "/" + it.SelfHref
		ok, err := client.FileExists(ctx, itemPath)
		if err != nil || !ok {
			t.Fatalf("item.json missing for %s (%s): ok=%v err=%v", name, itemPath, ok, err)
		}
	}
}

func TestIntegrationCombined(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	// Remove prior combined layout that placed OVF packages at the library root with
	// the same display names as nested ISO basenames (vSphere rejects duplicates).
	for folder, files := range integrationOVFFlatUploads {
		for _, file := range files {
			_ = client.Delete(ctx, integrationCombinedBasePath+"/"+folder+"/"+file)
		}
		_ = client.Delete(ctx, integrationCombinedBasePath+"/"+folder+"/item.json")
	}

	// One library: nested ISO catalog + OVF/OVA packages under ovf/ with distinct
	// display names (vSphere rejects duplicate item names in a library).
	for _, rel := range integrationISONestedPaths {
		objectPath := integrationCombinedBasePath + "/" + rel
		if err := client.Upload(ctx, objectPath, []byte("integration-combined-"+rel+"\n"), "application/octet-stream"); err != nil {
			t.Fatalf("upload %s: %v", objectPath, err)
		}
	}
	// Folder suffix (-ovf/-ova) keeps display names unique vs nested ISO basenames.
	combinedOVF := map[string][]string{
		"ovf/ubuntu-26.04-amd64-ovf": {
			"ubuntu-26.04-amd64.ovf",
			"ubuntu-26.04-amd64-disk-0.vmdk",
			"ubuntu-26.04-amd64.mf",
		},
		"ovf/ubuntu-24.04-amd64-ova": {
			"ubuntu-24.04-amd64.ova",
		},
		"ovf/rhel-10-amd64-ovf": {
			"rhel-10-amd64.ovf",
			"rhel-10-amd64-disk-0.vmdk",
			"rhel-10-amd64.mf",
		},
		"ovf/rhel-9-amd64-ova": {
			"rhel-9-amd64.ova",
		},
	}
	for folder, files := range combinedOVF {
		for _, file := range files {
			objectPath := integrationCombinedBasePath + "/" + folder + "/" + file
			if err := client.Upload(ctx, objectPath, []byte("integration-combined-"+file+"\n"), "application/octet-stream"); err != nil {
				t.Fatalf("upload %s: %v", objectPath, err)
			}
		}
	}
	flatISOs := []string{"media-tools", "drivers"}
	for _, name := range flatISOs {
		objectPath := integrationCombinedBasePath + "/" + name + "/" + name + ".iso"
		if err := client.Upload(ctx, objectPath, []byte("integration-combined-"+name+"\n"), "application/octet-stream"); err != nil {
			t.Fatalf("upload %s: %v", objectPath, err)
		}
	}

	if err := artifactory.Generate(ctx, client, "integration-combined", integrationCombinedBasePath, true); err != nil {
		t.Fatalf("generate: %v", err)
	}

	want := map[string]struct {
		name  string
		typ   vcsp.Type
		files int
	}{}
	for _, rel := range integrationISONestedPaths {
		dir := path.Dir(rel)
		name := strings.TrimSuffix(path.Base(rel), ".iso")
		want[dir+"/item.json"] = struct {
			name  string
			typ   vcsp.Type
			files int
		}{name, vcsp.TypeISO, 1}
	}
	for folder, files := range combinedOVF {
		want[folder+"/item.json"] = struct {
			name  string
			typ   vcsp.Type
			files int
		}{path.Base(folder), vcsp.TypeOVF, len(files)}
	}
	for _, name := range flatISOs {
		want[name+"/item.json"] = struct {
			name  string
			typ   vcsp.Type
			files int
		}{name, vcsp.TypeISO, 1}
	}

	coll := downloadJSON[vcsp.ItemsCollection](ctx, t, client, integrationCombinedBasePath+"/"+vcsp.ItemsFile)
	if len(coll.Items) != len(want) {
		t.Fatalf("expected %d items, got %d: %+v", len(want), len(coll.Items), coll.Items)
	}
	byHref := map[string]vcsp.Item{}
	seenNames := map[string]string{}
	for _, it := range coll.Items {
		byHref[it.SelfHref] = it
		if prev, ok := seenNames[it.Name]; ok {
			t.Fatalf("duplicate display name %q for %q and %q", it.Name, prev, it.SelfHref)
		}
		seenNames[it.Name] = it.SelfHref
	}
	for href, expect := range want {
		it, ok := byHref[href]
		if !ok {
			t.Fatalf("missing SelfHref %q", href)
		}
		if it.Name != expect.name {
			t.Fatalf("%s name = %q, want %q", href, it.Name, expect.name)
		}
		if it.Type != expect.typ {
			t.Fatalf("%s type = %q, want %q", href, it.Type, expect.typ)
		}
		if len(it.Files) != expect.files {
			t.Fatalf("%s: got %d files, want %d", href, len(it.Files), expect.files)
		}
		ok, err := client.FileExists(ctx, integrationCombinedBasePath+"/"+href)
		if err != nil || !ok {
			t.Fatalf("item.json missing for %s: ok=%v err=%v", href, ok, err)
		}
	}

	for _, container := range []string{
		"iso/item.json",
		"iso/ubuntu/item.json",
		"iso/rhel/item.json",
		"iso/debian/item.json",
		"ovf/item.json",
	} {
		ok, err := client.FileExists(ctx, integrationCombinedBasePath+"/"+container)
		if err != nil {
			t.Fatalf("FileExists %s: %v", container, err)
		}
		if ok {
			t.Fatalf("container must not get item.json: %s", container)
		}
	}
}

func TestIntegrationIdempotentRerun(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	uploadIntegrationLibrary(ctx, t, client)
	if err := artifactory.Generate(ctx, client, "integration-lib", integrationBasePath, true); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	libPath := integrationBasePath + "/" + vcsp.LibFile
	first := downloadJSON[vcsp.Library](ctx, t, client, libPath)

	if err := artifactory.Generate(ctx, client, "integration-lib", integrationBasePath, true); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	second := downloadJSON[vcsp.Library](ctx, t, client, libPath)

	if first.Version != second.Version {
		t.Fatalf("expected unchanged library version on idempotent re-run; got %q then %q", first.Version, second.Version)
	}
	if first.ID != second.ID {
		t.Fatalf("library id changed: %q -> %q", first.ID, second.ID)
	}
}

func TestIntegrationContentChangeBumpsVersion(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	uploadIntegrationLibrary(ctx, t, client)
	if err := artifactory.Generate(ctx, client, "integration-lib", integrationBasePath, true); err != nil {
		t.Fatalf("baseline generate: %v", err)
	}
	libPath := integrationBasePath + "/" + vcsp.LibFile
	before := downloadJSON[vcsp.Library](ctx, t, client, libPath)

	isoPath := integrationBasePath + "/debian-iso/debian.iso"
	payload := []byte("changed-iso-content-" + time.Now().UTC().Format(time.RFC3339Nano) + "\n")
	uploadBytes(ctx, t, client, isoPath, payload)

	// Give Artifactory a moment to settle checksums.
	time.Sleep(2 * time.Second)

	if err := artifactory.Generate(ctx, client, "integration-lib", integrationBasePath, true); err != nil {
		t.Fatalf("generate after change: %v", err)
	}
	after := downloadJSON[vcsp.Library](ctx, t, client, libPath)

	if before.Version == after.Version {
		t.Fatalf("expected library version bump after content change; still %q", after.Version)
	}
}

func TestIntegrationBasePath(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	uploadIntegrationLibrary(ctx, t, client)
	libPath := integrationBasePath + "/" + vcsp.LibFile
	rootLib := vcsp.LibFile

	if err := artifactory.Generate(ctx, client, "integration-lib", integrationBasePath, true); err != nil {
		t.Fatalf("generate: %v", err)
	}

	ok, _ := client.FileExists(ctx, libPath)
	if !ok {
		t.Fatalf("expected %s", libPath)
	}
	ok, _ = client.FileExists(ctx, rootLib)
	if ok {
		t.Fatalf("did not expect lib.json at repository root")
	}
}

func TestIntegrationBasicAuthSmoke(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	client := newIntegrationClient(t)

	// Use a dedicated path so the probe folder does not pollute generate tests.
	uploadBytes(ctx, t, client, integrationAuthSmokeBasePath+"/probe.txt", []byte("ok\n"))
	_, err := client.ListItems(ctx, integrationAuthSmokeBasePath)
	if err != nil {
		t.Fatalf("basic auth list failed: %v", err)
	}
}

func TestIntegrationBadCredentials(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()

	creds, err := auth.Resolve(auth.Config{
		URL:            envOr("ARTIFACTORY_URL", "http://localhost:8081/artifactory"),
		Repo:           envOr("ARTIFACTORY_REPOSITORY", "example-repo-local"),
		Username:       "admin",
		Password:       "definitely-wrong-password",
		TimeoutSeconds: 15,
		MaxRetries:     1,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	client, err := artifactory.NewClient(creds)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.ListItems(ctx, integrationAuthSmokeBasePath)
	if err == nil {
		t.Fatal("expected authentication failure")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "401") && !strings.Contains(msg, "403") &&
		!strings.Contains(msg, "unauthorized") && !strings.Contains(msg, "forbidden") &&
		!strings.Contains(msg, "authentication") && !strings.Contains(msg, "status") {
		// Artifactory error strings vary; ensure we at least failed.
		t.Logf("got expected failure with message: %v", err)
	}
}
