# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build backend with embedded frontend
FROM golang:1.25-alpine AS backend-builder
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
# Copy frontend dist into the go:embed path
COPY --from=frontend-builder /build/dist ./cmd/api/static/
# Pure-Go build (glebarez/sqlite, no CGO needed)
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o homelog ./cmd/api/

# Stage 3: Minimal runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata wget poppler-utils
RUN addgroup -g 1000 homelog && adduser -D -u 1000 -G homelog homelog
WORKDIR /app
COPY --from=backend-builder /build/homelog .
RUN mkdir -p /app/data /app/uploads && chown -R homelog:homelog /app
USER homelog
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1
CMD ["./homelog"]
