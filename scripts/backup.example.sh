#!/usr/bin/env bash
# MVP-1 manual backup (before `inv backup` exists).
# Usage: ./scripts/backup.example.sh [account_id] [lite|full]
set -euo pipefail

ACCOUNT_ID="${1:-${IA_ACCOUNT_ID:-default}}"
MODE="${2:-lite}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_ROOT="${DATA_ROOT:-${SCRIPT_DIR}/../data}"
BACKUP_ROOT="${BACKUP_ROOT:-${DATA_ROOT}/_backups}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
SRC="${DATA_ROOT}/accounts/${ACCOUNT_ID}"
DEST="${BACKUP_ROOT}/${ACCOUNT_ID}/${TIMESTAMP}"

if [[ ! -d "$SRC" ]]; then
  echo "Account path not found: $SRC" >&2
  exit 1
fi

mkdir -p "$DEST"

copy_if_exists() {
  local rel="$1"
  if [[ -d "${SRC}/${rel}" ]]; then
    mkdir -p "${DEST}/$(dirname "$rel")"
    cp -a "${SRC}/${rel}" "${DEST}/${rel}"
  fi
}

copy_if_exists "state"
copy_if_exists "db"
if [[ "$MODE" == "full" ]]; then
  copy_if_exists "library"
  copy_if_exists "reports"
fi

cat > "${DEST}/manifest.json" <<EOF
{
  "backup_id": "${TIMESTAMP}",
  "account_id": "${ACCOUNT_ID}",
  "mode": "${MODE}",
  "created_at": "$(date -Iseconds)",
  "data_root": "$(cd "$DATA_ROOT" && pwd)",
  "script": "backup.example.sh"
}
EOF

echo "Backup created: $DEST"
