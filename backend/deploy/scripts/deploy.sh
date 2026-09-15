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
cp "$release/compose.release.yml" compose.release.yml
mkdir -p scripts
cp "$release/rollback.sh" scripts/rollback.sh
export APP_IMAGE="$image"
compose=(docker compose --env-file .env -f compose.release.yml)
rollback(){
  echo '发布失败，恢复上一应用版本'
  if [[ -f "$release/compose.before.yml" ]]; then cp "$release/compose.before.yml" compose.release.yml; fi
  if [[ -n "$previous" ]]; then
    export APP_IMAGE="${previous#APP_IMAGE=}"
    "${compose[@]}" up -d --no-deps api worker
  else
    "${compose[@]}" stop api worker || true
  fi
}
trap rollback ERR
"${compose[@]}" up -d --wait --wait-timeout 300 mysql redis
"${compose[@]}" run --rm migrate up
if [[ ! -f .initialized ]]; then
  "${compose[@]}" run --rm migrate seed
  touch .initialized
fi
"${compose[@]}" up -d --wait --wait-timeout 180 api worker
# 就绪与工作进程心跳均通过后，才记录当前版本。
[[ -z "$previous" ]] || printf '%s\n' "$previous" > previous.env
printf 'APP_IMAGE=%s\n' "$image" > current.env
printf '%s\n' "$commit" > current.sha
trap - ERR
echo "发布完成：$commit"
