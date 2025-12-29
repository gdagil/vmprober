#!/usr/bin/env bash
# Install development tools for vmprober
set -euo pipefail

echo "Installing development tools..."

# Check Go version
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go version: $GO_VERSION"

# Install golangci-lint from source (to match project Go version)
echo "Installing golangci-lint from source..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install goimports
echo "Installing goimports..."
go install golang.org/x/tools/cmd/goimports@latest

# Install govulncheck
echo "Installing govulncheck..."
go install golang.org/x/vuln/cmd/govulncheck@latest

# Check if pre-commit is installed
if command -v pre-commit &> /dev/null; then
    echo "pre-commit is already installed"
else
    echo "Installing pre-commit..."
    if command -v pip3 &> /dev/null; then
        pip3 install pre-commit
    elif command -v pip &> /dev/null; then
        pip install pre-commit
    elif command -v brew &> /dev/null; then
        brew install pre-commit
    else
        echo "Error: Cannot install pre-commit. Please install pip or brew first."
        exit 1
    fi
fi

# Install pre-commit hooks
echo "Installing pre-commit hooks..."
pre-commit install

echo ""
echo "✅ All tools installed successfully!"
echo ""
echo "Available commands:"
echo "  pre-commit run --all-files  # Run all pre-commit checks"
echo "  make lint                   # Run golangci-lint"
echo "  make test                   # Run tests"
echo "  make fmt                    # Format code"
