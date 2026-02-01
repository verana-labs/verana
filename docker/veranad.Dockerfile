# syntax=docker/dockerfile:1.5
ARG GO_VERSION=1.25
ARG BASE_IMAGE=ubuntu:22.04

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app
COPY . .

ARG USE_PREBUILT=false
ARG PREBUILT_BINARY=binaries/veranad-linux-amd64
ARG LDFLAGS=""

RUN if [ "$USE_PREBUILT" = "true" ] && [ -f "$PREBUILT_BINARY" ]; then \
      cp "$PREBUILT_BINARY" /tmp/veranad; \
    else \
      CGO_ENABLED=0 go build -ldflags="$LDFLAGS" -o /tmp/veranad ./cmd/veranad; \
    fi

FROM ${BASE_IMAGE}

RUN apt-get update && apt-get install -y \
    bash \
    ca-certificates \
    curl \
    jq \
    s3cmd \
    tini \
    wget \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /tmp/veranad /usr/local/bin/veranad

EXPOSE 26656 26657 1317 9090

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["veranad"]
