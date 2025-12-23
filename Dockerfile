# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -o vmprober ./cmd/vmprober

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/vmprober .

# Copy example config
COPY configs/config.yaml.example /etc/vmprober/config.yaml

# Create WAL directory
RUN mkdir -p /var/lib/vmprober/wal

# Expose port
EXPOSE 8429

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8429/health || exit 1

# Run the application
CMD ["./vmprober", "--config=/etc/vmprober/config.yaml"]

