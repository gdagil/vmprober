# Docker Build Guide

This guide explains how to build Docker images for VMProber with support for multiple platforms and different base images.

## Available Dockerfiles

### Dockerfile (Alpine-based)
- **Base image**: `alpine:latest`
- **Size**: ~15-20 MB
- **Features**:
  - Full shell and utilities (wget, etc.)
  - Healthcheck support
  - CA certificates included
  - Timezone data

### Dockerfile.scratch (Minimal)
- **Base image**: `scratch` (empty)
- **Size**: ~5-10 MB
- **Features**:
  - Minimal size
  - Only binary and CA certificates
  - No shell or utilities
  - No healthcheck (configure at orchestrator level)

## Building Images

### Using Makefile

```bash
# Build Alpine image for current platform
make -f Makefile.docker docker-build

# Build Alpine image for multiple platforms (amd64, arm64)
make -f Makefile.docker docker-build-alpine

# Build Scratch image for multiple platforms
make -f Makefile.docker docker-build-scratch

# Build all variants
make -f Makefile.docker docker-build-all

# Build with custom version
make -f Makefile.docker docker-build-alpine VERSION=1.0.0

# Build for specific platforms
make -f Makefile.docker docker-build-alpine PLATFORMS=linux/amd64,linux/arm64,linux/arm/v7

# Push to registry
make -f Makefile.docker docker-push REGISTRY=myregistry.io VERSION=1.0.0
```

### Using Build Script

```bash
# Build Alpine image
./build-docker.sh --dockerfile Dockerfile --version 1.0.0

# Build Scratch image
./build-docker.sh --dockerfile Dockerfile.scratch --version 1.0.0

# Build for specific platforms
./build-docker.sh --platforms linux/amd64,linux/arm64

# Build and push to registry
./build-docker.sh --registry myregistry.io --push

# Build single platform
./build-docker.sh --single-platform linux/arm64
```

### Using Docker Buildx Directly

```bash
# Create buildx builder (if not exists)
docker buildx create --name multiarch --use --bootstrap

# Build Alpine image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=1.0.0 \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  -t vmprober:1.0.0-alpine \
  -f Dockerfile \
  --load \
  .

# Build Scratch image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=1.0.0 \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  -t vmprober:1.0.0-scratch \
  -f Dockerfile.scratch \
  --load \
  .
```

## Image Tags

Images are tagged with the following pattern:
- `{registry}/{image-name}:{version}-{variant}`
- `{registry}/{image-name}:latest-{variant}`

Examples:
- `vmprober:1.0.0-alpine`
- `vmprober:latest-scratch`
- `myregistry.io/vmprober:1.0.0-alpine`

## Supported Platforms

- `linux/amd64` - Intel/AMD 64-bit
- `linux/arm64` - ARM 64-bit (Apple Silicon, ARM servers)
- `linux/arm/v7` - ARM 32-bit (optional)

## Running Images

### Alpine Image

```bash
docker run -d \
  --name vmprober \
  -p 8429:8429 \
  -v $(pwd)/config.yaml:/etc/vmprober/config.yaml \
  vmprober:latest-alpine
```

### Scratch Image

```bash
docker run -d \
  --name vmprober \
  -p 8429:8429 \
  -v $(pwd)/config.yaml:/etc/vmprober/config.yaml \
  vmprober:latest-scratch
```

## Health Checks

### Alpine Image
The Alpine image includes a built-in healthcheck:
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8429/health || exit 1
```

### Scratch Image
The Scratch image does not include a healthcheck (no shell/tools available). Configure healthcheck at the orchestrator level:

**Kubernetes:**
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8429
  initialDelaySeconds: 5
  periodSeconds: 30
```

**Docker Compose:**
```yaml
healthcheck:
  test: ["CMD", "wget", "--spider", "http://localhost:8429/health"]
  interval: 30s
  timeout: 3s
  retries: 3
```

## Configuration

Both images expect configuration at `/etc/vmprober/config.yaml`. You can:

1. **Mount a config file:**
   ```bash
   docker run -v /path/to/config.yaml:/etc/vmprober/config.yaml vmprober:latest-alpine
   ```

2. **Use environment variables** (if supported by the application)

3. **Override config path:**
   ```bash
   docker run vmprober:latest-alpine --config=/custom/path/config.yaml
   ```

## WAL Directory

The WAL (Write-Ahead Log) directory is automatically created by the application at runtime. By default, it uses `/var/lib/vmprober/wal`. You can mount a volume:

```bash
docker run -v /host/wal:/var/lib/vmprober/wal vmprober:latest-alpine
```

## Troubleshooting

### Build fails with "platform not supported"
Ensure you have QEMU installed for cross-platform builds:
```bash
# macOS
brew install qemu

# Linux
sudo apt-get install qemu-user-static
```

### Scratch image can't connect to HTTPS endpoints
Ensure CA certificates are properly included. The scratch image includes CA certificates from the builder stage.

### Healthcheck fails in scratch image
Use external healthcheck tools or configure healthcheck at the orchestrator level (Kubernetes, Docker Swarm, etc.).

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build and Push Docker Images

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2

      - name: Login to Registry
        uses: docker/login-action@v2
        with:
          registry: ${{ secrets.REGISTRY }}
          username: ${{ secrets.REGISTRY_USERNAME }}
          password: ${{ secrets.REGISTRY_PASSWORD }}

      - name: Build and Push Alpine
        uses: docker/build-push-action@v4
        with:
          context: .
          file: ./Dockerfile
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ${{ secrets.REGISTRY }}/vmprober:${{ github.ref_name }}-alpine
            ${{ secrets.REGISTRY }}/vmprober:latest-alpine
          build-args: |
            VERSION=${{ github.ref_name }}
            BUILD_TIME=${{ github.event.head_commit.timestamp }}
            GIT_COMMIT=${{ github.sha }}

      - name: Build and Push Scratch
        uses: docker/build-push-action@v4
        with:
          context: .
          file: ./Dockerfile.scratch
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ${{ secrets.REGISTRY }}/vmprober:${{ github.ref_name }}-scratch
            ${{ secrets.REGISTRY }}/vmprober:latest-scratch
          build-args: |
            VERSION=${{ github.ref_name }}
            BUILD_TIME=${{ github.event.head_commit.timestamp }}
            GIT_COMMIT=${{ github.sha }}
```

## Best Practices

1. **Use Alpine for development** - easier debugging with shell access
2. **Use Scratch for production** - minimal attack surface and size
3. **Always specify version tags** - avoid using `latest` in production
4. **Build for your target platform** - test on the same architecture
5. **Mount configs as volumes** - don't bake configs into images
6. **Use healthchecks** - configure at orchestrator level for scratch images
