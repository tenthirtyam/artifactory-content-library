// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package artifactory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/tenthirtyam/artifactory-content-library/internal/logging"
	"github.com/tenthirtyam/artifactory-content-library/internal/security"
	"github.com/tenthirtyam/artifactory-content-library/internal/vcsp"
)

type fileEntry struct {
	itemPath string
	fileName string
}

// contentFolder is a directory that contains content files and becomes one library item.
type contentFolder struct {
	// relPath is relative to the library base path (SelfHref parent), e.g. "debian-iso" or
	// "iso/ubuntu/ubuntu-26.04/amd64".
	relPath string
	// absPath is the Artifactory path including the base path prefix.
	absPath string
}

// DirToItem builds item JSON structures for an Artifactory directory.
// itemRelPath is the path relative to the library base (used for SelfHref and file hrefs).
func DirToItem(ctx context.Context, client StorageClient, dirPath, itemRelPath string, skipCert bool, libID string, oldItem *vcsp.Item) (map[string]vcsp.Item, error) {
	itemsJSON := map[string]vcsp.Item{}
	var filesItems []vcsp.FileInfo
	vcspType := vcsp.TypeOther
	isVApp := "false"

	items, err := client.ListItems(ctx, dirPath)
	if err != nil || len(items) == 0 {
		logging.Warn("No items found in Artifactory path", "path", dirPath)
		return itemsJSON, nil
	}

	var ovfFiles, isoFiles, otherFiles []fileEntry

	for _, item := range items {
		if item.Folder {
			continue
		}
		fileName := strings.TrimLeft(item.URI, "/")
		itemPath := fileName
		if dirPath != "" {
			itemPath = strings.TrimRight(dirPath, "/") + "/" + fileName
		}
		if fileName == vcsp.LibFile || fileName == vcsp.ItemsFile || fileName == vcsp.ItemFile {
			continue
		}
		switch vcsp.TypeFromExt(fileName) {
		case vcsp.TypeOVF:
			ovfFiles = append(ovfFiles, fileEntry{itemPath, fileName})
			vcspType = vcsp.TypeOVF
		case vcsp.TypeISO:
			isoFiles = append(isoFiles, fileEntry{itemPath, fileName})
			if vcspType == vcsp.TypeOther {
				vcspType = vcsp.TypeISO
			}
		default:
			otherFiles = append(otherFiles, fileEntry{itemPath, fileName})
		}
	}

	for _, fe := range ovfFiles {
		if oldItem != nil {
			b, _ := json.Marshal(oldItem)
			if strings.Contains(string(b), "type-metadata") {
				continue
			}
		}
		content, err := client.Download(ctx, fe.itemPath)
		if err != nil {
			logging.Error("Failed to download OVF descriptor", "item_path", fe.itemPath, "error", err)
			continue
		}
		if strings.Contains(string(content), "<VirtualSystemCollection") {
			isVApp = "true"
			break
		}
	}

	allFiles := append(append(ovfFiles, isoFiles...), otherFiles...)
	for _, fe := range allFiles {
		if vcspType == vcsp.TypeOVF && strings.HasSuffix(fe.fileName, vcsp.FileExtCert) && skipCert {
			continue
		}
		meta, err := client.GetFileInfo(ctx, fe.itemPath)
		if err != nil || meta == nil {
			logging.Warn("Could not get file info", "item_path", fe.itemPath)
			continue
		}
		etag := meta.SHA1
		if etag == "" {
			etag = meta.MD5
		}
		var generationNum int64
		if meta.LastModified != "" {
			if t, err := time.Parse(time.RFC3339, strings.Replace(meta.LastModified, "Z", "+00:00", 1)); err == nil {
				generationNum = t.Unix()
			} else if t, err := time.Parse("2006-01-02T15:04:05.999-0700", meta.LastModified); err == nil {
				generationNum = t.Unix()
			}
		}
		filesItems = append(filesItems, vcsp.FileInfo{
			Name:          fe.fileName,
			Size:          meta.Size,
			ETag:          etag,
			GenerationNum: generationNum,
			Hrefs:         []string{itemRelPath + "/" + fe.fileName},
		})
	}

	if len(filesItems) > 0 {
		id := uuid.New().String()
		if oldItem != nil && oldItem.ID != "" {
			id = oldItem.ID
		}
		displayName := itemDisplayName(itemRelPath, vcspType, isoFiles, ovfFiles)
		item := vcsp.MakeItem(itemRelPath, vcspType, displayName, filesItems, vcsp.MakeItemOptions{
			Identifier:     id,
			LibraryID:      libID,
			IsVAppTemplate: isVApp,
		})
		if vcspType == vcsp.TypeOVF && oldItem != nil && len(oldItem.Metadata) > 0 {
			item.Metadata = oldItem.Metadata
		}
		itemsJSON[itemRelPath] = item
	}
	return itemsJSON, nil
}

