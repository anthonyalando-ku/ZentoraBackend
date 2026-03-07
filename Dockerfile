FROM golang:1.24 AS builder
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod tidy
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -trimpath \
    -o zentora ./cmd/api/main.go

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/zentora   /app/zentora
COPY --from=builder /app/secrets   /app/secrets
COPY --from=builder /app/static    /app/static

# Uploads directory must exist so Gin can serve and the app can write to it.
# The actual data is bind-mounted from the host via compose.
RUN mkdir -p /app/uploads/products

RUN useradd -u 10001 -M -s /sbin/nologin nonroot \
    && chown -R nonroot:nonroot /app

USER nonroot

EXPOSE 8002

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8002/api/v1/health || exit 1

ENTRYPOINT ["/app/zentora"]