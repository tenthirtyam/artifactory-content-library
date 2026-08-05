// SPDX-License-Identifier: MIT

package vcsp_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

func TestMakeLibDefaults(t *testing.T) {
	lib := vcsp.MakeLib("demo", "", time.Time{}, 0)
	if lib.Name != "demo" || lib.VcspVersion != "2" || lib.ItemsHref != vcsp.ItemsFile {
		t.Fatalf("%+v", lib)
	}
	if !strings.HasPrefix(lib.ID, "urn:uuid:") {
		t.Fatalf("id %q", lib.ID)
	}
}

func TestNormalizeItemID(t *testing.T) {
	if vcsp.NormalizeItemID("urn:uuid:abc") != "urn:uuid:abc" {
		t.Fatal()
	}
	if vcsp.NormalizeItemID("abc") != "urn:uuid:abc" {
		t.Fatal()
	}
}

func TestMakeItemOVF(t *testing.T) {
	item := vcsp.MakeItem("dir", vcsp.TypeOVF, "dir", []vcsp.FileInfo{{Name: "x.ovf", Size: 1, Hrefs: []string{"dir/x.ovf"}}}, vcsp.MakeItemOptions{
		LibraryID: "urn:uuid:lib",
	})
	if item.Type != vcsp.TypeOVF || len(item.Metadata) != 1 {
		t.Fatalf("%+v", item)
	}
}

func TestMakeItemPreservesPathSeparators(t *testing.T) {
	item := vcsp.MakeItem("parent/child", vcsp.TypeISO, "child", nil, vcsp.MakeItemOptions{})
	if item.SelfHref != "parent/child/item.json" {
		t.Fatalf("selfHref %q", item.SelfHref)
	}
}

func TestTypeFromExt(t *testing.T) {
	if vcsp.TypeFromExt("a.ovf") != vcsp.TypeOVF {
		t.Fatal()
	}
	if vcsp.TypeFromExt("a.ova") != vcsp.TypeOVF {
		t.Fatal()
	}
	if vcsp.TypeFromExt("a.iso") != vcsp.TypeISO {
		t.Fatal()
	}
	if vcsp.TypeFromExt("a.txt") != vcsp.TypeOther {
		t.Fatal()
	}
}

func TestMakeItems(t *testing.T) {
	coll := vcsp.MakeItems([]vcsp.Item{{Name: "a"}})
	if len(coll.Items) != 1 {
		t.Fatal()
	}
}
