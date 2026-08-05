// SPDX-License-Identifier: MIT

package security_test

import (
	"strings"
	"testing"

	"github.com/tenthirtyam/artifactory-content-library/internal/security"
)

func TestValidatePathAcceptsValid(t *testing.T) {
	dir := t.TempDir()
	got, err := security.ValidatePath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected normalized path")
	}
}

func TestValidatePathRejectsInvalid(t *testing.T) {
	if _, err := security.ValidatePath(""); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := security.ValidatePath("../../../etc/passwd"); err == nil {
		t.Fatal("expected traversal error")
	}
	long := strings.Repeat("a", security.MaxPathLength+1)
	if _, err := security.ValidatePath(long); err == nil {
		t.Fatal("expected length error")
	}
	if _, err := security.ValidatePath("test\x00path"); err == nil {
		t.Fatal("expected control char error")
	}
}

func TestSanitizeLibraryName(t *testing.T) {
	got, err := security.SanitizeLibraryName("  My Library  ")
	if err != nil || got != "My Library" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := security.SanitizeLibraryName(""); err == nil {
		t.Fatal("expected empty name error")
	}
	if _, err := security.SanitizeLibraryName(strings.Repeat("a", security.MaxLibraryNameLength+1)); err == nil {
		t.Fatal("expected length error")
	}
	if _, err := security.SanitizeLibraryName("Library<script>alert('xss')</script>"); err == nil {
		t.Fatal("expected script error")
	}
	if _, err := security.SanitizeLibraryName("javascript:alert('xss')"); err == nil {
		t.Fatal("expected javascript error")
	}
}

func TestMaskSensitive(t *testing.T) {
	short := security.MaskSensitive("abc", 4)
	if len(short) < 8 || strings.ContainsAny(short, "abc") {
		t.Fatalf("unexpected short mask %q", short)
	}
	med := security.MaskSensitive("password123", 4)
	if med != "pass***d123" {
		t.Fatalf("got %q", med)
	}
	long := security.MaskSensitive("very-long-api-key-here", 4)
	if !strings.HasPrefix(long, "very") || !strings.HasSuffix(long, "here") || !strings.Contains(long, "*") {
		t.Fatalf("got %q", long)
	}
}

func TestValidateArtifactoryURL(t *testing.T) {
	got, err := security.ValidateArtifactoryURL("https://packages.example.com/artifactory/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://packages.example.com/artifactory" {
		t.Fatalf("got %q", got)
	}
	if _, err := security.ValidateArtifactoryURL(""); err == nil {
		t.Fatal("expected empty url error")
	}
	if _, err := security.ValidateArtifactoryURL("ftp://x"); err == nil {
		t.Fatal("expected scheme error")
	}
}
