// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package artifactory_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/tenthirtyam/artifactory-content-library/internal/artifactory"
	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

type mockStorage struct {
	mu      sync.Mutex
	files   map[string][]byte
	dirs    map[string][]artifactory.ChildItem
	meta    map[string]*artifactory.FileMeta
	deleted []string
}

func newMock() *mockStorage {
	return &mockStorage{
		files: map[string][]byte{},
		dirs:  map[string][]artifactory.ChildItem{},
		meta:  map[string]*artifactory.FileMeta{},
	}
}

func (m *mockStorage) FileExists(_ context.Context, p string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.files[p]
	return ok, nil
}

func (m *mockStorage) GetFileInfo(_ context.Context, p string) (*artifactory.FileMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if meta, ok := m.meta[p]; ok {
		return meta, nil
	}
	if b, ok := m.files[p]; ok {
		return &artifactory.FileMeta{Size: int64(len(b)), SHA1: "abc", MD5: "def"}, nil
	}
	return nil, nil
}

func (m *mockStorage) ListItems(_ context.Context, p string) ([]artifactory.ChildItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dirs[p], nil
}

func (m *mockStorage) Download(_ context.Context, p string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.files[p], nil
}

func (m *mockStorage) Upload(_ context.Context, p string, content []byte, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[p] = content
	return nil
}

func (m *mockStorage) Delete(_ context.Context, p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleted = append(m.deleted, p)
	delete(m.files, p)
	return nil
}

func seedISOItem(m *mockStorage, folder, file, sha1 string) {
	m.dirs[""] = []artifactory.ChildItem{{URI: "/" + folder, Folder: true}}
	m.dirs[folder] = []artifactory.ChildItem{{URI: "/" + file, Folder: false}}
	m.meta[folder+"/"+file] = &artifactory.FileMeta{Size: 10, SHA1: sha1, MD5: "md5"}
	m.files[folder+"/"+file] = []byte("iso-data")
}

// seedISOFolder seeds a top-level folder containing one or more ISO files (and optional nested folders).
func seedISOFolder(m *mockStorage, folder string, files []string, nestedFolders []string) {
	var children []artifactory.ChildItem
	for _, file := range files {
		children = append(children, artifactory.ChildItem{URI: "/" + file, Folder: false})
		m.meta[folder+"/"+file] = &artifactory.FileMeta{Size: 10, SHA1: "sha-" + file, MD5: "md5"}
		m.files[folder+"/"+file] = []byte("iso-data-" + file)
	}
	for _, nested := range nestedFolders {
		children = append(children, artifactory.ChildItem{URI: "/" + nested, Folder: true})
	}
	m.dirs[folder] = children

	root := append([]artifactory.ChildItem{}, m.dirs[""]...)
	root = append(root, artifactory.ChildItem{URI: "/" + folder, Folder: true})
	m.dirs[""] = root
}

func itemsCollection(t *testing.T, m *mockStorage) vcsp.ItemsCollection {
	t.Helper()
	var coll vcsp.ItemsCollection
	data, ok := m.files[vcsp.ItemsFile]
	if !ok {
		t.Fatalf("expected %s; files=%v", vcsp.ItemsFile, keys(m.files))
	}
	if err := json.Unmarshal(data, &coll); err != nil {
		t.Fatalf("unmarshal items.json: %v", err)
	}
	return coll
}

func libVersion(t *testing.T, m *mockStorage) string {
	t.Helper()
	var lib vcsp.Library
	if err := json.Unmarshal(m.files[vcsp.LibFile], &lib); err != nil {
		t.Fatalf("unmarshal lib.json: %v", err)
	}
	return lib.Version
}

