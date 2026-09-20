# ==========================
# Build Stage
# ==========================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git (required for some Go modules)
RUN apk add --no-cache git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/financial-planner .

# Build migration binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/migrate ./migrate

# ==========================
# Runtime Stage
# ==========================
FROM alpine:3.22

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget

# Create non-root user
RUN addgroup -S appgroup && \
    adduser -S appuser -G appgroup

# Copy binaries
COPY --from=builder /bin/financial-planner .
COPY --from=builder /bin/migrate .

# Copy required assets
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/config ./config
COPY --from=builder /app/docs ./docs

# Set application ownership. Reports are uploaded directly to S3;
# no persistent local generated_reports directory is created.
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8081

HEALTHCHECK --interval=30s \
    --timeout=5s \
    --start-period=20s \
    --retries=3 \
CMD wget --spider -q http://127.0.0.1:8081/boltpe-financial-planner/health || exit 1

ENTRYPOINT ["./financial-planner"]
