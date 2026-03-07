SHELL := $(shell command -v bash)
.ONESHELL:
.SHELLFLAGS := -euo pipefail -c

# Keep recursive make calls quiet ("Entering directory" noise).
MAKEFLAGS += --no-print-directory

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z._-]+:.*?##/ {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build app
	@go build -o plexamp-tui

clean: ## Delete build output
	@rm -f plexamp-tui

run: ## Run app
	@./plexamp-tui

rebuild: clean build run ## Build and run app
