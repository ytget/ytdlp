SHELL:=bash

.DEFAULT_GOAL := help

GREEN := $(shell tput setaf 2 2>/dev/null || echo "")
YELLOW := $(shell tput setaf 3 2>/dev/null || echo "")
RED := $(shell tput setaf 1 2>/dev/null || echo "")
RESET := $(shell tput sgr0 2>/dev/null || echo "")

.PHONY: help
help: ## Available commands
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[0;33m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""


##@ Targets

.PHONY: build
build: ## Build application
	go build -o ytdlp_v2 ./cmd/ytdlp

.PHONY: install
install: ## Install application locally
	go install ./...

.PHONY: test
test: ## Run tests with coverage report
	go test -v -cover ./... -coverprofile=coverage.out
	@echo ""
	@echo "Coverage Summary:"
	@go tool cover -func=coverage.out | grep total | awk -v red="$(RED)" -v yellow="$(YELLOW)" -v green="$(GREEN)" -v reset="$(RESET)" '{raw=$$3; val=raw; gsub(/[^0-9.]/, "", val); if ((val+0)==0) color=red; else if ((val+0) < 80) color=yellow; else color=green; print "Total coverage: " color raw reset}'
	@echo "Detailed coverage saved to: coverage.out"
	@echo "Run 'make coverage' to view detailed report in console"
	@echo "Run 'make coverage-summary' for quick overview"

.PHONY: coverage
coverage-by-functions: ## Show detailed coverage by function
	@if [ -f coverage.out ]; then \
		echo "Coverage by function:"; \
		go tool cover -func=coverage.out; \
	else \
		echo "No coverage file found. Run 'make test' first."; \
	fi

.PHONY: coverage-summary
coverage-by-packages: ## Show coverage summary by package
	@if [ -f coverage.out ]; then \
		echo "Coverage by package:"; \
		echo ""; \
		go test -coverprofile=coverage.out ./... 2>/dev/null | grep -E "(coverage:|PASS|FAIL)"; \
		echo ""; \
		total=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
		echo "Total coverage: $$total%"; \
		echo "Run 'make coverage' for function-level details"; \
	else \
		echo "No coverage file found. Run 'make test' first."; \
	fi


.PHONY: lint
lint: ## Run linter (golangci-lint)
	golangci-lint run ./...

.PHONY: format
format: ## Format code
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -l -w .

.PHONY: tidy
tidy: ## Tidy go.mod
	go mod tidy

.PHONY: cover
cover: ## Run tests with coverage and generate HTML report
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

.PHONY: race
race: ## Run tests with -race
	go test -race ./...

##@ E2E

.PHONY: e2e
e2e: ## Run end-to-end test (requires YTDLP_E2E=1)
	YTDLP_E2E=1 go test -tags e2e ./e2e -v

.PHONY: e2e-url
e2e-url: ## Run e2e test with a specific URL: make e2e-url URL="https://..."
	YTDLP_E2E=1 YTDLP_E2E_URL="$(URL)" go test -tags e2e ./e2e -v

##@ Download

.PHONY: download
download: build ## Build and download video: make download URL="https://..."
	@if [ -z "$(URL)" ]; then \
		echo "Error: URL is required. Usage: make download URL=\"https://example.com/video/123\""; \
		exit 1; \
	fi
	./ytdlp_v2 "$(URL)"

.PHONY: dl
dl: ## Build and download video (alias for download)
	@make download URL="$(URL)"


##@ Aliases

.PHONY: b
b: ## Build application
	@make build

.PHONY: i
i: ## Install application locally
	@make install

.PHONY: t
t: ## Run tests
	@make test

.PHONY: c
cf: ## Show detailed coverage by functions
	@make coverage-by-functions

.PHONY: cs
cp: ## Show coverage summary by packages
	@make coverage-by-packages

.PHONY: l
l: ## Run linter (golangci-lint)
	@make lint

.PHONY: f
f: ## Format code
	@make format

.PHONY: ty
ty: ## Tidy go.mod
	@make tidy

.PHONY: c
c: ## Run tests with coverage and generate HTML report
	@make cover

.PHONY: r
r: ## Run tests with -race
	@make race

.PHONY: e
e: ## Run end-to-end test (requires YTDLP_E2E=1)
	@make e2e

.PHONY: eu
eu: ## Run e2e test with a specific URL: make eu URL="https://..."
	@make e2e-url URL="$(URL)"

