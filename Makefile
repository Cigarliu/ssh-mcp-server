.PHONY: build clean test release help

VERSION ?= 1.0.0
BUILD_DIR := build
DIST_DIR := dist
APP_NAME := sshmcp
REPO := github.com/Cigarliu/ssh-mcp-server
LDFLAGS := -s -w -X $(REPO)/pkg/version.Version=$(VERSION) -X $(REPO)/pkg/version.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Platform configurations
WINDOWS_AMD64 := windows/amd64
WINDOWS_386 := windows/386
WINDOWS_ARM64 := windows/arm64
LINUX_AMD64 := linux/amd64
LINUX_ARM64 := linux/arm64
DARWIN_AMD64 := darwin/amd64
DARWIN_ARM64 := darwin/arm64

PLATFORMS := $(WINDOWS_AMD64) $(WINDOWS_386) $(WINDOWS_ARM64) \
            $(LINUX_AMD64) $(LINUX_ARM64) \
            $(DARWIN_AMD64) $(DARWIN_ARM64)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build for current platform
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)$(if $(filter windows,$(OS)),.exe) ./cmd/server

build-all: clean ## Build for all platforms
	@echo "Building for all platforms..."
	@$(MAKE) -C . -f Makefile.multi build-all VERSION=$(VERSION)

clean: ## Clean build directories
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

release: clean ## Build release binaries for all platforms
	@echo "Building release v$(VERSION)..."
	@chmod +x build.sh
	@./build.sh

tag: ## Create git tag
	@echo "Creating tag v$(VERSION)..."
	@git tag -a v$(VERSION) -m "Release v$(VERSION)"
	@echo "Tag created. Push with: git push origin v$(VERSION)"

push-tag: tag ## Create and push git tag
	@echo "Pushing tag v$(VERSION)..."
	@git push origin v$(VERSION)

gh-release: ## Create GitHub release (requires gh CLI)
	@echo "Creating GitHub release v$(VERSION)..."
	@gh release create v$(VERSION) \
		--title "SSH MCP Server v$(VERSION)" \
		--notes "See CHANGELOG.md for details"

upload: gh-release ## Upload binaries to GitHub release
	@echo "Uploading binaries to GitHub release v$(VERSION)..."
	@gh release upload v$(VERSION) $(DIST_DIR)/*

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: fmt vet ## Format and lint code
	@echo "Running linters..."
	@golangci-lint run ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

.DEFAULT_GOAL := build
