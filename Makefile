# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
GO?=go

# Default target
.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Download dependencies
.PHONY: deps
deps: ## Download Go dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Run tests
.PHONY: test
test: ## Run tests
	@echo "Running tests"
	$(GO) test ./...

# Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Lint code
.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	$(GO) tool golangci-lint run

# Lint code and fixing it
.PHONY: lint-fix
lint-fix: ## Run linter and fixing it
	@echo "Running linter and fixing it..."
	$(GO) tool golangci-lint run --fix

.PHONY: install-git-hooks
install-git-hooks:
	prek install
