// SPDX-License-Identifier: MIT
package artifactory

import (
	"testing"

	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

func TestFileNamesEqual(t *testing.T) {
	a := []vcsp.FileInfo{{Name: "a.iso"}, {Name: "b.ovf"}}
	b := []vcsp.FileInfo{{Name: "b.ovf"}, {Name: "a.iso"}}
	if !fileNamesEqual(a, b) {
		t.Fatal("expected equal ignoring order")
	}
	if fileNamesEqual(a, []vcsp.FileInfo{{Name: "a.iso"}}) {
		t.Fatal("expected unequal lengths")
	}
	if fileNamesEqual(a, []vcsp.FileInfo{{Name: "a.iso"}, {Name: "c.vmdk"}}) {
		t.Fatal("expected different names")
	}
}

func TestMarshalIndent(t *testing.T) {
	data, err := MarshalIndent(map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatalf("expected trailing newline: %q", data)
	}
}
