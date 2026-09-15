#!/usr/bin/env bash
# 按提交发布固定镜像，应用失败回滚；数据库只允许向前迁移。
set -Eeuo pipefail
commit=${1:?需要完整提交 SHA}
[[ $commit =~ ^[0-9a-f]{40}$ ]] || exit 2
cd "${DEPLOY_ROOT:-$(dirname "$0")/..}"
exec 9>.deploy.lock
flock -w 600 9
[[ -f .env && $(stat -c %a .env) == 600 ]] || { echo '缺少权限为 600 的环境配置'; exit 1; }
release="$PWD/releases/$commit"
[[ -f "$release/image.tar.gz" && -f "$release/image.sha256" ]]
(cd "$release" && sha256sum -c image.sha256)
docker load -i "$release/image.tar.gz" >/dev/null
image="yingji-backend:$commit"
docker image inspect "$image" >/dev/null
previous=''
[[ ! -f current.env ]] || previous=$(cat current.env)
if [[ -f compose.release.yml ]]; then cp compose.release.yml "$release/compose.before.yml"; fi
cp .env "$release/env.before"
chmod 600 "$release/env.before"
transition=false
[[ -f .postgres-release || -z "$previous" ]] || transition=true
cp "$release/compose.release.yml" compose.release.yml
mkdir -p scripts
cp "$release/rollback.sh" scripts/rollback.sh
export APP_IMAGE="$image"
compose=(docker compose --env-file .env -f compose.release.yml)
rollback(){
  echo '发布失败，恢复上一应用版本与环境配置'
  cp "$release/env.before" .env
  chmod 600 .env
  if [[ -f "$release/compose.before.yml" ]]; then cp "$release/compose.before.yml" compose.release.yml; fi
  if [[ -n "$previous" ]]; then
    export APP_IMAGE="${previous#APP_IMAGE=}"
    "${compose[@]}" up -d --no-deps api worker
  else
    "${compose[@]}" stop api worker || true
  fi
}
trap rollback ERR
if [[ -f .env.next ]]; then
  [[ $(stat -c %a .env.next) == 600 ]]
  cp .env.next .env
fi
# 切换前确保环境已提供 PostgreSQL 和项目专属 Redis 前缀。
grep -q '^DATABASE_URL=' .env
grep -q '^REDIS_PREFIX=.*yingji:' .env
"${compose[@]}" run --rm migrate up
if [[ ! -f .initialized-postgres ]]; then
  "${compose[@]}" run --rm migrate seed
  touch .initialized-postgres
fi
"${compose[@]}" up -d --wait --wait-timeout 180 api worker
# 就绪与工作进程心跳均通过后，才记录当前版本。
if [[ "$transition" == true ]]; then
  # 旧 MySQL 版本不能使用新 PostgreSQL 配置，禁止普通镜像回滚跨数据库引擎。
  rm -f previous.env
elif [[ -n "$previous" ]]; then
  printf '%s\n' "$previous" > previous.env
fi
printf 'APP_IMAGE=%s\n' "$image" > current.env
printf '%s\n' "$commit" > current.sha
touch .postgres-release
rm -f .env.next
trap - ERR
echo "发布完成：$commit"