func TestGenerateArtifactoryNewLibrary(t *testing.T) {
	m := newMock()
	seedISOItem(m, "item1", "disk.iso", "sha1")

	if err := artifactory.Generate(context.Background(), m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.files[vcsp.LibFile]; !ok {
		t.Fatal("missing lib.json")
	}
	if _, ok := m.files[vcsp.ItemsFile]; !ok {
		t.Fatal("missing items.json")
	}
}

func TestGenerateIdempotent(t *testing.T) {
	m := newMock()
	seedISOItem(m, "item1", "disk.iso", "sha1")
	ctx := context.Background()

	if err := artifactory.Generate(ctx, m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}
	first := libVersion(t, m)

	if err := artifactory.Generate(ctx, m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}
	second := libVersion(t, m)
	if first != second {
		t.Fatalf("expected unchanged version; got %q then %q", first, second)
	}
}

func TestGenerateContentChangeBumpsVersion(t *testing.T) {
	m := newMock()
	seedISOItem(m, "item1", "disk.iso", "sha1-old")
	ctx := context.Background()

	if err := artifactory.Generate(ctx, m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}
	before := libVersion(t, m)

	m.meta["item1/disk.iso"] = &artifactory.FileMeta{Size: 10, SHA1: "sha1-new", MD5: "md5"}
	if err := artifactory.Generate(ctx, m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}
	after := libVersion(t, m)
	if before == after {
		t.Fatalf("expected version bump; still %q", after)
	}
}

func TestGenerateOrphanISODelete(t *testing.T) {
	m := newMock()
	seedISOItem(m, "item1", "disk.iso", "sha1")
	ctx := context.Background()

	if err := artifactory.Generate(ctx, m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}

	// Inject a legacy nested ISO orphan (folder/basename encoded as a single segment).
	var coll vcsp.ItemsCollection
	if err := json.Unmarshal(m.files[vcsp.ItemsFile], &coll); err != nil {
		t.Fatal(err)
	}
	orphanPath := "gone%2Fchild/item.json"
	m.files[orphanPath] = []byte(`{}`)
	coll.Items = append(coll.Items, vcsp.Item{
		Name:     "child",
		Type:     vcsp.TypeISO,
		Version:  "1",
		ID:       "urn:uuid:orphan",
		SelfHref: orphanPath,
	})
	data, err := artifactory.MarshalIndent(coll)
	if err != nil {
		t.Fatal(err)
	}
	m.files[vcsp.ItemsFile] = data

	if err := artifactory.Generate(ctx, m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}

	found := slices.Contains(m.deleted, orphanPath)
	if !found {
		t.Fatalf("expected delete of %s; deleted=%v", orphanPath, m.deleted)
	}
}

func TestGenerateISOItemJSONAlongsideFiles(t *testing.T) {
	m := newMock()
	seedISOItem(m, "debian-iso", "debian.iso", "sha1")

	if err := artifactory.Generate(context.Background(), m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.files["debian-iso/item.json"]; !ok {
		t.Fatalf("expected debian-iso/item.json; files=%v", keys(m.files))
	}
	for p := range m.files {
		if strings.Contains(p, "%2F") {
			t.Fatalf("unexpected encoded path %q", p)
		}
	}
	var coll vcsp.ItemsCollection
	if err := json.Unmarshal(m.files[vcsp.ItemsFile], &coll); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range coll.Items {
		if it.Name == "debian-iso" && it.Type == vcsp.TypeISO && it.SelfHref == "debian-iso/item.json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected debian-iso item; got %+v", coll.Items)
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestGenerateValidation(t *testing.T) {
	m := newMock()
	if err := artifactory.Generate(context.Background(), m, "", "", true); err == nil {
		t.Fatal("expected empty library name error")
	}
	if err := artifactory.Generate(context.Background(), m, "Lib", "../escape", true); err == nil {
		t.Fatal("expected invalid base path error")
	}
}

func TestGenerateEmptyDirectories(t *testing.T) {
	m := newMock()
	m.dirs[""] = nil
	if err := artifactory.Generate(context.Background(), m, "Empty", "", true); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.files[vcsp.LibFile]; ok {
		t.Fatal("should not write lib.json when no folders exist")
	}
}

func TestGenerateBasePath(t *testing.T) {
	m := newMock()
	base := "prefix"
	m.dirs[base] = []artifactory.ChildItem{{URI: "/item1", Folder: true}}
	m.dirs[base+"/item1"] = []artifactory.ChildItem{{URI: "/disk.iso", Folder: false}}
	m.meta[base+"/item1/disk.iso"] = &artifactory.FileMeta{Size: 10, SHA1: "sha1"}
	m.files[base+"/item1/disk.iso"] = []byte("iso")

	if err := artifactory.Generate(context.Background(), m, "Lib", base, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.files[base+"/"+vcsp.LibFile]; !ok {
		t.Fatal("missing prefixed lib.json")
	}
	if _, ok := m.files[vcsp.LibFile]; ok {
		t.Fatal("did not expect root lib.json")
	}
}

func TestDirToItemOVF(t *testing.T) {
	m := newMock()
	m.dirs["ovf-item"] = []artifactory.ChildItem{
		{URI: "/template.ovf", Folder: false},
		{URI: "/disk.vmdk", Folder: false},
	}
	m.meta["ovf-item/template.ovf"] = &artifactory.FileMeta{Size: 100, SHA1: "s1"}
	m.meta["ovf-item/disk.vmdk"] = &artifactory.FileMeta{Size: 200, SHA1: "s2"}
	m.files["ovf-item/template.ovf"] = []byte(`<Envelope><VirtualSystemCollection/></Envelope>`)

	items, err := artifactory.DirToItem(context.Background(), m, "ovf-item", "ovf-item", true, "urn:uuid:lib", nil)
	if err != nil {
		t.Fatal(err)
	}
	it, ok := items["ovf-item"]
	if !ok {
		t.Fatalf("items=%v", items)
	}
	if it.Type != vcsp.TypeOVF {
		t.Fatalf("type %s", it.Type)
	}
	if len(it.Metadata) == 0 || !strings.Contains(it.Metadata[0].Value, `"isVappTemplate":"true"`) {
		t.Fatalf("expected vapp metadata: %+v", it.Metadata)
	}
}

func TestDirToItemSkipCert(t *testing.T) {
	m := newMock()
	m.dirs["ovf"] = []artifactory.ChildItem{
		{URI: "/a.ovf", Folder: false},
		{URI: "/a.cert", Folder: false},
	}
	m.meta["ovf/a.ovf"] = &artifactory.FileMeta{Size: 1, SHA1: "1"}
	m.meta["ovf/a.cert"] = &artifactory.FileMeta{Size: 1, SHA1: "2"}
	m.files["ovf/a.ovf"] = []byte("<Envelope></Envelope>")

	items, err := artifactory.DirToItem(context.Background(), m, "ovf", "ovf", true, "urn:uuid:x", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range items["ovf"].Files {
		if strings.HasSuffix(f.Name, ".cert") {
			t.Fatal("cert should be skipped")
		}
	}
}

func TestDirToItemISOOnly(t *testing.T) {
	m := newMock()
	m.dirs["iso-folder"] = []artifactory.ChildItem{{URI: "/ubuntu.iso", Folder: false}}
	m.meta["iso-folder/ubuntu.iso"] = &artifactory.FileMeta{Size: 5, SHA1: "", MD5: "md5etag"}

	items, err := artifactory.DirToItem(context.Background(), m, "iso-folder", "iso-folder", true, "urn:uuid:lib", nil)
	if err != nil {
		t.Fatal(err)
	}
	it, ok := items["iso-folder"]
	if !ok {
		t.Fatalf("expected folder-level ISO item; got %v", items)
	}
	if it.Type != vcsp.TypeISO {
		t.Fatalf("type %s", it.Type)
	}
	if it.SelfHref != "iso-folder/item.json" {
		t.Fatalf("selfHref %q", it.SelfHref)
	}
	if len(it.Files) != 1 || it.Files[0].ETag != "md5etag" {
		t.Fatalf("expected MD5 etag fallback: %+v", it.Files)
	}
}

func TestDirToItemMultipleISOsOneItem(t *testing.T) {
	// Several .iso files in one folder become a single library item with multiple files,
	// not one library item per ISO.
	m := newMock()
	m.dirs["iso-bundle"] = []artifactory.ChildItem{
		{URI: "/a.iso", Folder: false},
		{URI: "/b.iso", Folder: false},
		{URI: "/c.iso", Folder: false},
	}
	for _, name := range []string{"a.iso", "b.iso", "c.iso"} {
		m.meta["iso-bundle/"+name] = &artifactory.FileMeta{Size: 5, SHA1: "sha-" + name}
		m.files["iso-bundle/"+name] = []byte(name)
	}

	items, err := artifactory.DirToItem(context.Background(), m, "iso-bundle", "iso-bundle", true, "urn:uuid:lib", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 library item for the folder, got %d: %v", len(items), items)
	}
	it := items["iso-bundle"]
	if it.Type != vcsp.TypeISO {
		t.Fatalf("type %s", it.Type)
	}
	if len(it.Files) != 3 {
		t.Fatalf("expected 3 files on the single item, got %+v", it.Files)
	}
	names := map[string]bool{}
	for _, f := range it.Files {
		names[f.Name] = true
	}
	for _, want := range []string{"a.iso", "b.iso", "c.iso"} {
		if !names[want] {
			t.Fatalf("missing file %q in %+v", want, it.Files)
		}
	}
}

func TestDirToItemSkipsNestedFolders(t *testing.T) {
	// DirToItem itself does not recurse; nested folders under an item folder are ignored.
	m := newMock()
	m.dirs["iso"] = []artifactory.ChildItem{
		{URI: "/top.iso", Folder: false},
		{URI: "/ubuntu", Folder: true},
		{URI: "/centos", Folder: true},
	}
	m.meta["iso/top.iso"] = &artifactory.FileMeta{Size: 5, SHA1: "top"}
	m.files["iso/top.iso"] = []byte("top")
	m.dirs["iso/ubuntu"] = []artifactory.ChildItem{{URI: "/ubuntu.iso", Folder: false}}
	m.meta["iso/ubuntu/ubuntu.iso"] = &artifactory.FileMeta{Size: 5, SHA1: "ubuntu"}
	m.files["iso/ubuntu/ubuntu.iso"] = []byte("ubuntu")
	m.dirs["iso/centos"] = []artifactory.ChildItem{{URI: "/centos.iso", Folder: false}}
	m.meta["iso/centos/centos.iso"] = &artifactory.FileMeta{Size: 5, SHA1: "centos"}
	m.files["iso/centos/centos.iso"] = []byte("centos")

	items, err := artifactory.DirToItem(context.Background(), m, "iso", "iso", true, "urn:uuid:lib", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d: %v", len(items), items)
	}
	it := items["iso"]
	if len(it.Files) != 1 || it.Files[0].Name != "top.iso" {
		t.Fatalf("expected only direct top.iso; nested ISOs should be skipped: %+v", it.Files)
	}
}

func TestGenerateMultipleISOsInOneFolder(t *testing.T) {
	m := newMock()
	seedISOFolder(m, "media", []string{"installer.iso", "tools.iso", "drivers.iso"}, nil)

	if err := artifactory.Generate(context.Background(), m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}

	coll := itemsCollection(t, m)
	if len(coll.Items) != 1 {
		t.Fatalf("expected 1 library item (folder-as-item), not one per ISO; got %d: %+v", len(coll.Items), coll.Items)
	}
	it := coll.Items[0]
	if it.Name != "media" || it.Type != vcsp.TypeISO {
		t.Fatalf("unexpected item: %+v", it)
	}
	if len(it.Files) != 3 {
		t.Fatalf("expected 3 ISO files on the item, got %+v", it.Files)
	}
	if _, ok := m.files["media/item.json"]; !ok {
		t.Fatal("expected media/item.json")
	}
	for _, name := range []string{"installer.iso", "tools.iso", "drivers.iso"} {
		if _, ok := m.files["media/"+strings.TrimSuffix(name, ".iso")+"/item.json"]; ok {
			t.Fatalf("unexpected per-ISO item.json for %s", name)
		}
	}
}

func TestGenerateRootLevelISOsIgnored(t *testing.T) {
	// Loose .iso files at the library root are not content library items.
	m := newMock()
	m.dirs[""] = []artifactory.ChildItem{
		{URI: "/root-a.iso", Folder: false},
		{URI: "/root-b.iso", Folder: false},
		{URI: "/debian-iso", Folder: true},
	}
	m.meta["root-a.iso"] = &artifactory.FileMeta{Size: 5, SHA1: "a"}
	m.files["root-a.iso"] = []byte("a")
	m.meta["root-b.iso"] = &artifactory.FileMeta{Size: 5, SHA1: "b"}
	m.files["root-b.iso"] = []byte("b")
	m.dirs["debian-iso"] = []artifactory.ChildItem{{URI: "/debian.iso", Folder: false}}
	m.meta["debian-iso/debian.iso"] = &artifactory.FileMeta{Size: 5, SHA1: "d"}
	m.files["debian-iso/debian.iso"] = []byte("d")

	if err := artifactory.Generate(context.Background(), m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}

	coll := itemsCollection(t, m)
	if len(coll.Items) != 1 {
		t.Fatalf("expected only the folder item; root ISOs should be ignored; got %d: %+v", len(coll.Items), coll.Items)
	}
	if coll.Items[0].Name != "debian-iso" {
		t.Fatalf("expected debian-iso item, got %+v", coll.Items[0])
	}
	if _, ok := m.files["root-a.iso/item.json"]; ok {
		t.Fatal("root ISO must not produce an item.json")
	}
}

func TestGenerateNestedISOCatalog(t *testing.T) {
	// Container folders are walked; each leaf folder with an ISO becomes one item.
	m := newMock()
	m.dirs[""] = []artifactory.ChildItem{{URI: "/iso", Folder: true}}
	m.dirs["iso"] = []artifactory.ChildItem{
		{URI: "/ubuntu", Folder: true},
		{URI: "/rhel", Folder: true},
		{URI: "/debian", Folder: true},
	}
	m.dirs["iso/ubuntu"] = []artifactory.ChildItem{
		{URI: "/ubuntu-26.04", Folder: true},
		{URI: "/ubuntu-24.04", Folder: true},
	}
	m.dirs["iso/ubuntu/ubuntu-26.04"] = []artifactory.ChildItem{
		{URI: "/amd64", Folder: true},
		{URI: "/arm64", Folder: true},
	}
	m.dirs["iso/ubuntu/ubuntu-26.04/amd64"] = []artifactory.ChildItem{{URI: "/ubuntu-26.04-amd64.iso", Folder: false}}
	m.meta["iso/ubuntu/ubuntu-26.04/amd64/ubuntu-26.04-amd64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "u64"}
	m.files["iso/ubuntu/ubuntu-26.04/amd64/ubuntu-26.04-amd64.iso"] = []byte("u64")
	m.dirs["iso/ubuntu/ubuntu-26.04/arm64"] = []artifactory.ChildItem{{URI: "/ubuntu-26.04-arm64.iso", Folder: false}}
	m.meta["iso/ubuntu/ubuntu-26.04/arm64/ubuntu-26.04-arm64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "uarm"}
	m.files["iso/ubuntu/ubuntu-26.04/arm64/ubuntu-26.04-arm64.iso"] = []byte("uarm")

	m.dirs["iso/ubuntu/ubuntu-24.04"] = []artifactory.ChildItem{
		{URI: "/amd64", Folder: true},
		{URI: "/arm64", Folder: true},
	}
	m.dirs["iso/ubuntu/ubuntu-24.04/amd64"] = []artifactory.ChildItem{{URI: "/ubuntu-24.04-amd64.iso", Folder: false}}
	m.meta["iso/ubuntu/ubuntu-24.04/amd64/ubuntu-24.04-amd64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "u24a"}
	m.files["iso/ubuntu/ubuntu-24.04/amd64/ubuntu-24.04-amd64.iso"] = []byte("u24a")
	m.dirs["iso/ubuntu/ubuntu-24.04/arm64"] = []artifactory.ChildItem{{URI: "/ubuntu-24.04-arm64.iso", Folder: false}}
	m.meta["iso/ubuntu/ubuntu-24.04/arm64/ubuntu-24.04-arm64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "u24r"}
	m.files["iso/ubuntu/ubuntu-24.04/arm64/ubuntu-24.04-arm64.iso"] = []byte("u24r")

	m.dirs["iso/rhel"] = []artifactory.ChildItem{
		{URI: "/rhel-10", Folder: true},
		{URI: "/rhel-9", Folder: true},
	}
	m.dirs["iso/rhel/rhel-10"] = []artifactory.ChildItem{
		{URI: "/amd64", Folder: true},
		{URI: "/arm64", Folder: true},
	}
	m.dirs["iso/rhel/rhel-10/amd64"] = []artifactory.ChildItem{{URI: "/rhel-10-amd64.iso", Folder: false}}
	m.meta["iso/rhel/rhel-10/amd64/rhel-10-amd64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "r10a"}
	m.files["iso/rhel/rhel-10/amd64/rhel-10-amd64.iso"] = []byte("r10a")
	m.dirs["iso/rhel/rhel-10/arm64"] = []artifactory.ChildItem{{URI: "/rhel-10-arm64.iso", Folder: false}}
	m.meta["iso/rhel/rhel-10/arm64/rhel-10-arm64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "r10r"}
	m.files["iso/rhel/rhel-10/arm64/rhel-10-arm64.iso"] = []byte("r10r")

	m.dirs["iso/rhel/rhel-9"] = []artifactory.ChildItem{
		{URI: "/amd64", Folder: true},
		{URI: "/arm64", Folder: true},
	}
	m.dirs["iso/rhel/rhel-9/amd64"] = []artifactory.ChildItem{{URI: "/rhel-9-amd64.iso", Folder: false}}
	m.meta["iso/rhel/rhel-9/amd64/rhel-9-amd64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "r9a"}
	m.files["iso/rhel/rhel-9/amd64/rhel-9-amd64.iso"] = []byte("r9a")
	m.dirs["iso/rhel/rhel-9/arm64"] = []artifactory.ChildItem{{URI: "/rhel-9-arm64.iso", Folder: false}}
	m.meta["iso/rhel/rhel-9/arm64/rhel-9-arm64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "r9r"}
	m.files["iso/rhel/rhel-9/arm64/rhel-9-arm64.iso"] = []byte("r9r")

	m.dirs["iso/debian"] = []artifactory.ChildItem{
		{URI: "/debian-13", Folder: true},
		{URI: "/debian-12", Folder: true},
	}
	m.dirs["iso/debian/debian-13"] = []artifactory.ChildItem{
		{URI: "/amd64", Folder: true},
		{URI: "/arm64", Folder: true},
	}
	m.dirs["iso/debian/debian-13/amd64"] = []artifactory.ChildItem{{URI: "/debian-13-amd64.iso", Folder: false}}
	m.meta["iso/debian/debian-13/amd64/debian-13-amd64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "d13a"}
	m.files["iso/debian/debian-13/amd64/debian-13-amd64.iso"] = []byte("d13a")
	m.dirs["iso/debian/debian-13/arm64"] = []artifactory.ChildItem{{URI: "/debian-13-arm64.iso", Folder: false}}
	m.meta["iso/debian/debian-13/arm64/debian-13-arm64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "d13r"}
	m.files["iso/debian/debian-13/arm64/debian-13-arm64.iso"] = []byte("d13r")

	m.dirs["iso/debian/debian-12"] = []artifactory.ChildItem{
		{URI: "/amd64", Folder: true},
		{URI: "/arm64", Folder: true},
	}
	m.dirs["iso/debian/debian-12/amd64"] = []artifactory.ChildItem{{URI: "/debian-12-amd64.iso", Folder: false}}
	m.meta["iso/debian/debian-12/amd64/debian-12-amd64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "d12a"}
	m.files["iso/debian/debian-12/amd64/debian-12-amd64.iso"] = []byte("d12a")
	m.dirs["iso/debian/debian-12/arm64"] = []artifactory.ChildItem{{URI: "/debian-12-arm64.iso", Folder: false}}
	m.meta["iso/debian/debian-12/arm64/debian-12-arm64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "d12r"}
	m.files["iso/debian/debian-12/arm64/debian-12-arm64.iso"] = []byte("d12r")

	if err := artifactory.Generate(context.Background(), m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}

	coll := itemsCollection(t, m)
	wantNames := []string{
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
	if len(coll.Items) != len(wantNames) {
		t.Fatalf("expected %d leaf ISO items, got %d: %+v", len(wantNames), len(coll.Items), coll.Items)
	}

	byName := map[string]vcsp.Item{}
	for _, it := range coll.Items {
		byName[it.Name] = it
	}
	for _, want := range wantNames {
		it, ok := byName[want]
		if !ok {
			t.Fatalf("missing item %q; got names %v", want, keysOfItems(byName))
		}
		if it.Type != vcsp.TypeISO || len(it.Files) != 1 {
			t.Fatalf("%s: %+v", want, it)
		}
	}

	if _, ok := m.files["iso/ubuntu/ubuntu-26.04/amd64/item.json"]; !ok {
		t.Fatalf("expected nested item.json; files=%v", keys(m.files))
	}
	if _, ok := m.files["iso/debian/debian-13/amd64/item.json"]; !ok {
		t.Fatalf("expected debian nested item.json; files=%v", keys(m.files))
	}
	if _, ok := m.files["iso/item.json"]; ok {
		t.Fatal("container folder iso/ must not get an item.json")
	}
}

func keysOfItems(m map[string]vcsp.Item) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestGenerateNestedISOTreeSiblingAndContainers(t *testing.T) {
	// Mix of container-only tree and flat sibling folders.
	m := newMock()
	m.dirs[""] = []artifactory.ChildItem{
		{URI: "/iso", Folder: true},
		{URI: "/ubuntu", Folder: true},
		{URI: "/centos", Folder: true},
	}

	m.dirs["iso"] = []artifactory.ChildItem{
		{URI: "/nested-a", Folder: true},
		{URI: "/nested-b", Folder: true},
	}
	m.dirs["iso/nested-a"] = []artifactory.ChildItem{{URI: "/a.iso", Folder: false}}
	m.meta["iso/nested-a/a.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "a"}
	m.files["iso/nested-a/a.iso"] = []byte("a")
	m.dirs["iso/nested-b"] = []artifactory.ChildItem{{URI: "/b.iso", Folder: false}}
	m.meta["iso/nested-b/b.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "b"}
	m.files["iso/nested-b/b.iso"] = []byte("b")

	m.dirs["ubuntu"] = []artifactory.ChildItem{{URI: "/ubuntu.iso", Folder: false}}
	m.meta["ubuntu/ubuntu.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "u"}
	m.files["ubuntu/ubuntu.iso"] = []byte("u")
	m.dirs["centos"] = []artifactory.ChildItem{{URI: "/centos.iso", Folder: false}}
	m.meta["centos/centos.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "c"}
	m.files["centos/centos.iso"] = []byte("c")

	if err := artifactory.Generate(context.Background(), m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}

	coll := itemsCollection(t, m)
	byName := map[string]vcsp.Item{}
	for _, it := range coll.Items {
		byName[it.Name] = it
	}

	if _, ok := byName["iso"]; ok {
		t.Fatalf("iso/ container should not be an item; got %+v", byName["iso"])
	}
	if len(coll.Items) != 4 {
		t.Fatalf("expected nested-a, nested-b, ubuntu, centos (4 items), got %d: %+v", len(coll.Items), coll.Items)
	}
	if byName["a"].Type != vcsp.TypeISO { // nested single-ISO display name from basename
		t.Fatalf("expected nested item display name a; names=%v", keysOfItems(byName))
	}
	if byName["ubuntu"].Type != vcsp.TypeISO || len(byName["ubuntu"].Files) != 1 {
		t.Fatalf("ubuntu item: %+v", byName["ubuntu"])
	}
	if byName["centos"].Type != vcsp.TypeISO || len(byName["centos"].Files) != 1 {
		t.Fatalf("centos item: %+v", byName["centos"])
	}
}

func TestGenerateManyISOsAcrossSiblingFolders(t *testing.T) {
	// A flat "one folder per ISO" layout creates one library item per folder.
	m := newMock()
	folders := []string{"iso-1", "iso-2", "iso-3", "iso-4", "iso-5"}
	var root []artifactory.ChildItem
	for _, folder := range folders {
		root = append(root, artifactory.ChildItem{URI: "/" + folder, Folder: true})
		file := folder + ".iso"
		m.dirs[folder] = []artifactory.ChildItem{{URI: "/" + file, Folder: false}}
		m.meta[folder+"/"+file] = &artifactory.FileMeta{Size: 1, SHA1: folder}
		m.files[folder+"/"+file] = []byte(folder)
	}
	m.dirs[""] = root

	if err := artifactory.Generate(context.Background(), m, "ArtLib", "", true); err != nil {
		t.Fatal(err)
	}

	coll := itemsCollection(t, m)
	if len(coll.Items) != 5 {
		t.Fatalf("expected 5 items (one per sibling folder), got %d: %+v", len(coll.Items), coll.Items)
	}
}

func TestGenerateDuplicateDisplayName(t *testing.T) {
	// Nested ISO basename and sibling OVF folder share the same display name.
	m := newMock()
	m.dirs[""] = []artifactory.ChildItem{
		{URI: "/iso", Folder: true},
		{URI: "/rhel-10-amd64", Folder: true},
	}
	m.dirs["iso"] = []artifactory.ChildItem{{URI: "/leaf", Folder: true}}
	m.dirs["iso/leaf"] = []artifactory.ChildItem{{URI: "/rhel-10-amd64.iso", Folder: false}}
	m.meta["iso/leaf/rhel-10-amd64.iso"] = &artifactory.FileMeta{Size: 1, SHA1: "iso"}
	m.files["iso/leaf/rhel-10-amd64.iso"] = []byte("iso")

	m.dirs["rhel-10-amd64"] = []artifactory.ChildItem{{URI: "/rhel-10-amd64.ova", Folder: false}}
	m.meta["rhel-10-amd64/rhel-10-amd64.ova"] = &artifactory.FileMeta{Size: 1, SHA1: "ova"}
	m.files["rhel-10-amd64/rhel-10-amd64.ova"] = []byte("ova")

	err := artifactory.Generate(context.Background(), m, "ArtLib", "", true)
	if err == nil {
		t.Fatal("expected duplicate display name error")
	}
	if !strings.Contains(err.Error(), "duplicate display name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDirToItemEmptyListing(t *testing.T) {
	m := newMock()
	m.dirs["empty"] = nil
	items, err := artifactory.DirToItem(context.Background(), m, "empty", "empty", true, "urn:uuid:lib", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty map; got %v", items)
	}
}

func TestDirToItemLastModified(t *testing.T) {
	m := newMock()
	m.dirs["ovf"] = []artifactory.ChildItem{{URI: "/a.ovf", Folder: false}}
	m.meta["ovf/a.ovf"] = &artifactory.FileMeta{
		Size:         1,
		SHA1:         "s",
		LastModified: "2024-01-02T03:04:05.000-0000",
	}
	m.files["ovf/a.ovf"] = []byte("<Envelope></Envelope>")

	items, err := artifactory.DirToItem(context.Background(), m, "ovf", "ovf", true, "urn:uuid:x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if items["ovf"].Files[0].GenerationNum == 0 {
		t.Fatal("expected generation from LastModified")
	}
}
