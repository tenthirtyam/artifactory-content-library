// SPDX-License-Identifier: MIT

package vcsp

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	vcspVersion    = 2
	dateTimeFormat = "2006-01-02T15:04Z"
	LibFile        = "lib.json"
	ItemsFile      = "items.json"
	ItemFile       = "item.json"
	maxNameLength  = 80
	FileExtCert    = ".cert"
)

// Type is a VCSP content type.
type Type string

const (
	TypeOVF   Type = "vcsp.ovf"
	TypeISO   Type = "vcsp.iso"
	TypeOther Type = "vcsp.other"
)

// FileInfo describes a file in an item.
type FileInfo struct {
	Name          string   `json:"name"`
	Size          int64    `json:"size"`
	ETag          string   `json:"etag,omitempty"`
	GenerationNum int64    `json:"generationNum,omitempty"`
	Hrefs         []string `json:"hrefs"`
}

// metadataEntry is OVF type-metadata.
type metadataEntry struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Type       string `json:"type"`
	Domain     string `json:"domain"`
	Visibility string `json:"visibility"`
}

// Item is a content library item.
type Item struct {
	Created        string          `json:"created"`
	Description    string          `json:"description"`
	Version        string          `json:"version"`
	ContentVersion string          `json:"contentVersion,omitempty"`
	Files          []FileInfo      `json:"files"`
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Properties     map[string]any  `json:"properties"`
	SelfHref       string          `json:"selfHref"`
	Type           Type            `json:"type"`
	Metadata       []metadataEntry `json:"metadata,omitempty"`
}

// Library is the lib.json structure.
type Library struct {
	VcspVersion    string         `json:"vcspVersion"`
	Version        string         `json:"version"`
	ContentVersion string         `json:"contentVersion"`
	Name           string         `json:"name"`
	ID             string         `json:"id"`
	Created        string         `json:"created"`
	Capabilities   map[string]any `json:"capabilities"`
	ItemsHref      string         `json:"itemsHref"`
}

// ItemsCollection is items.json.
type ItemsCollection struct {
	Items []Item `json:"items"`
}

// MakeLib creates a library descriptor.
func MakeLib(name string, libID string, creation time.Time, version int) Library {
	if libID == "" {
		libID = uuid.New().String()
	}
	if creation.IsZero() {
		creation = time.Now().UTC()
	}
	if version <= 0 {
		version = 1
	}
	return Library{
		VcspVersion:    fmt.Sprintf("%d", vcspVersion),
		Version:        fmt.Sprintf("%d", version),
		ContentVersion: "1",
		Name:           name,
		ID:             NormalizeItemID(libID),
		Created:        creation.UTC().Format(dateTimeFormat),
		Capabilities: map[string]any{
			"transferIn":  []string{"httpGet"},
			"transferOut": []string{"httpGet"},
		},
		ItemsHref: ItemsFile,
	}
}

// truncateName truncates a name preserving extension.
func truncateName(name string) string {
	if len(name) <= maxNameLength {
		return name
	}
	if idx := strings.LastIndex(name, "."); idx > 0 {
		ext := name[idx+1:]
		baseLen := maxNameLength - len(ext) - 1
		if baseLen < 1 {
			return name[:maxNameLength]
		}
		return name[:baseLen] + "." + ext
	}
	return name[:maxNameLength]
}

// NormalizeItemID ensures urn:uuid: prefix.
func NormalizeItemID(id string) string {
	s := id
	if strings.HasPrefix(s, "urn:uuid:") {
		return s
	}
	return "urn:uuid:" + s
}

// createOVFMetadata builds OVF type-metadata.
func createOVFMetadata(itemID string, version int, libraryID, isVAppTemplate string) metadataEntry {
	value := fmt.Sprintf(
		`{"id":"%s","version":"%d","libraryIdParent":"%s","isVappTemplate":"%s","vmTemplate":null,"vappTemplate":null,"networks":[],"storagePolicyGroups":null}`,
		itemID, version, libraryID, isVAppTemplate,
	)
	return metadataEntry{
		Key:        "type-metadata",
		Value:      value,
		Type:       "String",
		Domain:     "SYSTEM",
		Visibility: "READONLY",
	}
}

// MakeItemOptions holds optional MakeItem parameters.
type MakeItemOptions struct {
	Description    string
	Properties     map[string]any
	Identifier     string
	Creation       time.Time
	Version        int
	LibraryID      string
	IsVAppTemplate string
}

// MakeItem creates an item structure.
func MakeItem(directory string, vcspType Type, name string, files []FileInfo, opts MakeItemOptions) Item {
	if opts.Identifier == "" {
		opts.Identifier = uuid.New().String()
	}
	if opts.Creation.IsZero() {
		opts.Creation = time.Now().UTC()
	}
	if opts.Version <= 0 {
		opts.Version = 2
	}
	if opts.Properties == nil {
		opts.Properties = map[string]any{}
	}
	if opts.IsVAppTemplate == "" {
		opts.IsVAppTemplate = "false"
	}
	itemID := NormalizeItemID(opts.Identifier)
	item := Item{
		Created:     opts.Creation.UTC().Format(dateTimeFormat),
		Description: opts.Description,
		Version:     fmt.Sprintf("%d", opts.Version),
		Files:       files,
		ID:          itemID,
		Name:        truncateName(name),
		Properties:  opts.Properties,
		SelfHref:    quotePath(directory) + "/" + quotePath(ItemFile),
		Type:        vcspType,
	}

	if vcspType == TypeOVF {
		item.Metadata = []metadataEntry{
			createOVFMetadata(itemID, opts.Version, opts.LibraryID, opts.IsVAppTemplate),
		}
	}
	return item
}

// quotePath percent-encodes each path segment while preserving '/'.
func quotePath(s string) string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// MakeItems creates an items collection.
func MakeItems(items []Item) ItemsCollection {
	return ItemsCollection{Items: items}
}

// TypeFromExt determines VCSP type from file extension.
func TypeFromExt(filename string) Type {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".ovf", ".ova":
		return TypeOVF
	case ".iso":
		return TypeISO
	default:
		return TypeOther
	}
}

// StripURN returns the UUID portion of an urn:uuid: ID.
func StripURN(id string) string {
	return strings.TrimPrefix(id, "urn:uuid:")
}

// ParseCreated parses a VCSP datetime.
func ParseCreated(s string) (time.Time, error) {
	return time.Parse(dateTimeFormat, s)
}
