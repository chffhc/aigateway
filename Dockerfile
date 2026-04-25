# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

# Build stage
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

# CGO is enabled for sqlite-backed builds
RUN apk add --no-cache build-base ca-certificates

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/aigateway ./cmd/server

# Runtime stage
FROM --platform=$TARGETPLATFORM alpine:${ALPINE_VERSION}

WORKDIR /app

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S -G app -h /app app && \
    mkdir -p /app/templates /app/static /data && \
    chown -R app:app /app /data

COPY --from=builder /out/aigateway /app/aigateway
COPY --chown=app:app templates ./templates
COPY --chown=app:app static ./static

ENV PORT=8080 \
    DB_PATH=/data/aigateway.db \
    GIN_MODE=release

VOLUME ["/data"]

EXPOSE 8080
USER app

ENTRYPOINT ["/app/aigateway"]