func itemDisplayName(itemRelPath string, vcspType vcsp.Type, isoFiles, ovfFiles []fileEntry) string {
	// Flat layouts keep the folder name for stable display (e.g. debian-iso).
	if !strings.Contains(itemRelPath, "/") {
		return itemRelPath
	}
	// Nested single-ISO leaf: use the ISO basename (e.g. ubuntu-26.04-amd64).
	if vcspType == vcsp.TypeISO && len(isoFiles) == 1 && len(ovfFiles) == 0 {
		return strings.TrimSuffix(isoFiles[0].fileName, path.Ext(isoFiles[0].fileName))
	}
	if vcspType == vcsp.TypeOVF {
		return path.Base(itemRelPath)
	}
	return strings.ReplaceAll(itemRelPath, "/", "-")
}

func joinArtifactoryPath(parts ...string) string {
	var b []string
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p != "" {
			b = append(b, p)
		}
	}
	return strings.Join(b, "/")
}

func isMetadataFile(name string) bool {
	return name == vcsp.LibFile || name == vcsp.ItemsFile || name == vcsp.ItemFile
}

// discoverContentFolders walks container folders and returns leaf folders that hold content files.
// A folder with content files is an item (nested subfolders there are not walked).
// A folder with only subfolders is a container and is recursed into.
// Files directly under the library base path are ignored.
func discoverContentFolders(ctx context.Context, client StorageClient, basePath string) ([]contentFolder, error) {
	var out []contentFolder

	var walk func(relPath string) error
	walk = func(relPath string) error {
		absPath := joinArtifactoryPath(basePath, relPath)
		children, err := client.ListItems(ctx, absPath)
		if err != nil {
			return err
		}

		var folders []ChildItem
		hasFiles := false
		for _, child := range children {
			name := strings.TrimLeft(child.URI, "/")
			if name == "" {
				continue
			}
			if child.Folder {
				folders = append(folders, child)
				continue
			}
			if isMetadataFile(name) {
				continue
			}
			hasFiles = true
		}

		if hasFiles {
			if relPath == "" {
				// Loose files at the library root are ignored; still walk sibling folders.
				for _, folder := range folders {
					childRel := joinArtifactoryPath(relPath, strings.TrimLeft(folder.URI, "/"))
					if err := walk(childRel); err != nil {
						return err
					}
				}
				return nil
			}
			out = append(out, contentFolder{relPath: relPath, absPath: absPath})
			return nil
		}

		for _, folder := range folders {
			childRel := joinArtifactoryPath(relPath, strings.TrimLeft(folder.URI, "/"))
			if err := walk(childRel); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(""); err != nil {
		return nil, err
	}
	return out, nil
}

func unquotePath(s string) string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		if u, err := url.PathUnescape(p); err == nil {
			parts[i] = u
		}
	}
	return strings.Join(parts, "/")
}

// itemStoragePath returns the library-relative directory for an item (SelfHref parent).
func itemStoragePath(it vcsp.Item) string {
	if it.SelfHref != "" {
		href := unquotePath(it.SelfHref)
		if dir, ok := strings.CutSuffix(href, "/"+vcsp.ItemFile); ok {
			return dir
		}
		return path.Dir(href)
	}
	return it.Name
}

