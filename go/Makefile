.PHONY: help build install test test-verbose test-coverage fmt vet lint clean

# Default target shows help
help: ## Show this help message
	@echo "Box - Go annotation-driven API framework"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build/compile all packages
	@echo "Building packages..."
	@go build ./...
	@echo "✓ Build successful"

install: ## Install the box CLI tool
	@echo "Installing box CLI..."
	@go install ./cmd/box
	@echo "✓ box CLI installed to $(shell go env GOPATH)/bin/box"
	@echo "  Make sure $(shell go env GOPATH)/bin is in your PATH"

test: ## Run all tests
	@echo "Running tests..."
	@go test ./...
	@echo "✓ Tests complete"

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	@go test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

fmt: ## Format all Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete"

lint: ## Run golangci-lint (requires installation)
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "Error: golangci-lint not installed. Install with: go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@golangci-lint run
	@echo "✓ Lint complete"

clean: ## Remove build artifacts and coverage files
	@echo "Cleaning..."
	@go clean
	@rm -f coverage.out coverage.html
	@rm -rf bin/
	@echo "✓ Clean complete"

.DEFAULT_GOAL := help
