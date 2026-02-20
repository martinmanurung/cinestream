# ============================================================
# Stage 1: Builder
# Compile both API and Worker binaries
# ============================================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o bin/api ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o bin/worker ./cmd/worker


# ============================================================
# Stage 2: API Runtime
# Minimal image for running the API server
# ============================================================
FROM alpine:3.21 AS api

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary and config from builder
COPY --from=builder /app/bin/api .
COPY --from=builder /app/app-config.yaml .

# Set ownership
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["./api"]


# ============================================================
# Stage 3: Worker Runtime
# Image for the transcoding worker (needs ffmpeg)
# ============================================================
FROM alpine:3.21 AS worker

# Install runtime dependencies including ffmpeg for transcoding
RUN apk add --no-cache ca-certificates tzdata ffmpeg

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary and config from builder
COPY --from=builder /app/bin/worker .
COPY --from=builder /app/app-config.yaml .

# Set ownership
RUN chown -R appuser:appgroup /app

USER appuser

ENTRYPOINT ["./worker"]
