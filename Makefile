.PHONY: build test clean run docker-build docker-run help docs docs-install docs-serve docs-build

# Variables
BINARY_NAME=vmprober
VERSION?=1.0.0
BUILD_TIME=$(shell date +%Y-%m-%dT%H:%M:%S)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Build the application
build:
	@echo "Building ${BINARY_NAME}..."
	@go build ${LDFLAGS} -o bin/${BINARY_NAME} ./cmd/vmprober

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Run the application
run: build
	@echo "Running ${BINARY_NAME}..."
	@./bin/${BINARY_NAME} -config ./config/vmprober/config.yaml.example

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Documentation - install dependencies
docs-install:
	@echo "Installing documentation dependencies..."
	@cd docs && bundle install

# Documentation - serve locally
docs-serve:
	@echo "Starting documentation server at http://localhost:4000..."
	@cd docs && bundle exec jekyll serve --livereload

# Documentation - build static site
docs-build:
	@echo "Building documentation..."
	@cd docs && bundle exec jekyll build

# Documentation - shortcut for serve
docs: docs-serve

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Build and run the application"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Build and run Docker container"
	@echo "  deps           - Install dependencies"
	@echo "  docs           - Serve documentation locally (alias for docs-serve)"
	@echo "  docs-install   - Install documentation dependencies (bundle install)"
	@echo "  docs-serve     - Serve documentation at http://localhost:4000"
	@echo "  docs-build     - Build static documentation site"
	@echo "  help           - Show this help message"



