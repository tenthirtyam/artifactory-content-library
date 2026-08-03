---
icon: octicons/file-directory-16
---

# Content Layout

`generate` does **not** upload OVF/OVA/ISO content. You place content in Artifactory
first; `generate` then walks that tree and writes vSphere content library metadata
(`lib.json`, `items.json`, and per-item `item.json`).

## Model

**One content library item = one leaf folder that contains content files.**

- Folders that contain **files** (`.iso`, `.ovf`, `.ova`, disks, …) become items.
- Folders that contain **only subfolders** are containers and are walked recursively.
- Nested subfolders inside a content folder are **not** walked (OVF packages stay intact).

```text
{repo}/{path}/                 ← library root (base path)
├── lib.json                   ← written by generate
├── items.json                 ← written by generate
├── debian-iso/
│   ├── debian.iso             ← item (display name: debian-iso)
│   └── item.json              ← written by generate
├── debian-ova/                ← item (display name: debian-ova)
│   ├── debian.ova
│   └── item.json              ← written by generate
└── debian-ovf/                ← item (display name: debian-ovf)
    ├── debian.ovf
    ├── debian-disk-0.vmdk
    ├── debian.iso             ← sidecar file on the OVF item
    ├── debian.mf
    ├── debian.nvram
    ├── debian.cert            ← skipped when --skip-cert is true (default)
    └── item.json              ← written by generate
```

Nested structures also work. After `generate`:

??? example "Example: Nested Structure"

    ```text
    {path}/
    ├── lib.json
    ├── items.json
    └── iso/                                   ← container
        ├── ubuntu/                            ← container
        │   ├── ubuntu-26.04/                  ← container
        │   │   ├── amd64/
        │   │   │   ├── ubuntu-26.04-amd64.iso ← item (display name: ubuntu-26.04-amd64)
        │   │   │   └── item.json
        │   │   └── arm64/
        │   │       ├── ubuntu-26.04-arm64.iso ← item (display name: ubuntu-26.04-arm64)
        │   │       └── item.json
        │   └── ubuntu-24.04/                  ← container
        │       ├── amd64/
        │       │   ├── ubuntu-24.04-amd64.iso ← item (display name: ubuntu-24.04-amd64)
        │       │   └── item.json
        │       └── arm64/
        │           ├── ubuntu-24.04-arm64.iso ← item (display name: ubuntu-24.04-arm64)
        │           └── item.json
        ├── rhel/                              ← container
        │   ├── rhel-10/                       ← container
        │   │   ├── amd64/
        │   │   │   ├── rhel-10-amd64.iso      ← item (display name: rhel-10-amd64)
        │   │   │   └── item.json
        │   │   └── arm64/
        │   │       ├── rhel-10-arm64.iso      ← item (display name: rhel-10-arm64)
        │   │       └── item.json
        │   └── rhel-9/                        ← container
        │       ├── amd64/
        │       │   ├── rhel-9-amd64.iso       ← item (display name: rhel-9-amd64)
        │       │   └── item.json
        │       └── arm64/
        │           ├── rhel-9-arm64.iso       ← item (display name: rhel-9-arm64)
        │           └── item.json
        └── debian/                            ← container
            ├── debian-13/                     ← container
            │   ├── amd64/
            │   │   ├── debian-13-amd64.iso    ← item (display name: debian-13-amd64)
            │   │   └── item.json
            │   └── arm64/
            │       ├── debian-13-arm64.iso    ← item (display name: debian-13-arm64)
            │       └── item.json
            └── debian-12/                     ← container
                ├── amd64/
                │   ├── debian-12-amd64.iso    ← item (display name: debian-12-amd64)
                │   └── item.json
                └── arm64/
                    ├── debian-12-arm64.iso    ← item (display name: debian-12-arm64)
                    └── item.json
    ```

## Expectations

| Rule                         | Behavior                                                                           |
| ---------------------------- | ---------------------------------------------------------------------------------- |
| Item boundary                | Each **leaf content folder** (has files) becomes at most one library item          |
| Containers                   | Folders with only subfolders are recursed into                                     |
| Storage path / SelfHref      | Library-relative folder path (for example `iso/ubuntu/ubuntu-26.04/amd64`)          |
| Display name                 | Flat folder name; nested single-ISO uses the ISO basename (without `.iso`)         |
| Files in a folder            | All non-folder files in that directory become the item’s `files[]` list            |
| Nested folders under an item | **Skipped** (do not put required package files in subfolders)                      |
| Files at the base path       | **Ignored** (for example a loose `ubuntu-26.04-amd64.iso` next to folders)          |
| Empty folders                | No item is emitted                                                                 |
| Content upload               | You must upload content yourself (UI, REST, CI); `generate` only writes metadata   |

