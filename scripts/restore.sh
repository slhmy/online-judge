#!/usr/bin/env bash
# Restore PostgreSQL, MinIO, and Redis data from a backup directory.
# Usage: ./scripts/restore.sh <backup_dir>

set -euo pipefail

BACKUP_DIR="${1:?Usage: $0 <backup_dir>}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yaml}"

if [ ! -d "${BACKUP_DIR}" ]; then
  echo "Error: backup directory '${BACKUP_DIR}' does not exist"
  exit 1
fi

echo "=== Online Judge Restore ==="
echo "Backup dir: ${BACKUP_DIR}"
echo "Compose   : ${COMPOSE_FILE}"
echo ""

read -rp "This will OVERWRITE current data. Continue? [y/N] " confirm
if [[ "${confirm}" != [yY] ]]; then
  echo "Aborted."
  exit 0
fi

# --------------------------------------------------
# 1. PostgreSQL
# --------------------------------------------------
if [ -f "${BACKUP_DIR}/postgres_oj.sql" ]; then
  echo "[1/3] Restoring PostgreSQL (oj)..."
  docker compose -f "${COMPOSE_FILE}" exec -T postgres \
    psql -U oj -d oj < "${BACKUP_DIR}/postgres_oj.sql"
  echo "  - oj database restored"
else
  echo "[1/3] No oj database dump found, skipped"
fi

if [ -f "${BACKUP_DIR}/postgres_identra.sql" ]; then
  echo "      Restoring PostgreSQL (identra)..."
  # Ensure identra database exists
  docker compose -f "${COMPOSE_FILE}" exec -T postgres \
    psql -U oj -c "CREATE DATABASE identra;" 2>/dev/null || true
  docker compose -f "${COMPOSE_FILE}" exec -T postgres \
    psql -U oj -d identra < "${BACKUP_DIR}/postgres_identra.sql"
  echo "  - identra database restored"
fi

# --------------------------------------------------
# 2. MinIO
# --------------------------------------------------
if [ -d "${BACKUP_DIR}/minio_data" ]; then
  echo "[2/3] Restoring MinIO..."
  MINIO_CONTAINER="$(docker compose -f "${COMPOSE_FILE}" ps -q minio 2>/dev/null || true)"
  if [ -n "${MINIO_CONTAINER}" ]; then
    docker compose -f "${COMPOSE_FILE}" stop minio
    docker run --rm \
      --volumes-from "${MINIO_CONTAINER}" \
      -v "$(cd "${BACKUP_DIR}" && pwd):/backup" \
      alpine:latest \
      sh -c "rm -rf /data/* && cp -a /backup/minio_data/. /data/"
    docker compose -f "${COMPOSE_FILE}" start minio
    echo "  - MinIO data restored"
  else
    echo "  - MinIO container not found, skipped"
  fi
else
  echo "[2/3] No MinIO backup found, skipped"
fi

# --------------------------------------------------
# 3. Redis
# --------------------------------------------------
if [ -f "${BACKUP_DIR}/redis_dump.rdb" ]; then
  echo "[3/3] Restoring Redis..."
  REDIS_CONTAINER="$(docker compose -f "${COMPOSE_FILE}" ps -q redis 2>/dev/null || true)"
  if [ -n "${REDIS_CONTAINER}" ]; then
    docker compose -f "${COMPOSE_FILE}" stop redis
    docker cp "${BACKUP_DIR}/redis_dump.rdb" "${REDIS_CONTAINER}:/data/dump.rdb"
    if [ -f "${BACKUP_DIR}/redis_appendonly.aof" ]; then
      docker cp "${BACKUP_DIR}/redis_appendonly.aof" "${REDIS_CONTAINER}:/data/appendonly.aof"
    fi
    docker compose -f "${COMPOSE_FILE}" start redis
    echo "  - Redis data restored"
  else
    echo "  - Redis container not found, skipped"
  fi
else
  echo "[3/3] No Redis backup found, skipped"
fi

echo ""
echo "=== Restore Complete ==="
