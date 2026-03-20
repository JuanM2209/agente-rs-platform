FROM golang:1.22-bullseye AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /app

# Download dependencies first so the layer is cached when only source changes.
COPY go.mod ./
RUN go mod download

COPY . .

RUN set -eux; \
    export GOOS="${TARGETOS:-linux}"; \
    if [ -n "${TARGETARCH:-}" ]; then \
      export GOARCH="${TARGETARCH}"; \
    fi; \
    if [ "${TARGETARCH:-}" = "arm" ] && [ -n "${TARGETVARIANT:-}" ]; then \
      export GOARM="${TARGETVARIANT#v}"; \
    fi; \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /agente-rs ./cmd

# ---------------------------------------------------------------------------
# Runtime image
# ---------------------------------------------------------------------------
FROM debian:bullseye-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /agente-rs .
COPY third_party/mbusd/armv7/mbusd /tmp/mbusd-armv7

RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    if [ "${arch}" = "armhf" ]; then \
      install -m 0755 /tmp/mbusd-armv7 /usr/local/bin/mbusd; \
    fi; \
    rm -f /tmp/mbusd-armv7

# The agent does not bind any fixed port itself; it connects outbound.
EXPOSE 0

CMD ["./agente-rs"]