// Generate creates or updates Artifactory content library metadata.
func Generate(ctx context.Context, client StorageClient, libName, basePath string, skipCert bool) error {
	var err error
	libName, err = security.SanitizeLibraryName(libName)
	if err != nil {
		return err
	}
	basePath = strings.Trim(basePath, "/")
	if strings.Contains(basePath, "..") || strings.Contains(basePath, "\x00") {
		return security.NewError("base path contains invalid characters")
	}

	logging.Audit("Starting content library generation...",
		"library_name", libName, "base_path", basePath, "skip_cert", skipCert)

	libJSONPath := vcsp.LibFile
	itemsJSONPath := vcsp.ItemsFile
	if basePath != "" {
		libJSONPath = basePath + "/" + vcsp.LibFile
		itemsJSONPath = basePath + "/" + vcsp.ItemsFile
	}

	libID := uuid.New().String()
	libCreate := time.Now().UTC()
	libVersion := 1
	updating := false

	if ok, _ := client.FileExists(ctx, libJSONPath); ok {
		if content, err := client.Download(ctx, libJSONPath); err == nil {
			var oldLib vcsp.Library
			if json.Unmarshal(content, &oldLib) == nil {
				if oldLib.ID != "" {
					libID = vcsp.StripURN(oldLib.ID)
				}
				if t, err := vcsp.ParseCreated(oldLib.Created); err == nil {
					libCreate = t
				}
				if oldLib.Version != "" {
					_, _ = fmt.Sscanf(oldLib.Version, "%d", &libVersion)
					updating = true
				}
			}
		}
	}

	oldItems := map[string]vcsp.Item{}
	if ok, _ := client.FileExists(ctx, itemsJSONPath); ok {
		if content, err := client.Download(ctx, itemsJSONPath); err == nil {
			var coll vcsp.ItemsCollection
			if json.Unmarshal(content, &coll) == nil {
				for _, it := range coll.Items {
					oldItems[itemStoragePath(it)] = it
				}
			}
		}
	}

	folders, err := discoverContentFolders(ctx, client, basePath)
	if err != nil {
		logging.Error("Failed to discover content folders in Artifactory path", "path", basePath, "error", err)
		return err
	}
	if len(folders) == 0 {
		logging.Info("No content folders found in the specified path", "path", basePath)
		return nil
	}

	logging.Info(fmt.Sprintf("Processing %d content folders...", len(folders)))

	var items []vcsp.Item
	changed := false
	updateItemsJSON := false

	for _, folder := range folders {
		var oldPtr *vcsp.Item
		if old, ok := oldItems[folder.relPath]; ok {
			oldPtr = &old
		}
		generated, err := DirToItem(ctx, client, folder.absPath, folder.relPath, skipCert, vcsp.NormalizeItemID(libID), oldPtr)
		if err != nil {
			logging.Error("Error processing directory", "directory", folder.relPath, "error", err)
			continue
		}
		for itemRelPath, itemJSON := range generated {
			itemJSON.ContentVersion = "2"
			if _, exists := oldItems[itemRelPath]; !exists && updating {
				changed = true
			} else if old, exists := oldItems[itemRelPath]; exists {
				fileChanged := false
				itemJSON.ID = old.ID
				itemJSON.Created = old.Created
				itemJSON.Version = old.Version
				if old.ContentVersion != "" {
					itemJSON.ContentVersion = old.ContentVersion
				} else {
					changed = true
				}
				oldStr := fmt.Sprintf("%v", old)
				newStr := fmt.Sprintf("%v", itemJSON)
				if !strings.Contains(oldStr, "type-metadata") && strings.Contains(newStr, "type-metadata") {
					updateItemsJSON = true
				}
				if !fileNamesEqual(itemJSON.Files, old.Files) {
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
					v, cv := 0, 0
					_, _ = fmt.Sscanf(itemJSON.Version, "%d", &v)
					_, _ = fmt.Sscanf(itemJSON.ContentVersion, "%d", &cv)
					itemJSON.Version = fmt.Sprintf("%d", v+1)
					itemJSON.ContentVersion = fmt.Sprintf("%d", cv+1)
				}
				delete(oldItems, itemRelPath)
			}

			itemJSONPath := unquotePath(itemJSON.SelfHref)
			if basePath != "" {
				itemJSONPath = basePath + "/" + itemJSONPath
			}
			data, err := MarshalIndent(itemJSON)
			if err != nil {
				return err
			}
			if err := client.Upload(ctx, itemJSONPath, data, "application/json; charset=utf-8"); err != nil {
				logging.Error("Failed to upload item metadata", "path", itemJSONPath)
			}
			items = append(items, itemJSON)
		}
	}

	seenNames := map[string]string{}
	for _, it := range items {
		if prev, ok := seenNames[it.Name]; ok {
			return fmt.Errorf("duplicate display name %q for items %q and %q; vSphere requires unique item names within a library",
				it.Name, prev, itemStoragePath(it))
		}
		seenNames[it.Name] = itemStoragePath(it)
	}

	if updating && len(oldItems) > 0 {
		changed = true
		logging.Info("Removing orphaned items from content library..", "removed_count", len(oldItems))
		for _, old := range oldItems {
			orphaned := old.SelfHref
			if orphaned == "" && old.Type == vcsp.TypeISO {
				// Legacy nested ISO items stored item.json under folder/basename.
				orphaned = path.Join(old.Name, vcsp.ItemFile)
			}
			if orphaned == "" {
				continue
			}
			// Use SelfHref as stored (may include encoded path segments from legacy items).
			if basePath != "" {
				orphaned = basePath + "/" + orphaned
			}
			if ok, _ := client.FileExists(ctx, orphaned); ok {
				_ = client.Delete(ctx, orphaned)
			}
		}
	}

	if updating && !changed {
		logging.Info("Content library is already up to date.", "status", "up_to_date")
		if updateItemsJSON {
			data, _ := MarshalIndent(vcsp.MakeItems(items))
			_ = client.Upload(ctx, itemsJSONPath, data, "application/json; charset=utf-8")
		}
		return nil
	}
	if changed {
		libVersion++
	}

	logging.Info("Saving content library metadata...", "lib_file", libJSONPath, "items_file", itemsJSONPath)
	libData, err := MarshalIndent(vcsp.MakeLib(libName, libID, libCreate, libVersion))
	if err != nil {
		return err
	}
	itemsData, err := MarshalIndent(vcsp.MakeItems(items))
	if err != nil {
		return err
	}
	libOK := client.Upload(ctx, libJSONPath, libData, "application/json; charset=utf-8") == nil
	itemsOK := client.Upload(ctx, itemsJSONPath, itemsData, "application/json; charset=utf-8") == nil
	if libOK && itemsOK {
		logging.Info("Successfully created/updated content library.", "library_name", libName, "status", "success")
		return nil
	}
	return fmt.Errorf("failed to upload some content library metadata files")
}

func fileNamesEqual(a, b []vcsp.FileInfo) bool {
	if len(a) != len(b) {
		return false
	}
	ma := map[string]struct{}{}
	for _, f := range a {
		ma[f.Name] = struct{}{}
	}
	for _, f := range b {
		if _, ok := ma[f.Name]; !ok {
			return false
		}
	}
	return true
}
