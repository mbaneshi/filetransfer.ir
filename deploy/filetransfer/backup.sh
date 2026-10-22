#!/usr/bin/env bash
# Postgres dump for filetransfer.ir. Runs before every deploy and nightly from /etc/cron.d. Keeps 14 days.
# A dump only counts once pg_dump says it finished; a failed dump must never be mistaken for a backup.
set -euo pipefail
D="${FT_BACKUP_DIR:-/opt/filetransfer/backups}"; T=$(date +%Y%m%d-%H%M%S)
PART="$D/pg-$T.sql.gz.part"
trap 'rm -f "$PART"' EXIT
docker exec ft-postgres pg_dump -U filetransfer --clean --if-exists filetransfer | gzip -9 > "$PART"
gunzip -c "$PART" | tail -n 5 | grep -q "PostgreSQL database dump complete" || { echo "backup FAILED: dump is incomplete" >&2; exit 1; }
mv "$PART" "$D/pg-$T.sql.gz"
chmod 600 "$D"/pg-*.sql.gz
find "$D" -type f -name "pg-*.sql.gz" -mtime +14 -delete
echo "backup ok $T: $(find "$D" -name 'pg-*.sql.gz' | wc -l | tr -d ' ') dumps, $(du -sh "$D" | cut -f1)"
