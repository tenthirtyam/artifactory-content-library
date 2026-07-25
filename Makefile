# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

NAME=artifactory-content-library
BINARY=bin/${NAME}

COUNT?=1
TEST?=$(shell go list ./...)
GO ?= go
GOTOOLCHAIN ?= auto
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

VENV ?= .venv
BOOTSTRAP_PYTHON ?= $(shell \
	if command -v python3.12 >/dev/null 2>&1; then \
		command -v python3.12; \
	elif command -v python3.13 >/dev/null 2>&1; then \
		command -v python3.13; \
	elif command -v python3.11 >/dev/null 2>&1; then \
		command -v python3.11; \
	else \
		command -v python3; \
	fi)
PYTHON := $(VENV)/bin/python
PIP := $(PYTHON) -m pip
DOCS := $(PYTHON) -m properdocs
DOCS_PORT ?= 8000
DOCS_CONFIG := docs/properdocs.yml
DOCS_REQUIREMENTS := docs/requirements.txt
DOCS_STAMP := $(VENV)/.docs-installed
DOCS_SCRIPTS := docs/scripts

.PHONY: all build clean clean-build clean-cache clean-docs coverage deps-check deps-upgrade deps-vuln \
	docs docs-serve docs-serve-mike docs-serve-mike-only docs-backfill docs-deploy \
	format help install install-docs lint test test-integration test-integration-clean \
	test-integration-down test-unit

help:
	@echo "Testing:"
	@echo "  make test                            Run unit and integration tests."
	@echo "  make test-unit                       Run unit tests."
	@echo "  make test-integration                Run integration tests (local Docker by default)."
	@echo "  make test-integration-clean          Stop and remove integration Docker volumes."
	@echo "  make test-integration-down           Stop integration Docker stack."
	@echo ""
	@echo "Integration Targets:"
	@echo "  Local Docker:  make test-integration"
	@echo "  Fresh volumes: INTEGRATION_FRESH=1 make test-integration"
	@echo "  Teardown:      TEARDOWN_AFTER=1 make test-integration"
	@echo "  External:      USE_DOCKER=0 ARTIFACTORY_URL=... ARTIFACTORY_USERNAME=... \\"
	@echo "                   ARTIFACTORY_PASSWORD=... ARTIFACTORY_REPOSITORY=... \\"
	@echo "                   make test-integration"
	@echo ""
	@echo "Build:"
	@echo "  make all                             Clean build artifacts, build, and run unit tests."
	@echo "  make build                           Build ${BINARY}."
	@echo "  make install                         Install to GOPATH/bin."
	@echo "  make format                          Format Go sources with gofmt."
	@echo "  make lint                            Run golangci-lint (falls back to go vet)."
	@echo "  make coverage                        Run unit tests with coverage."
	@echo "  make clean                           Remove build artifacts and Go caches."
	@echo ""
	@echo "Documentation:"
	@echo "  make install-docs                    Install documentation dependencies."
	@echo "  make docs                            Build the documentation site."
	@echo "  make docs-serve                      Serve the documentation site locally."
	@echo "  make docs-serve-mike                 Build and serve a multi-version mike preview."
	@echo "  make docs-serve-mike-only            Serve an existing mike preview branch."
	@echo "  make docs-backfill                   Publish one or more versions to gh-pages."
	@echo "  make docs-deploy VERSION=x.y.z       Deploy a single version (add UPDATE_LATEST=1)."
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps-upgrade                    Upgrade module dependencies."
	@echo "  make deps-check                      Fail if build-graph modules are outdated."
	@echo "  make deps-vuln                       Run govulncheck."

all: clean-build build test-unit

build:
	@mkdir -p bin
	@GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) build -ldflags "$(LDFLAGS)" -o ${BINARY} .

install:
	@GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) install -ldflags "$(LDFLAGS)" .

format:
	@gofmt -w .

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		GOTOOLCHAIN=$(GOTOOLCHAIN) golangci-lint run ./...; \
	else \
		echo "golangci-lint not found; falling back to go vet"; \
		GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) vet ./...; \
	fi

coverage:
	@GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) test -race -count $(COUNT) $(TEST) -coverprofile=coverage.out -timeout=3m
	@GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) tool cover -func=coverage.out

