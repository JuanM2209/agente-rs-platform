FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /app

# Download dependencies first so the layer is cached when only source changes.
COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /agente-rs ./cmd

# ---------------------------------------------------------------------------
# Runtime image
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETVARIANT

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /agente-rs .
COPY third_party/mbusd/armv7/mbusd /tmp/mbusd-armv7

RUN set -eux; \
    if [ "${TARGETARCH}" = "arm" ] && [ "${TARGETVARIANT}" = "v7" ]; then \
      install -m 0755 /tmp/mbusd-armv7 /usr/local/bin/mbusd; \
    fi; \
    rm -f /tmp/mbusd-armv7

# The agent does not bind any fixed port itself; it connects outbound.
EXPOSE 0

CMD ["./agente-rs"]
