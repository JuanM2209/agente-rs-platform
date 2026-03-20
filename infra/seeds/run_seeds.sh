#!/usr/bin/env bash
# run_seeds.sh
# Applies all seed files in order against the database specified by $DATABASE_URL.
#
# Usage:
#   DATABASE_URL="postgres://user:password@host:5432/dbname" ./run_seeds.sh
#
# The script will abort on the first failure (set -e).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---------------------------------------------------------------------------
# Validate prerequisites
# ---------------------------------------------------------------------------
if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "[ERROR] DATABASE_URL is not set. Example:" >&2
    echo "  export DATABASE_URL=\"postgres://user:password@localhost:5432/nucleus\"" >&2
    exit 1
fi

if ! command -v psql &>/dev/null; then
    echo "[ERROR] psql is not installed or not in PATH." >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Ordered list of seed files
# ---------------------------------------------------------------------------
SEEDS=(
    "001_tenants.sql"
    "002_users.sql"
    "003_sites.sql"
    "004_devices.sql"
    "005_endpoints.sql"
    "006_sessions_history.sql"
)

# ---------------------------------------------------------------------------
# Apply each seed
# ---------------------------------------------------------------------------
echo "[INFO] Starting seed run against: ${DATABASE_URL}"
echo ""

for seed in "${SEEDS[@]}"; do
    seed_path="${SCRIPT_DIR}/${seed}"

    if [[ ! -f "${seed_path}" ]]; then
        echo "[ERROR] Seed file not found: ${seed_path}" >&2
        exit 1
    fi

    echo "[INFO] Applying seed: ${seed}"
    psql "${DATABASE_URL}" \
        --single-transaction \
        --set ON_ERROR_STOP=1 \
        --file "${seed_path}"
    echo "[OK]   ${seed} applied successfully."
    echo ""
done

echo "[INFO] All seeds applied successfully."
