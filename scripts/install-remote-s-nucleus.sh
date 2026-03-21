#!/usr/bin/env bash

set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/JuanM2209/agente-rs-platform.git}"
GIT_REF="${GIT_REF:-main}"
WORKDIR="${WORKDIR:-/tmp/agente-rs-platform}"
ARCHIVE_PATH="${ARCHIVE_PATH:-/tmp/agente-rs-platform-main.tar.gz}"
ARCHIVE_URL="${ARCHIVE_URL:-https://github.com/JuanM2209/agente-rs-platform/archive/refs/heads/${GIT_REF}.tar.gz}"
IMAGE_NAME="${IMAGE_NAME:-remote-s-local}"
PREBUILT_IMAGE_NAME="${PREBUILT_IMAGE_NAME:-remote-s-prebuilt:armv7}"
PREBUILT_TAG="${PREBUILT_TAG:-legacy-armv7-20260320}"
PREBUILT_ASSET="${PREBUILT_ASSET:-remote-s-armv7-image.tar.gz}"
PREBUILT_TAR_URL="${PREBUILT_TAR_URL:-https://github.com/JuanM2209/agente-rs-platform/releases/download/${PREBUILT_TAG}/${PREBUILT_ASSET}}"
CONTAINER_NAME="${CONTAINER_NAME:-Remote-S}"
FACTORY_PATH="${FACTORY_PATH:-/data/nucleus/factory}"
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-}"
AGENT_SECRET="${AGENT_SECRET:-}"
TENANT_ID="${TENANT_ID:-test-org}"
LOG_LEVEL="${LOG_LEVEL:-info}"
HEARTBEAT_INTERVAL="${HEARTBEAT_INTERVAL:-30s}"
INVENTORY_SCAN_INTERVAL="${INVENTORY_SCAN_INTERVAL:-60s}"
SERIAL_DEVICE="${SERIAL_DEVICE:-/dev/ttymxc5}"
MBUSD_HOST_PATH="${MBUSD_HOST_PATH:-}"
KEEP_SOURCE="${KEEP_SOURCE:-false}"
DISABLE_CONTENT_TRUST="${DISABLE_CONTENT_TRUST:-true}"
FORCE_PREBUILT_IMAGE="${FORCE_PREBUILT_IMAGE:-false}"

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

if [[ "${AGENT_SECRET}" = "TU_SECRET_REAL" || "${AGENT_SECRET}" = "YOUR_AGENT_SECRET" || "${AGENT_SECRET}" = "replace-with-real-agent-secret" ]]; then
  echo "[ERROR] AGENT_SECRET is still using a placeholder value. Use the real AGENT_WS_SECRET configured on the control plane." >&2
  exit 1
fi

