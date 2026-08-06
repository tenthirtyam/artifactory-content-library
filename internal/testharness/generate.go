// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

// Package testharness provides filesystem VCSP metadata generation for unit tests.
// It is not part of the public CLI or configuration surface.
package testharness

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/tenthirtyam/artifactory-content-library/internal/hash"
	"github.com/tenthirtyam/artifactory-content-library/internal/logging"
	"github.com/tenthirtyam/artifactory-content-library/internal/security"
	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

// dirToItem converts a directory into a VCSP item.
// directory is the library-relative path (SelfHref parent), e.g. "debian-iso" or "iso/ubuntu/amd64".
func dirToItem(dirPath, directory string, md5Enabled bool, libID string) (vcsp.Item, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return vcsp.Item{}, err
	}

	var files []vcsp.FileInfo
	vcspType := vcsp.TypeOther
	folderMD5 := ""
	isVApp := "false"
	var isoCount int
	var isoBase string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fname := entry.Name()
		if fname == ".DS_Store" || fname == directory+".json" || fname == vcsp.ItemFile {
			continue
		}
		full := filepath.Join(dirPath, fname)
		ft := vcsp.TypeFromExt(fname)
		if ft != vcsp.TypeOther {
			vcspType = ft
			if ft == vcsp.TypeISO {
				isoCount++
				isoBase = strings.TrimSuffix(fname, filepath.Ext(fname))
			}
			if ft == vcsp.TypeOVF {
				data, err := os.ReadFile(full)
				if err == nil && strings.Contains(string(data), "<VirtualSystemCollection") {
					isVApp = "true"
				}
			}
		}
		if md5Enabled && folderMD5 == "" {
			folderMD5, err = hash.MD5Folder(dirPath, vcsp.ItemFile)
			if err != nil {
				logging.Warn("Could not calculate MD5 for folder", "path", dirPath, "error", err)
				folderMD5 = ""
			}
		}
		info, err := entry.Info()
		if err != nil {
			return vcsp.Item{}, err
		}
		href := directory + "/" + fname
		files = append(files, vcsp.FileInfo{
			Name:  fname,
			Size:  info.Size(),
			ETag:  folderMD5,
			Hrefs: []string{quotePreserveSlash(href)},
		})
	}

	displayName := directory
	if strings.Contains(directory, "/") {
		if vcspType == vcsp.TypeISO && isoCount == 1 {
			displayName = isoBase
		} else if vcspType == vcsp.TypeOVF {
			displayName = filepath.Base(directory)
		} else {
			displayName = strings.ReplaceAll(directory, "/", "-")
		}
	}

	return vcsp.MakeItem(directory, vcspType, displayName, files, vcsp.MakeItemOptions{
		Identifier:     uuid.New().String(),
		LibraryID:      libID,
		IsVAppTemplate: isVApp,
	}), nil
}

