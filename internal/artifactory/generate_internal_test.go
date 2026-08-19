// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package artifactory

import (
	"strings"
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

func TestFormatChangeSummaryDryRunCreate(t *testing.T) {
	lines := formatChangeSummary("ArtLib", false, false, true, 1, []ItemChange{
		{Action: ChangeAdd, Path: "debian-iso", Name: "debian-iso", Reason: ReasonNew},
	})
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `Would create content library "ArtLib"`) {
		t.Fatalf("missing create header: %q", got)
	}
	if !strings.Contains(got, "add     debian-iso") {
		t.Fatalf("missing add line: %q", got)
	}
	if !strings.Contains(got, "write   lib.json, items.json") {
		t.Fatalf("missing write line: %q", got)
	}
}

func TestFormatChangeSummaryDryRunUpdate(t *testing.T) {
	lines := formatChangeSummary("ArtLib", true, true, true, 2, []ItemChange{
		{Action: ChangeAdd, Path: "rhel-iso", Name: "rhel-iso", Reason: ReasonNew},
		{Action: ChangeUpdate, Path: "ubuntu", Name: "ubuntu-26.04-amd64", Reason: ReasonChecksum},
		{Action: ChangeRemove, Path: "old-debian", Name: "old-debian", Reason: ReasonOrphan},
	})
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `Would update content library "ArtLib" (lib.json 2 -> 3)`) {
		t.Fatalf("missing update header: %q", got)
	}
	if !strings.Contains(got, "change  ubuntu (checksum)") {
		t.Fatalf("missing change line: %q", got)
	}
	if !strings.Contains(got, "remove  old-debian") {
		t.Fatalf("missing remove line: %q", got)
	}
}

func TestFormatChangeSummaryNoChange(t *testing.T) {
	dry := formatChangeSummary("ArtLib", true, false, true, 2, nil)
	if len(dry) != 1 || dry[0] != "No JSON metadata would change. Content library is already up to date." {
		t.Fatalf("dry-run no-op: %v", dry)
	}
	live := formatChangeSummary("ArtLib", true, false, false, 2, nil)
	if len(live) != 1 || live[0] != "Content library is already up to date." {
		t.Fatalf("live no-op: %v", live)
	}
}

func TestFormatChangeSummaryLiveCreate(t *testing.T) {
	lines := formatChangeSummary("ArtLib", false, false, false, 1, []ItemChange{
		{Action: ChangeAdd, Path: "iso/ubuntu/ubuntu-26.04/amd64", Name: "ubuntu-26.04-amd64", Reason: ReasonNew},
	})
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `Created content library "ArtLib"`) {
		t.Fatalf("missing created header: %q", got)
	}
}
