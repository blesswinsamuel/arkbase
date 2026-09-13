#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PGDATA="${REPO_DIR}/data/postgres"
SOCKET_DIR="${REPO_DIR}/data/postgres_socket"

mkdir -p "${SOCKET_DIR}"

if [ ! -f "${PGDATA}/PG_VERSION" ]; then
  echo "=> Initializing new PostgreSQL 17 cluster in ${PGDATA}..."
  mkdir -p "${PGDATA}"
  initdb -D "${PGDATA}" --auth=trust -U postgres --no-locale --encoding=UTF8

  echo "=> Starting temporary PostgreSQL instance for initial database seeding..."
  postgres -D "${PGDATA}" -k "${SOCKET_DIR}" -p 5432 &
  PG_PID=$!

  echo "=> Waiting for PostgreSQL to be ready..."
  until pg_isready -h "${SOCKET_DIR}" -p 5432 -U postgres >/dev/null 2>&1; do
    sleep 0.2
  done

  echo "=> Creating dev_db database..."
  psql -h "${SOCKET_DIR}" -p 5432 -U postgres -c "CREATE DATABASE dev_db;"

  if [ -f "${REPO_DIR}/testdata/seed.sql" ]; then
    echo "=> Seeding dev_db with ${REPO_DIR}/testdata/seed.sql..."
    psql -h "${SOCKET_DIR}" -p 5432 -U postgres -d dev_db -f "${REPO_DIR}/testdata/seed.sql"
  fi

  echo "=> Seeding complete. Stopping temporary instance..."
  kill "${PG_PID}"
  wait "${PG_PID}" 2>/dev/null || true
  echo "=> Cluster initialized successfully!"
fi

echo "=> Starting PostgreSQL on localhost:5432..."
exec postgres -D "${PGDATA}" -k "${SOCKET_DIR}" -p 5432
