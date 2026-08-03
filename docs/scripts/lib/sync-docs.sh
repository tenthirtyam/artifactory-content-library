#!/usr/bin/env bash
# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MIT

# Sync curated docs files from a git tag for historical version deploys.

set -euo pipefail

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=paths.sh
source "${LIB_DIR}/paths.sh"

sync_docs_from_tag() {
  local tag="$1"
  local path

  if ! git -C "$REPO_ROOT" rev-parse "$tag" >/dev/null 2>&1; then
    echo "error: tag ${tag} not found" >&2
    return 1
  fi

  for path in "${DOCS_SYNC_PATHS[@]}"; do
    if git -C "$REPO_ROOT" cat-file -e "${tag}:${path}" 2>/dev/null; then
      git -C "$REPO_ROOT" checkout "$tag" -- "$path"
    else
      echo "warning: ${path} missing on ${tag}; keeping working tree copy" >&2
    fi
  done
}

restore_docs_from_head() {
  local path
  if ! git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    return 0
  fi
  for path in "${DOCS_SYNC_PATHS[@]}"; do
    git -C "$REPO_ROOT" checkout HEAD -- "$path" 2>/dev/null || true
  done
}
