# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary (auto-detect architecture)
RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w" \
    -o homelog-api \
    ./cmd/api/main.go

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies including poppler-utils for PDF processing
RUN apk add --no-cache ca-certificates sqlite-libs tzdata wget poppler-utils

# Create non-root user
RUN addgroup -g 1000 homelog && \
    adduser -D -u 1000 -G homelog homelog

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/homelog-api .

# Create data directory
RUN mkdir -p /app/data /app/uploads && \
    chown -R homelog:homelog /app

# Switch to non-root user
USER homelog

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Run application
CMD ["./homelog-api"]
