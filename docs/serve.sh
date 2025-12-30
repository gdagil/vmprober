#!/bin/bash

# Script to serve Jekyll documentation locally using Docker

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Starting Jekyll server with Docker..."
echo "Documentation will be available at http://localhost:4000"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running. Please start Docker Desktop first."
    exit 1
fi

# Run Jekyll in Docker
docker run --rm -it \
    -p 4000:4000 \
    -v "$PWD":/srv/jekyll \
    -v "$PWD/vendor/bundle":/usr/local/bundle \
    jekyll/jekyll:latest \
    jekyll serve --host 0.0.0.0 --force_polling
