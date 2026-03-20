#!/usr/bin/env bash
# run_migrations.sh
# Applies all migration files in order against the database specified by $DATABASE_URL.
#
# Usage:
#   DATABASE_URL="postgres://user:password@host:5432/dbname" ./run_migrations.sh
#
# The script will abort on the first failure (set -e).
# All migrations are idempotent (CREATE TABLE IF NOT EXISTS, CREATE OR REPLACE FUNCTION, etc.)
# so they are safe to re-run.

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
# Ordered list of migration files
# ---------------------------------------------------------------------------
MIGRATIONS=(
    "001_initial_schema.sql"
    "002_session_functions.sql"
)

# ---------------------------------------------------------------------------
# Apply each migration
# ---------------------------------------------------------------------------
echo "[INFO] Starting migration run against: ${DATABASE_URL}"
echo ""

for migration in "${MIGRATIONS[@]}"; do
    migration_path="${SCRIPT_DIR}/${migration}"

    if [[ ! -f "${migration_path}" ]]; then
        echo "[ERROR] Migration file not found: ${migration_path}" >&2
        exit 1
    fi

    echo "[INFO] Applying migration: ${migration}"
    psql "${DATABASE_URL}" \
        --single-transaction \
        --set ON_ERROR_STOP=1 \
        --file "${migration_path}"
    echo "[OK]   ${migration} applied successfully."
    echo ""
done

echo "[INFO] All migrations applied successfully."
