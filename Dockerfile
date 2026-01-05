# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -o vmprober ./cmd/vmprober

# Runtime stage - Alpine
FROM alpine:latest

ARG TARGETOS
ARG TARGETARCH

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/vmprober .

# Copy example config
COPY config/vmprober/config.yaml.example /etc/vmprober/config.yaml

# Create WAL directory
RUN mkdir -p /var/lib/vmprober/wal

# Expose port
EXPOSE 8429

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8429/health || exit 1

# Run the application
CMD ["./vmprober", "--config=/etc/vmprober/config.yaml"]
