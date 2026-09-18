# Deployment and Operations

Operational details (server addresses, first-time setup, gateway snippets, rollback commands) live in [backend/deploy/README.md](../backend/deploy/README.md). This document describes the shape of the system and the rules; the deploy README is the runbook.

## 1. Environments

| Env | `APP_ENV` | Purpose | Mini Program build | API base URL | Gen model | Vision | Storage |
|---|---|---|---|---|---|---|---|
| `dev` | `dev` | local development | `pnpm dev:mp-weixin` / `dev:h5` (DevTools "不校验合法域名" on) | `http://localhost:8080` | `mock` (or `newapi` with a key) | `mock` | local disk, served at `/files` |
| test | `staging` | QA, review submission | `pnpm build:test` → 体验版 (trial) | `https://test-www.qhzhiyin.com/yingji` | `newapi` | `mock` allowed | OSS, prefix `yingji/test` |
| production | `prod` | users | `pnpm build:prod` → 正式版 (release) | `https://platform.qhzhiyin.com/yingji` | `newapi` | `tencent`, or `disabled` until credentials exist | OSS, prefix `yingji/production` |

The client picks its API base URL at build time from `VITE_APP_ENV` (`client/src/config/index.ts`); `VITE_API_BASE` overrides it for local work. All hosts the Mini Program talks to must have ICP filing and valid TLS and be registered in the MP console: the API domain under request / uploadFile, and the OSS public endpoint domain under downloadFile (images are served as signed OSS URLs).

`config.Validate` refuses to start `APP_ENV=prod` with the `mock` gen model or `mock` vision, with a storage driver other than `oss`, or with signing secrets shorter than 32 characters or containing `change-me`. With `FACE_PROVIDER=disabled` photo processing is rejected, so the generation flow stays closed in production until Tencent vision credentials are configured.

## 2. Infrastructure

Everything runs on Alibaba Cloud (region `cn-beijing`), on hosts and data services the team already operates for other projects:

```
Internet ──▶ existing Nginx gateway (TLS) ── /yingji/ ──▶ api container (127.0.0.1:<API_PORT>)
                                                          worker container (no port)
                                                               │
                         same VPC, internal endpoints ─────────┼──────────────────────────────┐
                                                               ▼                              ▼
                               PostgreSQL 16 (existing RDS instance)          Redis (existing instance)
                               db yingji_test / yingji_prod                    DB 5 / DB 6
                               account yingji_test_app / yingji_prod_app       prefix yingji:test: / yingji:prod:
                                                               │
                                                               ▼
                               OSS bucket (shared, private) — prefix yingji/test / yingji/production
                               upload via the internal endpoint, sign via the public endpoint
```

- One host per environment, each in its own directory (`/opt/yingji-test`, `/opt/yingji-production`). Only the `api` and `worker` containers belong to this project; no database containers run on the servers.
- The gateway is shared. The test gateway belongs to another project, so every change there must keep the `/yingji/` route; production proxies `/yingji/` to `127.0.0.1:8014`. `/yingji/metrics` returns 404 at both gateways. Snippets: `backend/deploy/gateway/`.
- Shared-resource rules: never flush the shared Redis or delete keys outside this project's prefix; never change the shared bucket's ACL or policy; the RDS instance has SSL off, so connections stay on the VPC network with `sslmode=disable`.
- Sized for small shared hosts: each process opens at most 2 database connections with no idle pool (`DB_MAX_OPEN=2`, `DB_MAX_IDLE=0`); `GEN_CONCURRENCY=1`; `api` is limited to 128 MB / 0.35 CPU and `worker` to 384 MB / 0.5 CPU.

**Alternative, not in use:** 微信云托管 (WeChat Cloud Run) could host both containers and remove the ICP/domain work for `callContainer` traffic. The API keeps its own JWT auth so the code would be identical; only the client transport differs.

## 3. Containers

```
backend/deploy/
├── Dockerfile.release    # test/prod: golang:1.26 build → scratch; one image with api, worker, migrate, healthcheck + seed/assets; runs as 65532
├── compose.release.yml   # test/prod: api, worker, migrate (tools profile); read-only rootfs, cap_drop ALL, no-new-privileges, healthchecks
├── Dockerfile.api / Dockerfile.worker
├── docker-compose.yml    # local dev only: postgres:16, redis:7, api, worker with local storage
├── gateway/              # Nginx location blocks for the test and prod gateways
├── scripts/deploy.sh, scripts/rollback.sh
└── .env.example          # server .env template (test values)
```

