#!/usr/bin/env bash

set -euo pipefail

IMAGE="${IMAGE:-ghcr.io/juanm2209/agente-rs:latest}"
CONTAINER_NAME="${CONTAINER_NAME:-agente-rs}"
FACTORY_PATH="${FACTORY_PATH:-/data/nucleus/factory}"
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-}"
AGENT_SECRET="${AGENT_SECRET:-}"
TENANT_ID="${TENANT_ID:-test-org}"
LOG_LEVEL="${LOG_LEVEL:-info}"
HEARTBEAT_INTERVAL="${HEARTBEAT_INTERVAL:-30s}"
INVENTORY_SCAN_INTERVAL="${INVENTORY_SCAN_INTERVAL:-60s}"
SERIAL_DEVICE="${SERIAL_DEVICE:-}"
MBUSD_HOST_PATH="${MBUSD_HOST_PATH:-}"

if ! command -v docker >/dev/null 2>&1; then
  echo "[ERROR] docker is not installed or not in PATH." >&2
  exit 1
fi

if [[ -z "${CONTROL_PLANE_URL}" ]]; then
  echo "[ERROR] CONTROL_PLANE_URL is required." >&2
  exit 1
fi

if [[ -z "${AGENT_SECRET}" ]]; then
  echo "[ERROR] AGENT_SECRET is required." >&2
  exit 1
fi

if [[ ! -d "${FACTORY_PATH}" ]]; then
  echo "[ERROR] FACTORY_PATH does not exist: ${FACTORY_PATH}" >&2
  exit 1
fi

SERIAL_NUMBER_FILE="${FACTORY_PATH}/nucleus_serial_number"
if [[ ! -f "${SERIAL_NUMBER_FILE}" ]]; then
  echo "[WARN] ${SERIAL_NUMBER_FILE} was not found. The agent will need DEVICE_ID from another source." >&2
fi

echo "[INFO] Pulling image ${IMAGE}"
docker pull "${IMAGE}"

if docker ps -a --format '{{.Names}}' | grep -Fxq "${CONTAINER_NAME}"; then
  echo "[INFO] Replacing existing container ${CONTAINER_NAME}"
  docker rm -f "${CONTAINER_NAME}" >/dev/null
fi

args=(
  run -d
  --name "${CONTAINER_NAME}"
  --restart unless-stopped
  --network host
  -v "${FACTORY_PATH}:/data/nucleus/factory:ro"
  -e "CONTROL_PLANE_URL=${CONTROL_PLANE_URL}"
  -e "AGENT_SECRET=${AGENT_SECRET}"
  -e "TENANT_ID=${TENANT_ID}"
  -e "LOG_LEVEL=${LOG_LEVEL}"
  -e "HEARTBEAT_INTERVAL=${HEARTBEAT_INTERVAL}"
  -e "INVENTORY_SCAN_INTERVAL=${INVENTORY_SCAN_INTERVAL}"
)

if [[ -n "${SERIAL_DEVICE}" ]]; then
  args+=(--device "${SERIAL_DEVICE}:${SERIAL_DEVICE}")
fi

if [[ -n "${MBUSD_HOST_PATH}" ]]; then
  if [[ ! -f "${MBUSD_HOST_PATH}" ]]; then
    echo "[ERROR] MBUSD_HOST_PATH does not exist: ${MBUSD_HOST_PATH}" >&2
    exit 1
  fi

  args+=(-v "${MBUSD_HOST_PATH}:/usr/local/bin/mbusd:ro")
fi

args+=("${IMAGE}")

echo "[INFO] Starting container ${CONTAINER_NAME}"
docker "${args[@]}"

echo "[OK] ${CONTAINER_NAME} started."
echo "[INFO] Check status with: docker ps"
echo "[INFO] Check logs with: docker logs -f ${CONTAINER_NAME}"
