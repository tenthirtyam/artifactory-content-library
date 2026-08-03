#!/usr/bin/env bash
# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MIT

# Shared paths and defaults for docs scripts.

set -euo pipefail

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "${LIB_DIR}/.." && pwd)"
DOCS_SITE_DIR="$(cd "${SCRIPTS_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${DOCS_SITE_DIR}/.." && pwd)"

DOCS_CONFIG="${DOCS_CONFIG:-${DOCS_SITE_DIR}/properdocs.yml}"
DOCS_SYNC_PATHS=(docs)