The same image serves both roles: `api` is the default entrypoint, the worker overrides it with `/app/worker`. Health checks: `api` runs `/app/healthcheck` (calls `/readyz`, which pings PostgreSQL and Redis); `worker` runs `/app/healthcheck worker`, which asks the Asynq inspector whether this host has an active server.

Key environment variables (full list: `backend/.env.example`, `backend/deploy/.env.example`):

| Variable | Example | Notes |
|---|---|---|
| `DEPLOY_ENV` / `API_PORT` / `GATEWAY_NETWORK` | `test` / `8013` / `weishen-test_default` | compose only; prod uses `production` / `8014` / `weishen-prod_default` |
| `APP_ENV` | `staging`, `prod` | |
| `HTTP_ADDR` / `PUBLIC_BASE_URL` / `CORS_ORIGINS` | `:8080` / `https://test-www.qhzhiyin.com/yingji` | |
| `DATABASE_URL` | `postgres://yingji_test_app:***@<rds-internal>:5432/yingji_test?sslmode=disable` | URL-encode special characters in the password |
| `DB_MAX_OPEN` / `DB_MAX_IDLE` | `2` / `0` | per process |
| `REDIS_ADDR` / `REDIS_USERNAME` / `REDIS_PASSWORD` / `REDIS_DB` | `<redis-internal>:6379` / / / `5` | |
| `REDIS_PREFIX` | `yingji:test:` | must match `yingji:<env>:`; applied to cache keys, Asynq keys and channels |
| `JWT_SECRET_MP` / `JWT_SECRET_ADMIN` / `SIGN_SECRET` | | ≥ 32 random characters in prod |
| `WECHAT_APPID` / `WECHAT_SECRET` | | api and worker |
| `WECHAT_AD_CALLBACK_SECRET` / `WECHAT_REWARD_AD_UNIT_ID` / `WECHAT_SUBSCRIBE_TASK_FINISHED` | | ads and subscribe messages |
| `STORAGE_DRIVER` | `oss` | `local` in dev only |
| `OSS_REGION` / `OSS_BUCKET` / `OSS_PREFIX` | `cn-beijing` / / `yingji/test` | prefix is mandatory and must differ per environment |
| `OSS_ENDPOINT` / `OSS_PUBLIC_ENDPOINT` | `https://oss-cn-beijing-internal.aliyuncs.com` / `https://oss-cn-beijing.aliyuncs.com` | upload internally, sign publicly; HTTPS only |
| `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET` | | |
| `GEN_PROVIDER_DEFAULT` / `GEN_CONCURRENCY` | `newapi` / `1` | |
| `NEWAPI_BASE_URL` / `NEWAPI_MODEL` / `NEWAPI_API_KEY` | `https://www.ggwk1.online/v1` / `gpt-image-2.5` / | key lives only on the server |
| `FACE_PROVIDER` / `TENCENT_SECRET_ID` / `TENCENT_SECRET_KEY` / `TENCENT_REGION` | `mock` (test), `disabled` or `tencent` (prod) | |
| `LOG_LEVEL` | `info` | |
| `ADMIN_INIT_USER` / `ADMIN_INIT_PASSWORD` | | used by the first `migrate seed` only |
| `POSTER_FONT_PATH` | `/usr/share/fonts/.../NotoSansCJK.ttc` | required for the visible AI label and poster text; `.ttf`, `.otf` and `.ttc` collections are supported. Without it the label degrades to a marker with no text, which does not satisfy COMPLIANCE.md §3.2. The release image is built `FROM scratch` and ships no font yet: add one to the image (or mount it) and set this variable before opening generation in production |

Secrets live only in the server's `.env` (mode `600`, checked by `deploy.sh`), never in the repo. To change configuration, place the new file as `.env.next` (mode `600`); the next deploy switches to it and restores the previous `.env` if the deploy fails. Each release keeps its pre-deploy config as `releases/<sha>/env.before` (mode `600`). Locally, `.env.deploy.local` holds both environments' connection settings and is ignored by Git.

## 4. Database migrations

`deploy.sh` runs `migrate up` in a one-off container before the new `api`/`worker` start; a failed migration stops the release and the previous containers keep running. The first release in an environment also runs `migrate seed` and writes `.initialized-postgres`; later releases never seed, so templates edited through the admin API are not overwritten. Migrations are forward-only; see DATA_MODEL.md §5 for the file, checksum and lock rules. Rollback restores the application image only, so every migration must stay compatible with the previous image. After the switch from MySQL, the deploy script refuses to roll back across database engines; the old MySQL data directory is kept for reconciliation only.

## 5. CI/CD

GitHub Actions, backend only for now:

