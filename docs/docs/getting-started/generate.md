---
icon: octicons/file-added-16
---

# Generate

The `generate` command scans content under an Artifactory repository path and writes
vSphere content library metadata (`lib.json`, `items.json`, and per-item `item.json`).

It does **not** upload OVF/OVA/ISO binaries. Place content in Artifactory first using the
[content layout](content-layout.md) rules (one item per leaf folder under `--path`).

## CLI Example

Refer to [`generate` command options](../configuration/reference.md#generate-command-options)
for the full flag list.

```bash
artifactory-content-library generate \
  --url "https://packages.example.com/artifactory" \
  --api-key "api-key" \
  --ssl-verify true \
  --name "artifactory-library" \
  --repo "example-repository" \
  --path "example-path"
```

## Environment Variables

Refer to [Artifactory environment variables](../configuration/reference.md#artifactory-environment-variables)
for the full list.

```bash
export ARTIFACTORY_URL="https://packages.example.com/artifactory"
export ARTIFACTORY_API_KEY="api-key"
export ARTIFACTORY_SSL_VERIFY="true"

artifactory-content-library generate \
  --name "artifactory-library" \
  --repo "example-repository" \
  --path "example-path"
```

## Configuration File

Refer to [`init` command options](../configuration/reference.md#init-command-options)
for the full flag list.

Generate an example Artifactory configuration:

```bash
artifactory-content-library init --type artifactory --output config.yaml
```

Example:

```yaml
--8<-- "docs/snippets/example-artifactory-config.yaml"
```

```bash
artifactory-content-library generate --config config.yaml
```

Use exactly one Artifactory authentication method (`api_key`, username/password, or
`token`). Prefer `${ENV}` references for secrets.

## Incremental Updates

On re-run, `generate`:

- Reuses existing library and item IDs when metadata already exists (matched by storage
  path / SelfHref).
- Bumps item and library versions when file names or checksums (SHA1, falling back to
  MD5) change.
- Removes orphaned `item.json` entries for folders that disappeared.
- Reports the content library as up to date when nothing changed.
