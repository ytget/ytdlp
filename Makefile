SHELL:=bash

.DEFAULT_GOAL := help


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
test: ## Run tests
	go test -v ./...

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

.PHONY: l
l: ## Run linter (golangci-lint)
	@make lint

.PHONY: f
f: ## Format code
	@make format