| Workflow | Trigger | What it does |
|---|---|---|
| `backend-check.yml` | PRs touching `backend/**` or workflows; called by `test.yml` | `go test ./...` and `go vet ./...` against PostgreSQL 16 and Redis 7 service containers; `migrate up` twice and `migrate seed` |
| `test.yml` | push to `main` touching `backend/**` or `.github/workflows/**`; manual | runs the check, then deploys that commit to test |
| `prod.yml` | manual, input: full 40-character SHA | verifies the SHA is on `main` and that `test.yml` succeeded for it, then deploys it to production |
| `deploy.yml` | called by `test.yml` / `prod.yml` | builds a `linux/amd64` image `yingji-backend:<sha>`, saves it as a gzip tarball with a SHA-256 file, copies it with the compose file and scripts over SSH (pinned `known_hosts`, key from repository secrets), and runs `deploy.sh <sha>` |

`deploy.sh` holds a lock, verifies the tarball checksum, loads the image, backs up `.env` and the compose file, applies `.env.next` if present, checks that `DATABASE_URL` and a `yingji:` Redis prefix are set, migrates, seeds once, then `docker compose up --wait` until both health checks pass. Only then does it record the release in `current.env` / `previous.env`; any failure restores the previous image and configuration. No container registry is involved. Deploys to one environment are serialised by a GitHub concurrency group.

Pushing to `main` deploys to test automatically; production is always a deliberate, manual step with an explicit commit.

**client (uni-app) — planned, not automated yet.** Today the Mini Program is built locally (`pnpm build:test` / `pnpm build:prod`) and uploaded from WeChat DevTools. Target pipeline: on every PR `vue-tsc --noEmit` and a grep gate that fails if `wx.` appears outside `src/platform/*.mp.ts`; on `main`, `build:test` plus `miniprogram-ci preview`; on tag, `build:prod` plus `miniprogram-ci upload`, after which a human sets 体验版 and submits for review. App builds (later) run on a separate pipeline.

**admin** — there is no admin console yet; catalogue and configuration are managed through the admin API (`/admin/v1/*`).

## 6. Observability and alerts

Current state: containers log JSON to Docker's `json-file` driver (10 MB × 3 files per container); `/metrics` is served inside the container but blocked at the gateway; no Prometheus, alerting or Asynqmon is deployed yet. The table below is the target once monitoring is connected.

| Signal | Threshold | Channel |
|---|---|---|
| API 5xx rate | > 1 % over 5 min | WeCom |
| p95 latency `POST /v1/photos` | > 4 s | WeCom |
| Task failure ratio | > 10 % over 15 min | WeCom + phone |
| Queue depth `generation` | > 100 | WeCom |
| Provider breaker open | > 5 min | WeCom + phone |
| Refunds per hour | > 3× 7-day average | WeCom |
| Consistency job found unrefunded failures | ≥ 1 | WeCom |
| Disk / DB connections / Redis memory | platform defaults | WeCom |

Dashboards: funnel (PRD §2 metrics), tasks per module, cost per day, ad claims accepted/rejected by reason, queue depth, provider latency.

## 7. Backups and recovery

- PostgreSQL: backups are handled by the existing RDS operations for the shared instance; confirm retention and run a restore drill for `yingji_prod` before launch. Application images are not a substitute for data backups.
- OSS: versioning off (cost); works are the only irreplaceable objects. If budget allows, replicate the `yingji/production/works/` prefix to another region.
- Redis: shared instance, persistence as configured by its owners; losing Redis loses queued jobs, which `task:requeue-stuck` recovers from PostgreSQL within a minute.
- Application: `scripts/rollback.sh` in the environment directory swaps back to the previous image and leaves the database unchanged.

## 8. Release checklist

- [ ] Commit deployed to test and verified there; production deploy started from `prod.yml` with that SHA.
- [ ] New migrations are additive and compatible with the previous image.
- [ ] Server `.env` reviewed (mode `600`): prefixes, OSS endpoints, `FACE_PROVIDER`, `POSTER_FONT_PATH`, secrets.
- [ ] `app_configs` reviewed: `ads_enabled`, credit numbers, `provider_prices`, retention.
- [ ] Spec seed data verified (UI-16) and marked with `source_note`.
- [ ] Domains whitelisted (API and OSS public endpoint); TLS valid > 30 days.
- [ ] Test gateway still routes `/yingji/` after any change by the gateway's owning project.
- [ ] Subscribe message template approved; ad unit id (if any) in config.
- [ ] COMPLIANCE.md launch checklist complete.
- [ ] Test-environment e2e flow green on iOS and Android with ads on and off.
- [ ] Rollback plan: `scripts/rollback.sh` target (previous image) known; no migration in this release blocks it.
- [ ] On-call owner and alert channels confirmed for launch week.