func quotePreserveSlash(s string) string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// Generate creates or updates a content library on the local filesystem.
func Generate(libName, libPath string, md5Enabled bool) error {
	var err error
	libName, err = security.SanitizeLibraryName(libName)
	if err != nil {
		return err
	}
	libPath, err = security.ValidatePath(libPath)
	if err != nil {
		return err
	}

	logging.Audit("Generating content library...", "library_name", libName, "path", libPath, "md5_enabled", md5Enabled)

	libJSONLoc := filepath.Join(libPath, vcsp.LibFile)
	itemsJSONLoc := filepath.Join(libPath, vcsp.ItemsFile)

	libID := uuid.New().String()
	libCreate := time.Now().UTC()
	libVersion := 1
	updating := false

	if data, err := os.ReadFile(libJSONLoc); err == nil {
		var oldLib vcsp.Library
		if json.Unmarshal(data, &oldLib) == nil {
			if oldLib.ID != "" {
				libID = vcsp.StripURN(oldLib.ID)
			}
			if oldLib.Created != "" {
				if t, err := vcsp.ParseCreated(oldLib.Created); err == nil {
					libCreate = t
				}
			}
			if oldLib.Version != "" {
				_, _ = fmt.Sscanf(oldLib.Version, "%d", &libVersion)
				updating = true
			}
		}
	}

	oldItems := map[string]vcsp.Item{}
	if data, err := os.ReadFile(itemsJSONLoc); err == nil {
		var coll vcsp.ItemsCollection
		if json.Unmarshal(data, &coll) == nil {
			for _, it := range coll.Items {
				key := it.Name
				if it.SelfHref != "" {
					href := it.SelfHref
					if dir, ok := strings.CutSuffix(href, "/"+vcsp.ItemFile); ok {
						key = dir
					}
				}
				oldItems[key] = it
			}
		}
	}

	var contentDirs []string
	var walk func(abs, rel string) error
	walk = func(abs, rel string) error {
		ents, err := os.ReadDir(abs)
		if err != nil {
			return err
		}
		var subdirs []os.DirEntry
		hasFiles := false
		for _, e := range ents {
			if e.IsDir() {
				subdirs = append(subdirs, e)
				continue
			}
			name := e.Name()
			if name == vcsp.LibFile || name == vcsp.ItemsFile || name == vcsp.ItemFile || name == ".DS_Store" {
				continue
			}
			hasFiles = true
		}
		if hasFiles {
			if rel == "" {
				for _, d := range subdirs {
					if err := walk(filepath.Join(abs, d.Name()), d.Name()); err != nil {
						return err
					}
				}
				return nil
			}
			contentDirs = append(contentDirs, rel)
			return nil
		}
		for _, d := range subdirs {
			childRel := d.Name()
			if rel != "" {
				childRel = filepath.ToSlash(filepath.Join(rel, d.Name()))
			}
			if err := walk(filepath.Join(abs, d.Name()), childRel); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(libPath, ""); err != nil {
		return err
	}

	var items []vcsp.Item
	changed := false

	for _, itemPath := range contentDirs {
		p := filepath.Join(libPath, filepath.FromSlash(itemPath))
		itemJSON, err := dirToItem(p, itemPath, md5Enabled, vcsp.NormalizeItemID(libID))
		if err != nil {
			return err
		}

		storageKey := itemPath
		if _, exists := oldItems[storageKey]; !exists && updating {
			changed = true
		} else if old, exists := oldItems[storageKey]; exists {
			itemJSON.ID = old.ID
			itemJSON.Created = old.Created
			itemJSON.Version = old.Version
			fileChanged := false
			newNames := fileNameSet(itemJSON.Files)
			oldNames := fileNameSet(old.Files)
			if !stringSetEqual(newNames, oldNames) {
				changed = true
				fileChanged = true
			}
			if !fileChanged {
				for _, f := range itemJSON.Files {
					for _, of := range old.Files {
						if f.Name == of.Name && f.ETag != of.ETag {
							changed = true
							fileChanged = true
							break
						}
					}
					if fileChanged {
						break
					}
				}
			}
			if fileChanged {
				v := 0
				_, _ = fmt.Sscanf(itemJSON.Version, "%d", &v)
				itemJSON.Version = fmt.Sprintf("%d", v+1)
			}
			delete(oldItems, storageKey)
		}

		outPath := filepath.Join(p, vcsp.ItemFile)
		if err := writeJSON(outPath, itemJSON); err != nil {
			return err
		}
		items = append(items, itemJSON)
	}

	if updating && len(oldItems) > 0 {
		changed = true
	}
	if updating && !changed {
		logging.Info("Nothing to update.", "status", "up_to_date")
		return nil
	}
	if changed {
		libVersion++
	}

	created := libCreate
	lib := vcsp.MakeLib(libName, libID, created, libVersion)
	if err := writeJSON(libJSONLoc, lib); err != nil {
		return err
	}
	return writeJSON(itemsJSONLoc, vcsp.MakeItems(items))
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func fileNameSet(files []vcsp.FileInfo) map[string]struct{} {
	m := make(map[string]struct{}, len(files))
	for _, f := range files {
		m[f.Name] = struct{}{}
	}
	return m
}

func stringSetEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}
