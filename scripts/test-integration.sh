#!/usr/bin/env bash

# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

# Local Artifactory with Docker (default):
#   make test-integration
#   ./scripts/test-integration.sh
#
# External Artifactory:
#   USE_DOCKER=0 \
#     ARTIFACTORY_URL=https://artifactory.example.com/artifactory \
#     ARTIFACTORY_USERNAME=admin \
#     ARTIFACTORY_PASSWORD=secret \
#     ARTIFACTORY_REPOSITORY=generic-local \
#     make test-integration
#
# Environment:
#   USE_DOCKER=1              Start Artifactory stack with Docker (default: 1)
#   INTEGRATION_FRESH=1       Reset Artifactory stack volumes in Docker (default: 0)
#   TEARDOWN_AFTER=1          Stop Artifactory stack when tests complete (default: 0)
#   ARTIFACTORY_URL           Artifactory URL (required for external Artifactory)
#   ARTIFACTORY_USERNAME      Artifactory Username (for password authentication)
#   ARTIFACTORY_PASSWORD      Artifactory Password (for password authentication)
#   ARTIFACTORY_API_KEY       Artifactory API Key (for API key authentication)
#   ARTIFACTORY_TOKEN         Artifactory Access Token (for access token authentication)
#   ARTIFACTORY_REPOSITORY    Artifactory Repository (required for external Artifactory)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/test/docker-compose.yaml"

USE_DOCKER="${USE_DOCKER:-1}"
INTEGRATION_FRESH="${INTEGRATION_FRESH:-0}"
TEARDOWN_AFTER="${TEARDOWN_AFTER:-0}"

source "${SCRIPT_DIR}/artifactory-docker.sh"

normalize_auth_env() {
  if [ -z "${ARTIFACTORY_PASSWORD:-}" ] && [ -n "${ARTIFACTORY_API_KEY:-}" ]; then
    export ARTIFACTORY_PASSWORD="${ARTIFACTORY_API_KEY}"
    export ARTIFACTORY_USERNAME="${ARTIFACTORY_USERNAME:-admin}"
  fi

  if [ -z "${ARTIFACTORY_PASSWORD:-}" ] && [ -n "${ARTIFACTORY_TOKEN:-}" ]; then
    # Artifactory accepts access tokens using basic authentication (username optional).
    export ARTIFACTORY_PASSWORD="${ARTIFACTORY_TOKEN}"
    export ARTIFACTORY_USERNAME="${ARTIFACTORY_USERNAME:-}"
  fi
}

require_external_artifactory_env() {
  if [ -z "${ARTIFACTORY_URL:-}" ]; then
    echo "error: USE_DOCKER=0 requires ARTIFACTORY_URL" >&2
    echo "example:" >&2
    echo "  USE_DOCKER=0 ARTIFACTORY_URL=https://artifactory.example.com/artifactory \\" >&2
    echo "    ARTIFACTORY_USERNAME=admin ARTIFACTORY_PASSWORD=secret \\" >&2
    echo "    ARTIFACTORY_REPOSITORY=generic-local make test-integration" >&2
    return 1
  fi

  normalize_auth_env

  if [ -z "${ARTIFACTORY_PASSWORD:-}" ]; then
    echo "error: USE_DOCKER=0 requires auth via ARTIFACTORY_PASSWORD, ARTIFACTORY_API_KEY, or ARTIFACTORY_TOKEN" >&2
    return 1
  fi

  if [ -z "${ARTIFACTORY_REPOSITORY:-}" ]; then
    echo "error: USE_DOCKER=0 requires ARTIFACTORY_REPOSITORY" >&2
    return 1
  fi
}

configure_environment() {
  if [ "${USE_DOCKER}" = "1" ]; then
    export_artifactory_defaults
    return 0
  fi

  require_external_artifactory_env
}

start_docker_stack() {
  if [ "${INTEGRATION_FRESH}" = "1" ]; then
    echo "Resetting Artifactory stack volumes in Docker for integration tests..."
    artifactory_compose down -v --remove-orphans || true
    echo "Artifactory stack volumes in Docker removed."
  fi

  echo "Starting Artifactory stack in Docker for integration tests..."
  if artifactory_compose up -d --wait 2>/dev/null; then
    echo "Artifactory stack in Docker is up."
    return 0
  fi

  artifactory_compose up -d
  wait_for_artifactory
  echo "Artifactory stack in Docker is up."
}

teardown_docker_stack() {
  echo "Stopping Artifactory stack in Docker for integration tests..."
  artifactory_compose down --remove-orphans
  echo "Artifactory stack in Docker stopped."
}

trap_handler() {
  if [ "${TEARDOWN_AFTER}" = "1" ] && [ "${USE_DOCKER}" = "1" ]; then
    teardown_docker_stack
  fi
}

print_test_configuration() {
  echo "Running integration tests..."
  echo "  USE_DOCKER=${USE_DOCKER}"
  echo "  ARTIFACTORY_URL=${ARTIFACTORY_URL}"
  echo "  ARTIFACTORY_USERNAME=${ARTIFACTORY_USERNAME:-}"
  echo "  ARTIFACTORY_REPOSITORY=${ARTIFACTORY_REPOSITORY}"
}

run_integration_tests() {
  INTEGRATION_TESTS=1 go test -v -count=1 -run "TestIntegration" \
    ./internal/artifactory/ -timeout=15m
  echo "Integration tests completed."
}

main() {
  trap trap_handler EXIT

  cd "${ROOT_DIR}"

  configure_environment

  if [ "${USE_DOCKER}" = "1" ]; then
    start_docker_stack
  else
    echo "Skipping Docker startup; using ARTIFACTORY_URL=${ARTIFACTORY_URL}"
    wait_for_artifactory
  fi

  print_test_configuration
  run_integration_tests
}

main "$@"
