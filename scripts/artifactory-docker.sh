#!/usr/bin/env bash

# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

# Shared helpers and defaults for the local Artifactory Docker test stack.

# Defaults for Artifactory OSS.
DEFAULT_ARTIFACTORY_URL="http://localhost:8081/artifactory"
DEFAULT_ARTIFACTORY_USERNAME="admin"
DEFAULT_ARTIFACTORY_PASSWORD="password"
DEFAULT_ARTIFACTORY_REPOSITORY="example-repo-local"

export_artifactory_defaults() {
  export ARTIFACTORY_URL="${ARTIFACTORY_URL:-${DEFAULT_ARTIFACTORY_URL}}"
  export ARTIFACTORY_USERNAME="${ARTIFACTORY_USERNAME:-${DEFAULT_ARTIFACTORY_USERNAME}}"
  export ARTIFACTORY_PASSWORD="${ARTIFACTORY_PASSWORD:-${DEFAULT_ARTIFACTORY_PASSWORD}}"
  export ARTIFACTORY_REPOSITORY="${ARTIFACTORY_REPOSITORY:-${DEFAULT_ARTIFACTORY_REPOSITORY}}"
}

artifactory_compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "${COMPOSE_FILE}" "$@"
    return
  fi

  if command -v docker-compose >/dev/null 2>&1; then
    docker-compose -f "${COMPOSE_FILE}" "$@"
    return
  fi

  echo "error: docker compose is required" >&2
  return 1
}

wait_for_artifactory() {
  local attempt max_attempts ping_url http_code auth_args

  max_attempts=60
  ping_url="${ARTIFACTORY_URL%/}/api/system/ping"
  auth_args=(-u "${ARTIFACTORY_USERNAME:-}:${ARTIFACTORY_PASSWORD}")

  echo "Waiting for Artifactory at ${ping_url}..."

  for attempt in $(seq 1 "${max_attempts}"); do
    http_code="$(
      curl -s -o /dev/null -w "%{http_code}" \
        "${auth_args[@]}" \
        "${ping_url}" 2>/dev/null || true
    )"

    if [ "${http_code}" = "200" ]; then
      echo "Artifactory is ready"
      return 0
    fi

    echo "  attempt ${attempt}/${max_attempts} (HTTP ${http_code:-000})"
    sleep 10
  done

  echo "error: Artifactory did not become ready in time" >&2
  return 1
}

verify_artifactory_credentials() {
  local base_url="${1%/}"
  local username="$2"
  local password="$3"
  local http_code

  http_code="$(
    curl -s -o /dev/null -w "%{http_code}" \
      -u "${username}:${password}" \
      "${base_url}/api/system/ping" 2>/dev/null || true
  )"

  if [ "${http_code}" = "200" ]; then
    return 0
  fi

  echo "error: authentication failed for ${base_url} (HTTP ${http_code:-000})" >&2
  return 1
}

fetch_artifactory_api_key() {
  local base_url="${1%/}"
  local username="$2"
  local password="$3"
  local response api_key

  response="$(
    curl -s -u "${username}:${password}" \
      "${base_url}/api/security/apiKey" 2>/dev/null || true
  )"
  api_key="$(printf '%s' "${response}" | sed -n 's/.*"apiKey"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

  if [ -n "${api_key}" ]; then
    printf '%s' "${api_key}"
    return 0
  fi

  response="$(
    curl -s -u "${username}:${password}" -X POST \
      "${base_url}/api/security/apiKey" 2>/dev/null || true
  )"
  api_key="$(printf '%s' "${response}" | sed -n 's/.*"apiKey"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

  if [ -n "${api_key}" ]; then
    printf '%s' "${api_key}"
    return 0
  fi

  echo "error: failed to retrieve API key from ${base_url}" >&2
  return 1
}

is_artifactory_oss() {
  local base_url="${1%/}"
  local username="$2"
  local password="$3"
  local response

  response="$(
    curl -s -u "${username}:${password}" \
      "${base_url}/api/system/version" 2>/dev/null || true
  )"

  case "${response}" in
    *"Artifactory OSS"*) return 0 ;;
  esac

  return 1
}
