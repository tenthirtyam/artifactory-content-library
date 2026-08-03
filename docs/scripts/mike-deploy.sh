#!/usr/bin/env bash
# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MIT

# Deploy a documentation version to GitHub Pages with mike.

# Usage:
#   mike-deploy.sh <version> [--update-latest]
#
# Builds documentation with Properdocs/MaterialX via mike and pushes the
# result to the gh-pages branch. When tag v<version> exists, curated docs
# paths are staged from that tag first.
#
# Environment:
#   MIKE_BRANCH          Remote git branch for mike (default: gh-pages)
#   MIKE_COMMIT_MESSAGE  Optional git commit message for mike deploy/set-default
#   MIKE_COMMIT_VERSION  Expands {version} in MIKE_COMMIT_MESSAGE

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"

# shellcheck source=lib/paths.sh
source "${LIB_DIR}/paths.sh"
# shellcheck source=lib/mike-env.sh
source "${LIB_DIR}/mike-env.sh"
# shellcheck source=lib/sync-docs.sh
source "${LIB_DIR}/sync-docs.sh"

usage() {
  cat <<EOF
usage: $0 <version> [--update-latest]

  <version>         Release version without the v prefix (e.g. 0.1.0)
  --update-latest   Also set the latest alias and default version

Stages docs from tag v<version> when available, builds with mike, and
pushes to gh-pages.

Environment:
  MIKE_BRANCH          Remote branch for mike (default: gh-pages)
  MIKE_COMMIT_MESSAGE  Optional git commit message (supports {version}, {branch})
  MIKE_COMMIT_VERSION  Set automatically to the deployed version

Examples:
  $0 0.1.0
  $0 0.1.0 --update-latest
EOF
  exit 1
}

UPDATE_LATEST=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h | --help) usage ;;
    --update-latest)
      UPDATE_LATEST=true
      shift
      ;;
    -*)
      echo "error: unknown option: $1" >&2
      usage
      ;;
    *)
      if [[ -n "${VERSION:-}" ]]; then
        echo "error: unexpected argument: $1" >&2
        usage
      fi
      VERSION="${1#v}"
      shift
      ;;
  esac
done

[[ -n "${VERSION:-}" ]] || usage

TAG="v${VERSION}"
DOCS_SYNCED=false

restore_synced_docs() {
  if [[ "$DOCS_SYNCED" == "true" ]]; then
    restore_docs_from_head
  fi
}

if git -C "$REPO_ROOT" rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Staging documentation for ${VERSION} from ${TAG}..."
  sync_docs_from_tag "$TAG"
  DOCS_SYNCED=true
else
  echo "warning: ${TAG} not found; using current docs" >&2
fi

trap restore_synced_docs EXIT

rm -rf "${REPO_ROOT}/site"
resolve_mike
export MIKE_BRANCH="${MIKE_BRANCH:-gh-pages}"
MIKE_COMMIT_VERSION="$VERSION"

if [[ "$UPDATE_LATEST" == "true" ]]; then
  mike_cmd deploy --push --update-aliases "$VERSION" latest
  mike_cmd set-default --push latest
else
  mike_cmd deploy --push "$VERSION"
fi

trap - EXIT
restore_synced_docs
