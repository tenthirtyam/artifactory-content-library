#!/usr/bin/env bash
# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MIT

# Shared mike binary resolution for versioned docs deploy/preview.

set -euo pipefail

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=paths.sh
source "${LIB_DIR}/paths.sh"

resolve_mike() {
  MIKE="${REPO_ROOT}/.venv/bin/mike"
  if [[ ! -x "$MIKE" ]]; then
    MIKE="$(command -v mike)"
  fi
  if [[ -z "${MIKE:-}" || ! -x "$MIKE" ]]; then
    echo "error: mike not found; run: make install-docs" >&2
    exit 1
  fi
  # Ensure properdocs/mkdocs from the same venv are on PATH for mike's build.
  local bindir
  bindir="$(cd "$(dirname "$MIKE")" && pwd)"
  export PATH="${bindir}:${PATH}"
}

mike_commit_args() {
  MIKE_CMD_COMMIT_ARGS=()

  if [[ -z "${MIKE_COMMIT_MESSAGE:-}" ]]; then
    return 0
  fi

  local msg="$MIKE_COMMIT_MESSAGE"
  if [[ -n "${MIKE_COMMIT_VERSION:-}" ]]; then
    msg="${msg//\{version\}/$MIKE_COMMIT_VERSION}"
  fi
  msg="${msg//\{branch\}/${MIKE_BRANCH:-gh-pages}}"
  MIKE_CMD_COMMIT_ARGS=(-m "$msg")
}

mike_cmd() {
  local subcommand="$1"
  shift
  local branch="${MIKE_BRANCH:-gh-pages}"
  local config="${DOCS_CONFIG}"
  MIKE_CMD_COMMIT_ARGS=()
  case "$subcommand" in
    deploy | set-default | delete | rename | retitle)
      mike_commit_args
      ;;
  esac
  (
    cd "$REPO_ROOT"
    if ((${#MIKE_CMD_COMMIT_ARGS[@]})); then
      "$MIKE" "$subcommand" -b "$branch" -F "$config" "${MIKE_CMD_COMMIT_ARGS[@]}" "$@"
    else
      "$MIKE" "$subcommand" -b "$branch" -F "$config" "$@"
    fi
  )
}
