FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o authy ./cmd/api/main.go

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/authy /app/authy
COPY --from=builder /app/secrets /app/secrets
COPY --from=builder /app/static /app/static

RUN useradd -u 10001 nonroot && chown -R nonroot:nonroot /app
USER nonroot

EXPOSE 8001
ENTRYPOINT ["/app/authy"]