## Content Types

Type is detected from file extensions in the folder:

| Presence in folder     | Item type                                                          |
| ---------------------- | ------------------------------------------------------------------ |
| Any `.ovf` or `.ova`   | `vcsp.ovf` (OVF wins even if `.iso` files are also present)        |
| Else any `.iso`        | `vcsp.iso`                                                         |
| Else other files only  | `vcsp.other`                                                       |

Certificate files (`.cert`) on OVF items are omitted when `--skip-cert` / `skip_cert` is
`true` (the default).

## Recommended Layouts

**Nested ISOs Content Library Items**

???+ example "Example: Nested ISOs Content Library Items"

    ```text
    {path}/
    ├── lib.json
    ├── items.json
    └── iso/                                   ← container
        ├── ubuntu/                            ← container
        │   ├── ubuntu-26.04/                  ← container
        │   │   ├── amd64/
        │   │   │   ├── ubuntu-26.04-amd64.iso ← item (display name: ubuntu-26.04-amd64)
        │   │   │   └── item.json
        │   │   └── arm64/
        │   │       ├── ubuntu-26.04-arm64.iso ← item (display name: ubuntu-26.04-arm64)
        │   │       └── item.json
        │   └── ubuntu-24.04/      ← container
        │       ├── amd64/
        │       │   ├── ubuntu-24.04-amd64.iso ← item (display name: ubuntu-24.04-amd64)
        │       │   └── item.json
        │       └── arm64/
        │           ├── ubuntu-24.04-arm64.iso ← item (display name: ubuntu-24.04-arm64)
        │           └── item.json
        ├── rhel/                              ← container
        │   ├── rhel-10/                       ← container
        │   │   ├── amd64/
        │   │   │   ├── rhel-10-amd64.iso      ← item (display name: rhel-10-amd64)
        │   │   │   └── item.json
        │   │   └── arm64/
        │   │       ├── rhel-10-arm64.iso      ← item (display name: rhel-10-arm64)
        │   │       └── item.json
        │   └── rhel-9/                        ← container
        │       ├── amd64/
        │       │   ├── rhel-9-amd64.iso       ← item (display name: rhel-9-amd64)
        │       │   └── item.json
        │       └── arm64/
        │           ├── rhel-9-arm64.iso       ← item (display name: rhel-9-arm64)
        │           └── item.json
        └── debian/                            ← container
            ├── debian-13/                     ← container
            │   ├── amd64/
            │   │   ├── debian-13-amd64.iso    ← item (display name: debian-13-amd64)
            │   │   └── item.json
            │   └── arm64/
            │       ├── debian-13-arm64.iso    ← item (display name: debian-13-arm64)
            │       └── item.json
            └── debian-12/                     ← container
                ├── amd64/
                │   ├── debian-12-amd64.iso    ← item (display name: debian-12-amd64)
                │   └── item.json
                └── arm64/
                    ├── debian-12-arm64.iso    ← item (display name: debian-12-arm64)
                    └── item.json
    ```

    Result: One ISO content library item per architecture folder.

**Flat ISO Content Library Items**

???+ example "Example: Flat ISO Content Library Items"

    ```text
    {path}/
    ├── lib.json
    ├── items.json
    ├── ubuntu-26.04-amd64/
    │   ├── ubuntu-26.04-amd64.iso             ← item (display name: ubuntu-26.04-amd64)
    │   └── item.json
    ├── ubuntu-26.04-arm64/
    │   ├── ubuntu-26.04-arm64.iso             ← item (display name: ubuntu-26.04-arm64)
    │   └── item.json
    ├── ubuntu-24.04-amd64/
    │   ├── ubuntu-24.04-amd64.iso             ← item (display name: ubuntu-24.04-amd64)
    │   └── item.json
    ├── ubuntu-24.04-arm64/
    │   ├── ubuntu-24.04-arm64.iso             ← item (display name: ubuntu-24.04-arm64)
    │   └── item.json
    ├── rhel-10-amd64/
    │   ├── rhel-10-amd64.iso                  ← item (display name: rhel-10-amd64)
    │   └── item.json
    ├── rhel-10-arm64/
    │   ├── rhel-10-arm64.iso                  ← item (display name: rhel-10-arm64)
    │   └── item.json
    ├── rhel-9-amd64/
    │   ├── rhel-9-amd64.iso                   ← item (display name: rhel-9-amd64)
    │   └── item.json
    ├── rhel-9-arm64/
    │   ├── rhel-9-arm64.iso                   ← item (display name: rhel-9-arm64)
    │   └── item.json
    ├── debian-13-amd64/
    │   ├── debian-13-amd64.iso                ← item (display name: debian-13-amd64)
    │   └── item.json
    ├── debian-13-arm64/
    │   ├── debian-13-arm64.iso                ← item (display name: debian-13-arm64)
    │   └── item.json
    ├── debian-12-amd64/
    │   ├── debian-12-amd64.iso                ← item (display name: debian-12-amd64)
    │   └── item.json
    └── debian-12-arm64/
        ├── debian-12-arm64.iso                ← item (display name: debian-12-arm64)
        └── item.json
    ```

    Result: 12 ISO content library items without container folders.

