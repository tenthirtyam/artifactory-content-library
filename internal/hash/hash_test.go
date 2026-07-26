// SPDX-License-Identifier: MIT
package hash_test

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/tenthirtyam/artifactory-content-library/internal/hash"
	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

func TestMD5File(t *testing.T) {
	content := []byte("Hello, World!")
	expected := md5.Sum(content)
	h, err := hash.MD5File(bytes.NewReader(content), nil)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(h.Sum(nil)) != hex.EncodeToString(expected[:]) {
		t.Fatal("md5 mismatch")
	}
}

func TestMD5FileContinues(t *testing.T) {
	h, err := hash.MD5File(bytes.NewReader([]byte("Hello, ")), nil)
	if err != nil {
		t.Fatal(err)
	}
	h, err = hash.MD5File(bytes.NewReader([]byte("World!")), h)
	if err != nil {
		t.Fatal(err)
	}
	expected := md5.Sum([]byte("Hello, World!"))
	if hex.EncodeToString(h.Sum(nil)) != hex.EncodeToString(expected[:]) {
		t.Fatal("continued md5 mismatch")
	}
}

func TestMD5Folder(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, vcsp.ItemFile), []byte("{}"), 0o600)
	sum, err := hash.MD5Folder(dir, vcsp.ItemFile)
	if err != nil {
		t.Fatal(err)
	}
	if sum == "" {
		t.Fatal("empty sum")
	}
	empty := t.TempDir()
	sum2, err := hash.MD5Folder(empty, vcsp.ItemFile)
	if err != nil {
		t.Fatal(err)
	}
	if sum2 != hex.EncodeToString(md5.New().Sum(nil)) {
		t.Fatalf("empty folder hash %q", sum2)
	}
}
