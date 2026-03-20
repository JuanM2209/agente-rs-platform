FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Download dependencies first so the layer is cached when only source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /nucleus-agent ./cmd

# ---------------------------------------------------------------------------
# Runtime image
# ---------------------------------------------------------------------------
FROM alpine:3.19

# ca-certificates: needed for TLS connections to the control plane
# tzdata: proper timezone support
# mbusd: Modbus TCP↔RTU bridge used by the serial bridge feature
RUN apk --no-cache add ca-certificates tzdata mbusd

WORKDIR /root/

COPY --from=builder /nucleus-agent .

# The agent does not bind any fixed port itself; it connects outbound.
EXPOSE 0

CMD ["./nucleus-agent"]
