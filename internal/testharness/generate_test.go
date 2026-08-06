// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package testharness_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tenthirtyam/artifactory-content-library/internal/testharness"
	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

func TestGenerateCreatesLibrary(t *testing.T) {
	root := t.TempDir()
	item := filepath.Join(root, "debian-iso")
	if err := os.Mkdir(item, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(item, "debian.iso"), []byte("iso"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := testharness.Generate("Test Library", root, true); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, vcsp.LibFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, vcsp.ItemsFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(item, vcsp.ItemFile)); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(root, vcsp.LibFile))
	var lib vcsp.Library
	if err := json.Unmarshal(data, &lib); err != nil {
		t.Fatal(err)
	}
	if lib.Name != "Test Library" {
		t.Fatalf("%+v", lib)
	}
}

func TestGenerateSkipsWhenUnchanged(t *testing.T) {
	root := t.TempDir()
	item := filepath.Join(root, "item1")
	_ = os.Mkdir(item, 0o755)
	_ = os.WriteFile(filepath.Join(item, "file.iso"), []byte("x"), 0o644)

	if err := testharness.Generate("Lib", root, true); err != nil {
		t.Fatal(err)
	}
	lib1, _ := os.ReadFile(filepath.Join(root, vcsp.LibFile))
	if err := testharness.Generate("Lib", root, true); err != nil {
		t.Fatal(err)
	}
	lib2, _ := os.ReadFile(filepath.Join(root, vcsp.LibFile))
	if string(lib1) != string(lib2) {
		t.Fatal("library should be unchanged")
	}
}

func TestGenerateRejectsBadName(t *testing.T) {
	if err := testharness.Generate("", t.TempDir(), false); err == nil {
		t.Fatal("expected error")
	}
}
