// SPDX-License-Identifier: MIT

package hash

import (
	"crypto/md5"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"
)

const BlockSize = 1 << 20 // 1MB

// MD5File updates or creates an MD5 hash from a reader.
func MD5File(r io.Reader, existing hash.Hash) (hash.Hash, error) {
	if existing == nil {
		existing = md5.New()
	}
	buf := make([]byte, BlockSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			_, _ = existing.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return existing, nil
}

// MD5Folder calculates a combined MD5 for all files in a folder, excluding excludeName.
func MD5Folder(folderPath string, excludeName string) (string, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return "", err
	}
	var h hash.Hash
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == excludeName {
			continue
		}
		f, err := os.Open(filepath.Join(folderPath, entry.Name()))
		if err != nil {
			return "", err
		}
		h, err = MD5File(f, h)
		_ = f.Close()
		if err != nil {
			return "", err
		}
	}
	if h == nil {
		h = md5.New()
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