**OVF / OVA Content Library Items**

Keep the OVF/OVA and its disks/sidecars together in one folder:

???+ example "Example: OVF / OVA Content Library Items"

    ```text
    {path}/
    ├── lib.json
    ├── items.json
    ├── ubuntu-26.04-amd64-ovf/                 ← item (display name: ubuntu-26.04-amd64-ovf)
    │   ├── ubuntu-26.04-amd64.ovf
    │   ├── ubuntu-26.04-amd64-disk-0.vmdk
    │   ├── ubuntu-26.04-amd64.mf
    │   └── item.json
    ├── ubuntu-24.04-amd64-ova/                 ← item (display name: ubuntu-24.04-amd64-ova)
    │   ├── ubuntu-24.04-amd64.ova
    │   └── item.json
    ├── rhel-10-amd64-ovf/                      ← item (display name: rhel-10-amd64-ovf)
    │   ├── rhel-10-amd64.ovf
    │   ├── rhel-10-amd64-disk-0.vmdk
    │   ├── rhel-10-amd64.mf
    │   └── item.json
    └── rhel-9-amd64-ova/                       ← item (display name: rhel-9-amd64-ova)
        ├── rhel-9-amd64.ova
        └── item.json
    ```

## Guardrails

!!! warning "Root-level Files are Ignored"

    ```text
    {path}/
    ├── ubuntu-26.04-amd64.iso ← ignored
    ├── ubuntu-26.04-amd64.ova ← ignored
    ├── ubuntu-26.04-arm64.iso ← ignored
    ├── ubuntu-26.04-arm64.ova ← ignored
    ├── ubuntu-24.04-amd64.iso ← ignored
    ├── ubuntu-24.04-amd64.ova ← ignored
    ├── ubuntu-24.04-arm64.iso ← ignored
    ├── ubuntu-24.04-arm64.ova ← ignored
    ├── rhel-10-amd64.iso      ← ignored
    ├── rhel-10-amd64.ova      ← ignored
    ├── rhel-10-arm64.iso      ← ignored
    ├── rhel-10-arm64.ova      ← ignored
    ├── rhel-9-amd64.iso       ← ignored
    ├── rhel-9-amd64.ova       ← ignored
    ├── rhel-9-arm64.iso       ← ignored
    ├── rhel-9-arm64.ova       ← ignored
    ├── debian-13-amd64.iso    ← ignored
    ├── debian-13-amd64.ova    ← ignored
    ├── debian-13-arm64.iso    ← ignored
    ├── debian-13-arm64.ova    ← ignored
    ├── debian-12-amd64.iso    ← ignored
    ├── debian-12-amd64.ova    ← ignored
    ├── debian-12-arm64.iso    ← ignored
    └── debian-12-arm64.ova    ← ignored
    ```

    Wrap each object in a folder if you want a content library item.

!!! warning "Keep Display Names Unique"

    A subscribed content library will reject library metadata that contains two items with the same display name.

    When content library items have distinct names so they do not collide.

    For example, you could use `ubuntu-26.04-amd64-ova` and `ubuntu-26.04-amd64-ovf` naming for OVA/OVF and `ubuntu-26.04-amd64` for ISO.

!!! warning "Keep Package Files as Direct Children"

    If a folder already has content files (`.iso`, `.ovf`, `.ova`, disks, …),
    `generate` does **not** recurse into its subfolders. Put every file for that
    library item in the leaf folder itself — not in nested directories.

!!! tip "Use One Object per Leaf Folder"

    For trees, use a leaf folder per object so each becomes its own content library item.