if [[ ! "${TENANT_ID}" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
  echo "[ERROR] TENANT_ID must be a real tenant UUID. Example for Alpha in this environment: a1000000-0000-0000-0000-000000000001" >&2
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

download_source() {
  rm -rf "${WORKDIR}" "${ARCHIVE_PATH}" "${WORKDIR}-main"

  if command -v git >/dev/null 2>&1; then
    echo "[INFO] Cloning ${REPO_URL} (${GIT_REF}) into ${WORKDIR}"
    git clone --depth 1 --branch "${GIT_REF}" "${REPO_URL}" "${WORKDIR}"
    return
  fi

  if ! command -v curl >/dev/null 2>&1; then
    echo "[ERROR] Neither git nor curl is available to download the source." >&2
    exit 1
  fi

  echo "[INFO] Downloading source archive ${ARCHIVE_URL}"
  curl -fsSL "${ARCHIVE_URL}" -o "${ARCHIVE_PATH}"
  tar -xzf "${ARCHIVE_PATH}" -C /tmp
  mv "${WORKDIR}-main" "${WORKDIR}"
}

load_prebuilt_image() {
  if ! command -v curl >/dev/null 2>&1; then
    echo "[ERROR] curl is required to download the prebuilt Remote-S image." >&2
    exit 1
  fi

  if ! command -v tar >/dev/null 2>&1; then
    echo "[ERROR] tar is required to unpack the prebuilt Remote-S image." >&2
    exit 1
  fi

  local temp_dir
  local archive_path
  local extract_dir
  local inner_tar
  local repacked_tar

  temp_dir="$(mktemp -d /tmp/remote-s-prebuilt.XXXXXX)"
  archive_path="${temp_dir}/${PREBUILT_ASSET}"
  extract_dir="${temp_dir}/extract"
  repacked_tar="${temp_dir}/docker-archive.tar"
  mkdir -p "${extract_dir}"

  echo "[INFO] Loading prebuilt image ${PREBUILT_IMAGE_NAME} from ${PREBUILT_TAR_URL}"
  curl -fsSL "${PREBUILT_TAR_URL}" -o "${archive_path}"

  if tar -xzf "${archive_path}" -C "${extract_dir}" >/dev/null 2>&1; then
    inner_tar="$(find "${extract_dir}" -maxdepth 2 -type f -name '*.tar' | head -n 1 || true)"
    if [[ -n "${inner_tar}" ]]; then
      docker load -i "${inner_tar}"
    else
      tar -cf "${repacked_tar}" -C "${extract_dir}" .
      docker load -i "${repacked_tar}"
    fi
  else
    docker load -i "${archive_path}"
  fi

  rm -rf "${temp_dir}"
  IMAGE_NAME="${PREBUILT_IMAGE_NAME}"
}

build_local_image() {
  download_source

  echo "[INFO] Building ${IMAGE_NAME} locally with classic Docker compatibility"
  if [[ "${DISABLE_CONTENT_TRUST}" = "true" ]]; then
    echo "[INFO] Disabling Docker Content Trust for this legacy build path"
    DOCKER_CONTENT_TRUST=0 DOCKER_BUILDKIT=0 docker build --disable-content-trust=true -t "${IMAGE_NAME}" "${WORKDIR}/apps/agent"
  else
    DOCKER_BUILDKIT=0 docker build -t "${IMAGE_NAME}" "${WORKDIR}/apps/agent"
  fi
}

if [[ "${FORCE_PREBUILT_IMAGE}" = "true" ]]; then
  load_prebuilt_image
else
  if ! build_local_image; then
    echo "[WARN] Local build failed on this Nucleus. Falling back to the prebuilt ARMv7 Remote-S image." >&2
    load_prebuilt_image
  fi
fi

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

if [[ -n "${SERIAL_DEVICE}" && -e "${SERIAL_DEVICE}" ]]; then
  args+=(--device "${SERIAL_DEVICE}:${SERIAL_DEVICE}")
fi

if [[ -n "${MBUSD_HOST_PATH}" ]]; then
  if [[ ! -f "${MBUSD_HOST_PATH}" ]]; then
    echo "[ERROR] MBUSD_HOST_PATH does not exist: ${MBUSD_HOST_PATH}" >&2
    exit 1
  fi

  args+=(-v "${MBUSD_HOST_PATH}:/usr/local/bin/mbusd:ro")
fi

args+=("${IMAGE_NAME}")

echo "[INFO] Starting container ${CONTAINER_NAME}"
docker "${args[@]}"

if [[ "${KEEP_SOURCE}" != "true" ]]; then
  rm -rf "${WORKDIR}" "${ARCHIVE_PATH}" "${WORKDIR}-main"
fi

echo "[OK] ${CONTAINER_NAME} started."
echo "[INFO] Check status with: docker ps"
echo "[INFO] Check logs with: docker logs -f ${CONTAINER_NAME}"
if [[ -n "${SERIAL_DEVICE}" && -e "${SERIAL_DEVICE}" ]]; then
  echo "[INFO] Serial device mapped: ${SERIAL_DEVICE}"
  echo "[WARN] If MBUSD is activated from the portal on ${SERIAL_DEVICE}, Node-RED Modbus serial communication on the same port will be interrupted until the bridge stops."
fi
if [[ -z "${MBUSD_HOST_PATH}" ]]; then
  echo "[INFO] Using bundled MBUSD when the image is running on an ARMv7 Nucleus device."
else
  echo "[INFO] Using host-provided MBUSD from ${MBUSD_HOST_PATH}."
fi
if [[ "${DISABLE_CONTENT_TRUST}" = "true" ]]; then
  echo "[INFO] Docker Content Trust was disabled for the local build to avoid old-engine signature issues."
fi
