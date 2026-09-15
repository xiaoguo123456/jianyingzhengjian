> 部署与新增集成以 [部署说明](deploy/README.md) 为准：已加入 NewAPI 图像编辑、阿里云 OSS 和版本化 SQL 迁移。

# 映己 backend (Go)

Gin + GORM (MySQL 8) + Redis/Asynq. Architecture: `../docs/BACKEND_ARCHITECTURE.md`; API contract: `../docs/API.md`;
credit and pipeline rules: `../docs/GENERATION_PIPELINE.md`.

## Run locally

Needs MySQL and Redis on localhost. Defaults are in `.env.example`.

```bash
cp .env.example .env            # set MYSQL_DSN if your MySQL has a password
go run ./cmd/migrate up         # schema (GORM AutoMigrate in V1, see migrations/README.md)
go run ./cmd/migrate seed       # catalogue, runtime config, admin user, sample assets
go run ./cmd/api                # http://localhost:8080
go run ./cmd/worker             # second terminal
```

If port 8080 is taken, set `HTTP_ADDR=:8090` **and** `PUBLIC_BASE_URL=http://localhost:8090` (signed
URLs for local storage are built from it), and point the client at it with `VITE_API_BASE`.

Dev defaults use the `mock` face/matting/gen providers and local disk storage served from `/files/...`,
so the whole journey works offline: login (`provider: h5_dev`), upload, generate, poll, download,
recolor, share.

### Smoke test

```bash
API=http://localhost:8090 bash scripts/smoke.sh
```

Runs the full journey against a running api + worker: login, privacy consent, catalogue, upload with
photo check, free ID photo task, download, free recolor, a credit-consuming template task, the
NO_CREDITS gate, share creation, works summary and admin stats. It is deterministic: it sets the test
user's balance through the admin API before the credit-gate step.

### Tests

```bash
go test ./...                   # unit tests; database tests skip themselves
TEST_MYSQL=1 go test ./...      # adds the credit/task matrix against MySQL (database `yingji_test`)
```

`TEST_MYSQL_DSN` overrides the server and base database name. Each package gets its own database
(`yingji_test_credit`, `yingji_test_task`, …) created on demand, because `go test ./...` runs packages
in parallel and they would otherwise truncate each other's tables. Covered: the credit ledger matrix from
`../docs/GENERATION_PIPELINE.md` §9 (daily grant and reset, bucket split, idempotent consume and
refund, ad cap, share reward, admin adjust), task creation (free vs gen, idempotency, breaker,
timeout, refund-missing, ownership), ID photo crop geometry, AIGC metadata, and the matting keyer.

## Layout

```
cmd/              api, worker, migrate
internal/app      wiring of config, db, redis, storage, providers, engines, services
internal/domain   models, enums and recipes (GenConfig, CropRule)
internal/engine   local (imaging), vision (face/matting), genmodel (router: capabilities, breakers, fallback)
internal/pipeline idphoto, template, poster, shared steps
internal/provider storage (local, cos), wechat, face, matting, genmodel (mock, volcengine), tencentcloud (TC3 signer)
internal/service  credit (ledger), auth, ads, catalogue, photo, task, work, favorite, share, event, notify, profile, admin
internal/transport http (public, admin, hooks, middleware, dto), queue (asynq handlers + scheduler)
internal/seed     initial data — spec sizes marked 示例数据 must be verified before production
internal/testutil MySQL test harness
testdata/         fixtures for the smoke test (upscaled and sharpened from the mockup crops)
```

## Providers

| Env | Dev default | Production |
|---|---|---|
| `FACE_PROVIDER` | `mock` | `tencent` (iai DetectFace/CompareFace, bda SegmentPortraitPic) |
| `GEN_PROVIDER_DEFAULT` | `mock` | `volcengine` (Ark Seedream) |
| `STORAGE_DRIVER` | `local` | `cos` |

Only the `mock` providers and local storage have been exercised end to end. The Tencent, Volcengine,
COS and WeChat adapters compile but have never been called against the live APIs — treat their request
shapes as unverified and check them against current provider documentation before switching over.

The `mock` matting provider is a background-colour keyer (border-seeded flood fill with a local
gradient constraint), not a segmentation model. It works on ID-photo-style shots with an even backdrop
and falls back to keeping the whole frame when keying fails. Production uses a real segmentation API.

`POSTER_FONT_PATH` must point at a CJK font (`.ttf`, `.otf` or `.ttc`) for the visible "AI生成" label
and poster text. On macOS, `/System/Library/Fonts/Supplemental/Songti.ttc` works. Without it the label
falls back to a text-free marker, which does not satisfy the labelling rules in `../docs/COMPLIANCE.md`.

## Admin

`POST /admin/v1/auth/login` with `ADMIN_INIT_USER` / `ADMIN_INIT_PASSWORD` (seeded on first run).
Resources: `categories`, `specs`, `templates`, `collections`, `banners` via `GET/POST /admin/v1/{resource}`
and `DELETE /admin/v1/{resource}/{id}`. Template writes are validated against provider capabilities and
the banned-word list before they are saved.
