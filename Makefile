SHELL         := /usr/bin/env bash
.DEFAULT_GOAL := help

GO       ?= go
BIN_DIR  := bin
TOOLS    := $(notdir $(wildcard cmd/*))
LDFLAGS  ?=
GOFLAGS  ?=

GO_BUILD_FLAGS = $(GOFLAGS) $(if $(LDFLAGS),-ldflags '$(LDFLAGS)',)

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "; printf "Targets:\n"} /^[a-zA-Z_%-]+:.*?## / { printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: all
all: check build ## Lint, test, then build everything

.PHONY: build
build: ## Build every tool into bin/
	@mkdir -p $(BIN_DIR)
	@set -e; for t in $(TOOLS); do \
	  echo "  $(GO) build -o $(BIN_DIR)/$$t ./cmd/$$t"; \
	  $(GO) build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$$t ./cmd/$$t; \
	done

build-%: ## Build a single tool (e.g. make build-uhsort)
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$* ./cmd/$*

.PHONY: install
install: ## go install all tools to $$GOBIN (or $$GOPATH/bin)
	$(GO) install $(GO_BUILD_FLAGS) ./cmd/...

.PHONY: test
test: ## Run unit tests
	$(GO) test ./...

.PHONY: race
race: ## Run tests with the race detector
	$(GO) test -race ./...

.PHONY: cover
cover: ## Run tests with coverage report at coverage.out
	$(GO) test -coverprofile=coverage.out ./...
	@echo "View: go tool cover -html=coverage.out"

.PHONY: vet
vet: ## go vet ./...
	$(GO) vet ./...

.PHONY: fmt
fmt: ## Format all Go files in place
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any Go file is not gofmt-clean
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
	  echo "Unformatted files:"; echo "$$out"; \
	  exit 1; \
	fi

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

.PHONY: check
check: vet fmt-check test ## vet + fmt-check + test

.PHONY: clean
clean: ## Remove bin/ and coverage artifacts
	rm -rf $(BIN_DIR) coverage.out

.PHONY: tools
tools: ## List tools discovered under cmd/
	@printf '%s\n' $(TOOLS)
