# Deployment and Operations

## 1. Environments

| Env | Purpose | Mini Program version | API host | Ads | Provider |
|---|---|---|---|---|---|
| `dev` | local development | DevTools | `http://localhost:8080` (DevTools "不校验合法域名" on) | off | `mock` |
| `staging` | QA, review submission | 体验版 (trial) | `https://api-staging.<domain>` | off or test unit | real provider, low quota |
| `prod` | users | 正式版 (release) | `https://api.<domain>` | on once 流量主 approved | real provider |

The Mini Program picks the API host from `wx.getAccountInfoSync().miniProgram.envVersion` (`develop` / `trial` / `release`). All production hosts must have ICP filing and valid TLS; register them in the MP console under request / uploadFile / downloadFile.

## 2. Infrastructure (production)

Recommended on Tencent Cloud (same ecosystem as WeChat and COS):

```
Internet ──▶ CLB (TLS termination) ──▶ api ×2 (containers)
                                   └─▶ admin console (static, COS + CDN, path /admin)
                        worker ×2 (containers, no public port)
                        TencentDB for MySQL 8 (1 primary + 1 replica, daily backup, 7-day binlog)
                        TencentDB for Redis 7 (standard, AOF)
                        COS bucket (private) + CDN domain for the public assets/ and shares/ prefixes
                        CLS (logs) · Prometheus + Grafana (metrics) · alert to WeCom
```

Start on two small CVM instances or a Lighthouse pair with docker-compose if budget is tight; the compose file below works unchanged. Move to TKE when replicas exceed what one host can run.

**Alternative:** 微信云托管 (WeChat Cloud Run) can host both containers and removes ICP/domain work for `callContainer` traffic. The API keeps its own JWT auth so the code is identical; only the client transport differs. Evaluate if ICP filing is a blocker.

## 3. Containers

```
deploy/
├── Dockerfile.api        # multi-stage: golang:1.23 build → distroless
├── Dockerfile.worker
├── docker-compose.yml    # dev/staging: api, worker, mysql, redis, minio, asynqmon
└── .env.example
```

Key environment variables:

| Variable | Example | Notes |
|---|---|---|
| `APP_ENV` | `prod` | |
| `HTTP_ADDR` | `:8080` | |
| `MYSQL_DSN` | `user:pass@tcp(host:3306)/yingji?parseTime=true&loc=UTC` | |
| `REDIS_ADDR` / `REDIS_PASSWORD` | | |
| `JWT_SECRET_MP` / `JWT_SECRET_ADMIN` | | rotate via dual-secret grace |
| `WECHAT_APPID` / `WECHAT_SECRET` | | api and worker |
| `WECHAT_AD_CALLBACK_SECRET` | | webhook signature |
| `COS_BUCKET` / `COS_REGION` / `COS_SECRET_ID` / `COS_SECRET_KEY` / `COS_CDN_HOST` | | |
| `FACE_PROVIDER` / `TENCENT_IAI_SECRET_ID` / `..._KEY` | | worker |
| `GEN_PROVIDER_DEFAULT` / `SEEDREAM_API_KEY` / `WANX_API_KEY` / `HUNYUAN_*` | | worker; only providers referenced by templates need keys |
| `GEN_CONCURRENCY` | `4` | worker |
| `LOG_LEVEL` | `info` | |
| `POSTER_FONT_PATH` | `/usr/share/fonts/.../NotoSansCJK.ttc` | required for the visible AI label and poster text; `.ttf`, `.otf` and `.ttc` collections are supported. Without it the label degrades to a marker with no text, which does not satisfy COMPLIANCE.md §3.2 — ship a CJK font in the worker image |

Secrets come from the platform's secret store (or `.env` on a single host with `chmod 600`), never from the repo.

## 4. Database migrations

`cmd/migrate up` runs before the new `api`/`worker` start (init container or deploy script step). Migrations are forward-only in production; a failed migration halts the rollout. Schema changes that drop or rename columns are done in two releases (add → backfill → switch → drop).

## 5. CI/CD

GitHub Actions (or Coding.net) pipelines:

**backend**
1. `go vet`, `golangci-lint`, `go test ./...` (unit) on every PR.
2. Integration tests with testcontainers on `main`.
3. Build and push images tagged with the commit SHA; deploy to staging automatically; deploy to prod on a tagged release after a manual approval.

**client (uni-app)**
1. `vue-tsc --noEmit`, ESLint, Vitest, and a grep gate that fails if `wx.` appears outside `src/platform/*.mp.ts` on every PR.
2. On `main`: `npm run build:mp-weixin` with `VITE_APP_ENV=staging`, then `miniprogram-ci preview` on `dist/build/mp-weixin` posts a QR code to the PR / WeCom.
3. On tag: build with `VITE_APP_ENV=prod`, `miniprogram-ci upload` with the version and a changelog; a human sets it as 体验版 and submits for review.
4. App builds (later): `npm run build:app` produces the native project resources; packaging runs in HBuilderX cloud packaging or the offline SDK on a separate pipeline.

**admin**: static build uploaded to COS `admin/` and CDN purge.

## 6. Observability and alerts

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

Dashboards: funnel (PRD §2 metrics), tasks per module, cost per day, ad claims accepted/rejected by reason, queue depth, provider latency. Asynqmon is deployed internally for queue inspection.

## 7. Backups and recovery

- MySQL: automated daily snapshot + binlog, 7-day retention; restore drill once before launch.
- COS: versioning off (cost); works are the only irreplaceable objects and are covered by cross-region replication if budget allows.
- Redis: AOF; losing Redis loses queued jobs, which `task:requeue-stuck` recovers from MySQL within a minute.

## 8. Release checklist

- [ ] Migrations applied on staging and prod.
- [ ] `app_configs` reviewed: `ads_enabled`, credit numbers, provider, retention.
- [ ] Spec seed data verified (UI-16) and marked with `source_note`.
- [ ] Domains whitelisted; TLS valid > 30 days.
- [ ] Subscribe message template approved; ad unit id (if any) in config.
- [ ] COMPLIANCE.md launch checklist complete.
- [ ] Staging e2e flow green on iOS and Android with ads on and off.
- [ ] Rollback plan: previous image tag and `migrate down` tested for the release's migrations.
- [ ] On-call owner and alert channels confirmed for launch week.
