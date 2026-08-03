# Docs site scripts

Versioned documentation helpers for Properdocs + MaterialX + mike.

| Script | Purpose |
|--------|---------|
| `mike-deploy.sh <version> [--update-latest]` | Deploy one version to `gh-pages` |
| `mike-preview.sh` | Local multi-version preview on `docs-preview` |
| `mike-backfill.sh [VERSION ...]` | Publish multiple versions to `gh-pages` |

Makefile targets: `docs`, `docs-serve`, `docs-serve-mike`, `docs-serve-mike-only`, `docs-backfill`, `docs-deploy`.
