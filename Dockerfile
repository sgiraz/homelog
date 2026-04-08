# syntax=docker/dockerfile:1

# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
ARG APP_VERSION=dev
WORKDIR /build
COPY frontend/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY frontend/ .
RUN APP_VERSION=${APP_VERSION} npm run build

# Stage 2: Build backend with embedded frontend
FROM golang:1.25-alpine AS backend-builder
RUN apk add --no-cache git
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY backend/ .
# Copy frontend dist into the go:embed path
COPY --from=frontend-builder /build/dist ./cmd/api/static/
# Pure-Go build (glebarez/sqlite, no CGO needed)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.appVersion=${APP_VERSION}" -o homelog ./cmd/api/

# Stage 3: Minimal runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata wget poppler-utils && \
    addgroup -g 1000 homelog && adduser -D -u 1000 -G homelog homelog
WORKDIR /app
COPY --from=backend-builder /build/homelog .
RUN mkdir -p /app/data /app/uploads /app/data/avatars && chown -R homelog:homelog /app
USER homelog
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1
CMD ["./homelog"]
