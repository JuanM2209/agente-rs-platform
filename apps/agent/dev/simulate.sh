#!/usr/bin/env bash
# simulate.sh — Start multiple simulated Nucleus edge agents locally.
#
# Usage:
#   ./dev/simulate.sh [API_URL] [NUM_AGENTS]
#
# Defaults:
#   API_URL     = ws://localhost:8080/ws/agent
#   NUM_AGENTS  = 4
#
# Each agent runs in its own Docker container with a unique DEVICE_ID.
# Containers are removed automatically when this script exits (Ctrl+C).

set -euo pipefail

API_URL="${1:-ws://localhost:8080/ws/agent}"
NUM_AGENTS="${2:-4}"
AGENT_SECRET="${AGENT_SECRET:-dev-secret}"
TENANT_ID="${TENANT_ID:-tenant-alpha}"
IMAGE_TAG="nucleus-agent:local"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$(dirname "${SCRIPT_DIR}")"

# ---------------------------------------------------------------------------
# Build the agent image once.
# ---------------------------------------------------------------------------
echo "==> Building agent image: ${IMAGE_TAG}"
docker build -t "${IMAGE_TAG}" "${AGENT_DIR}"

# ---------------------------------------------------------------------------
# Track started container IDs so we can clean them up on exit.
# ---------------------------------------------------------------------------
CONTAINER_IDS=()

cleanup() {
  echo ""
  echo "==> Stopping simulated agents..."
  for cid in "${CONTAINER_IDS[@]}"; do
    docker stop "${cid}" 2>/dev/null || true
    docker rm   "${cid}" 2>/dev/null || true
  done
  echo "==> All agents stopped."
}

trap cleanup EXIT INT TERM

# ---------------------------------------------------------------------------
# Start NUM_AGENTS containers.
# ---------------------------------------------------------------------------
for i in $(seq 1 "${NUM_AGENTS}"); do
  DEVICE_ID="$(printf 'N-%04d' "${i}")"
  CONTAINER_NAME="nucleus-sim-agent-${DEVICE_ID,,}"

  echo "==> Starting agent ${DEVICE_ID} (container: ${CONTAINER_NAME})"

  CID=$(docker run -d \
    --name "${CONTAINER_NAME}" \
    --rm \
    -e DEVICE_ID="${DEVICE_ID}" \
    -e CONTROL_PLANE_URL="${API_URL}" \
    -e AGENT_SECRET="${AGENT_SECRET}" \
    -e TENANT_ID="${TENANT_ID}" \
    -e LOG_LEVEL="debug" \
    -e HEARTBEAT_INTERVAL="10s" \
    -e INVENTORY_SCAN_INTERVAL="30s" \
    --network host \
    "${IMAGE_TAG}"
  )

  CONTAINER_IDS+=("${CID}")
  echo "    Container ID: ${CID:0:12}"
done

echo ""
echo "==> ${NUM_AGENTS} agent(s) running. Control plane: ${API_URL}"
echo "==> Press Ctrl+C to stop all agents."
echo ""

# ---------------------------------------------------------------------------
# Tail logs from all containers.
# ---------------------------------------------------------------------------
# Build a list of container names for docker logs tailing.
for i in $(seq 1 "${NUM_AGENTS}"); do
  DEVICE_ID="$(printf 'N-%04d' "${i}")"
  CONTAINER_NAME="nucleus-sim-agent-${DEVICE_ID,,}"
  docker logs -f "${CONTAINER_NAME}" --since 0s 2>&1 | sed "s/^/[${DEVICE_ID}] /" &
done

# Wait for any log tail process to exit (which means a container died).
wait
