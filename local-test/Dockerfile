FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates build-base git

WORKDIR /app

# Copy source code
COPY . .

# Temporarily remove tool block for build (tool block syntax not fully supported in all Go versions)
# The tool block is only needed for local development tools, not for building the binary
RUN sed -i '/^tool (/,/^)$/d' go.mod || true

# Build the binary
RUN CGO_ENABLED=0 go build -o veranad ./cmd/veranad

# Final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/veranad /usr/local/bin/veranad
EXPOSE 26656 26657 1317 9090
ENTRYPOINT ["veranad"]