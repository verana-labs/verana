FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash curl git jq ca-certificates nodejs npm && rm -rf /var/lib/apt/lists/*

ENV GO_VERSION=1.22.5
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

RUN npm install -g @anthropic-ai/claude-code

RUN git config --global user.name  "cipher-bot" \
 && git config --global user.email "cipher-bot@users.noreply.github.com" \
 && git config --global --add safe.directory '*'

WORKDIR /workspace
ENTRYPOINT ["claude"]
