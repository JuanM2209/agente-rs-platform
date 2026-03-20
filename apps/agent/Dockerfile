FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

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
FROM alpine:3.19

# ca-certificates: needed for TLS connections to the control plane
# tzdata: proper timezone support
#
# NOTE:
# mbusd is not bundled here because it is not available as a stable Alpine
# package in this build path. For serial bridge support, provide an mbusd
# binary from the host or build a derived image that adds it to PATH.
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /agente-rs .

# The agent does not bind any fixed port itself; it connects outbound.
EXPOSE 0

CMD ["./agente-rs"]
