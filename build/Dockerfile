# syntax=docker/dockerfile:1.7

FROM golang:alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/releasebot \
    ./cmd/releasebot

FROM alpine:3.22

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S releasebot \
    && adduser -S -D -H -G releasebot releasebot

WORKDIR /app

COPY --from=builder /out/releasebot /app/releasebot

USER releasebot

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/releasebot"]
