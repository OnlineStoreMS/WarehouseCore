#!/usr/bin/env bash
# 在已有 Postgres 上创建 warehousecore 角色与库（幂等）
set -euo pipefail
HOST="${POSTGRES_HOST:-127.0.0.1}"
PORT="${POSTGRES_PORT:-5432}"
ADMIN_USER="${POSTGRES_USER:-postgres}"
PASS="${WAREHOUSECORE_DB_PASSWORD:-warehousecore}"

psql -h "$HOST" -p "$PORT" -U "$ADMIN_USER" -d postgres <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'warehousecore') THEN
    CREATE ROLE warehousecore LOGIN PASSWORD '${PASS}';
  END IF;
END
\$\$;
SELECT 'CREATE DATABASE warehousecore OWNER warehousecore'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'warehousecore')\gexec
GRANT ALL PRIVILEGES ON DATABASE warehousecore TO warehousecore;
SQL
echo "warehousecore database ready"
