#!/usr/bin/env bash
set -Eeuo pipefail
cd "$(dirname "$0")/.."
exec 9>.deploy.lock
flock -w 600 9
[[ -s previous.env ]] || { echo '没有上一版本'; exit 1; }
export "$(cat previous.env)"
docker compose --env-file .env -f compose.release.yml up -d --no-deps --wait --wait-timeout 180 api worker
cp current.env rollback-from.env
cp previous.env current.env
cp rollback-from.env previous.env
echo '应用已回滚，数据库保持不变'