test: test-unit test-integration

test-unit:
	@GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) test -race -count $(COUNT) $(TEST) -timeout=3m

test-integration: build
	@./scripts/test-integration.sh

test-integration-clean: test-integration-down
	@docker compose -f test/docker-compose.yaml down -v --remove-orphans

test-integration-down:
	@docker compose -f test/docker-compose.yaml down --remove-orphans

deps-upgrade:
	@GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) get -u ./...
	@GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) mod tidy
	@echo "Dependencies upgraded. Run 'make test-unit' to verify."

# Checks packages in our compile graph only (not pruned test-only deps of deps).
deps-check:
	@outdated=""; \
	for entry in $$(GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) list -deps -f '{{with .Module}}{{.Path}}@{{.Version}}{{end}}' ./... | sort -u); do \
		[ -z "$$entry" ] && continue; \
		upd=$$(GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) list -m -u -f '{{if .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' "$$entry" 2>/dev/null); \
		if [ -n "$$upd" ]; then \
			echo "$$upd"; \
			outdated="1"; \
		fi; \
	done; \
	if [ -n "$$outdated" ]; then \
		echo "Build-graph dependencies are outdated."; \
		exit 1; \
	fi; \
	echo "All build-graph dependencies are current."

deps-vuln:
	@GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

$(PYTHON):
	$(BOOTSTRAP_PYTHON) -m venv $(VENV)
	$(PIP) install --upgrade pip

install-docs: $(DOCS_STAMP)

$(DOCS_STAMP): $(PYTHON) $(DOCS_REQUIREMENTS)
	@if ! $(PIP) --version >/dev/null 2>&1; then \
		echo "Rebuilding broken virtual environment at $(VENV)"; \
		rm -rf $(VENV); \
		$(MAKE) $(PYTHON); \
	fi
	$(PIP) install -r $(DOCS_REQUIREMENTS)
	@touch $(DOCS_STAMP)

docs: install-docs
	$(DOCS) build -f $(DOCS_CONFIG)

docs-serve: install-docs
	@pids="$$(lsof -tiTCP:$(DOCS_PORT) -sTCP:LISTEN 2>/dev/null || true)"; \
	for pid in $$pids; do \
		echo "Stopping existing server on port $(DOCS_PORT): $$pid"; \
		kill "$$pid" 2>/dev/null || true; \
	done; \
	for attempt in 1 2 3 4 5 6 7 8 9 10; do \
		if ! lsof -tiTCP:$(DOCS_PORT) -sTCP:LISTEN >/dev/null 2>&1; then \
			break; \
		fi; \
		sleep 1; \
	done; \
	if lsof -tiTCP:$(DOCS_PORT) -sTCP:LISTEN >/dev/null 2>&1; then \
		echo "Force stopping server on port $(DOCS_PORT)"; \
		lsof -tiTCP:$(DOCS_PORT) -sTCP:LISTEN | xargs kill -9 2>/dev/null || true; \
	fi
	$(DOCS) serve -f $(DOCS_CONFIG) --open --livereload -a 127.0.0.1:$(DOCS_PORT) -w ./

docs-serve-mike: install-docs
	@$(DOCS_SCRIPTS)/mike-preview.sh $(if $(VERSIONS),$(VERSIONS),) $(MIKE_PREVIEW_ARGS)

docs-serve-mike-only: install-docs
	@$(DOCS_SCRIPTS)/mike-preview.sh --serve-only

docs-backfill: install-docs
	@$(DOCS_SCRIPTS)/mike-backfill.sh $(if $(VERSIONS),$(VERSIONS),)

docs-deploy: install-docs
	@test -n "$(VERSION)" || (echo "Set VERSION=x.y.z (without v prefix)" && exit 1)
	@$(DOCS_SCRIPTS)/mike-deploy.sh $(VERSION) $(if $(filter 1 true yes,$(UPDATE_LATEST)),--update-latest,)

clean: clean-build clean-cache clean-docs

clean-build:
	@rm -rf bin/ coverage.out

clean-cache:
	@$(GO) clean -testcache
	@$(GO) clean -cache

clean-docs:
	rm -rf site
	rm -f $(DOCS_STAMP)
