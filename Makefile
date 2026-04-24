.PHONY: build run test fmt lint clean help

# Build variables
BINARY_NAME := x402-server
BUILD_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X github.com/AxLabs/x402-go-server-example/internal/version.Version=$(VERSION) \
                     -X github.com/AxLabs/x402-go-server-example/internal/version.Commit=$(COMMIT) \
                     -X github.com/AxLabs/x402-go-server-example/internal/version.BuildTime=$(BUILD_TIME)"

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOFMT := gofmt
GOMOD := $(GO) mod

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "x402-go-server-example"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the server binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

## run: Run the server (requires .env or environment variables)
run: build
	@echo "Starting server..."
	@if [ -f .env ]; then \
		export $$(cat .env | grep -v '^#' | xargs) && ./$(BUILD_DIR)/$(BINARY_NAME); \
	else \
		./$(BUILD_DIR)/$(BINARY_NAME); \
	fi

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

## test-cover: Run tests with coverage report
test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## fmt: Format Go source files
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	@mkdir -p .cache/go-build .cache/golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then \
		GOCACHE=$$(pwd)/.cache/go-build GOLANGCI_LINT_CACHE=$$(pwd)/.cache/golangci-lint golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

## tidy: Tidy go.mod and go.sum
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

## verify: Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## all: Run fmt, vet, lint, and test
all: fmt vet lint test
	@echo "All checks passed!"
