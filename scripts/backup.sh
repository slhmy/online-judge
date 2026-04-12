#!/usr/bin/env bash
# Backup PostgreSQL, MinIO, and Redis data from Docker Compose services.
# Usage: ./scripts/backup.sh [backup_dir]
#   backup_dir defaults to ./backups/<timestamp>

set -euo pipefail

BACKUP_ROOT="${1:-./backups}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_DIR="${BACKUP_ROOT}/${TIMESTAMP}"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yaml}"
PGUSER="${PGUSER:-postgres}"
PGDATABASE="${PGDATABASE:-oj}"

echo "=== Online Judge Backup ==="
echo "Timestamp : ${TIMESTAMP}"
echo "Backup dir: ${BACKUP_DIR}"
echo "Compose   : ${COMPOSE_FILE}"
echo ""

mkdir -p "${BACKUP_DIR}"

# --------------------------------------------------
# 1. PostgreSQL (oj + identra databases)
# --------------------------------------------------
echo "[1/3] Backing up PostgreSQL..."

docker compose -f "${COMPOSE_FILE}" exec -T postgres \
  pg_dump -U "${PGUSER}" -d "${PGDATABASE}" --no-owner --clean --if-exists \
  > "${BACKUP_DIR}/postgres_oj.sql"
echo "  - oj database dumped"

# identra database may not exist in dev setups; ignore errors
if docker compose -f "${COMPOSE_FILE}" exec -T postgres \
  pg_dump -U "${PGUSER}" -d identra --no-owner --clean --if-exists \
  > "${BACKUP_DIR}/postgres_identra.sql" 2>/dev/null; then
  echo "  - identra database dumped"
else
  rm -f "${BACKUP_DIR}/postgres_identra.sql"
  echo "  - identra database skipped (not found)"
fi

# --------------------------------------------------
# 2. MinIO (all buckets)
# --------------------------------------------------
echo "[2/3] Backing up MinIO..."

MINIO_CONTAINER="$(docker compose -f "${COMPOSE_FILE}" ps -q minio 2>/dev/null || true)"
if [ -n "${MINIO_CONTAINER}" ]; then
  # Use mc (MinIO Client) inside the minio container to mirror data out
  # Alternatively, copy the data volume directly
  docker run --rm \
    --volumes-from "${MINIO_CONTAINER}" \
    -v "$(cd "${BACKUP_DIR}" && pwd):/backup" \
    alpine:latest \
    sh -c "cp -a /data /backup/minio_data"
  echo "  - MinIO data copied"
else
  echo "  - MinIO container not running, skipped"
fi

# --------------------------------------------------
# 3. Redis (RDB snapshot)
# --------------------------------------------------
echo "[3/3] Backing up Redis..."

REDIS_CONTAINER="$(docker compose -f "${COMPOSE_FILE}" ps -q redis 2>/dev/null || true)"
if [ -n "${REDIS_CONTAINER}" ]; then
  # Trigger a synchronous save
  docker compose -f "${COMPOSE_FILE}" exec -T redis redis-cli BGSAVE >/dev/null 2>&1 || true
  sleep 2
  docker cp "${REDIS_CONTAINER}:/data/dump.rdb" "${BACKUP_DIR}/redis_dump.rdb" 2>/dev/null || true
  docker cp "${REDIS_CONTAINER}:/data/appendonly.aof" "${BACKUP_DIR}/redis_appendonly.aof" 2>/dev/null || true
  echo "  - Redis snapshot copied"
else
  echo "  - Redis container not running, skipped"
fi

# --------------------------------------------------
# Summary
# --------------------------------------------------
echo ""
echo "=== Backup Complete ==="
du -sh "${BACKUP_DIR}"/*
echo ""
echo "Location: ${BACKUP_DIR}"
