SHELL := $(shell command -v bash)
.ONESHELL:
.SHELLFLAGS := -euo pipefail -c

# Keep recursive make calls quiet ("Entering directory" noise).
MAKEFLAGS += --no-print-directory

GO_FILES := $(shell find . -name '*.go' -not -path './.git/*')

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z._-]+:.*?##/ {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build app
	@go build -o plexamp-tui

fmt: ## Format Go files
	@gofmt -w $(GO_FILES)

fmt-check: ## Check Go formatting
	@files="$$(gofmt -l $(GO_FILES))"
	if [[ -n "$$files" ]]; then
		printf 'gofmt needed for:\n%s\n' "$$files"
		exit 1
	fi

vet: ## Run go vet
	@go vet ./...

lint: ## Run golangci-lint
	@if ! command -v golangci-lint >/dev/null 2>&1; then
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2"
		exit 1
	fi
	golangci-lint run ./...

test: ## Run tests
	@go test ./...

check: fmt-check vet lint test ## Run formatting, linting, vet, and tests

clean: ## Delete build output
	@rm -f plexamp-tui

run: ## Run app
	@./plexamp-tui

rebuild: clean build run ## Build and run app
