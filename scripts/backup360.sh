#!/usr/bin/env bash
# Бэкап пилота 360: PostgreSQL + каталог фото. Требует DB_DSN и FILES_DIR в окружении.
set -euo pipefail
: "${DB_DSN:?нужен DB_DSN}"
: "${FILES_DIR:?нужен FILES_DIR}"
OUT_DIR="${BACKUP_DIR:-/var/backups}"
STAMP=$(date -u +%F_%H%M)
mkdir -p "$OUT_DIR"
pg_dump "$DB_DSN" > "$OUT_DIR/proakt360_${STAMP}.sql"
tar -czf "$OUT_DIR/proakt360_files_${STAMP}.tgz" -C "$FILES_DIR" .
echo "OK: $OUT_DIR/proakt360_${STAMP}.sql + proakt360_files_${STAMP}.tgz"
