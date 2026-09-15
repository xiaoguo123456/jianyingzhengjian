# Backend Architecture — Go

## 1. Stack (D-20)

| Concern | Choice | Why |
|---|---|---|
| Language | Go 1.23 | Requested; good fit for an I/O-bound API plus image workers |
| HTTP | Gin | Most widely used Go router in Chinese teams, mature middleware, easy hiring |
| ORM / SQL | GORM v2 + `golang-migrate` for schema | Fast CRUD for catalogue/admin; migrations are explicit SQL, never `AutoMigrate` in prod |
| Database | MySQL 8.0 | Ubiquitous on Tencent Cloud, team familiarity |
| Queue / cache | Redis 7 + Asynq | Asynq gives retries, timeouts, uniqueness, scheduled jobs and a web UI with zero extra infrastructure |
| Object storage | Tencent COS (`cos-go-sdk-v5`), S3-compatible interface so MinIO works locally | Same cloud as the WeChat ecosystem, image-processing URLs built in |
| Config | Environment variables (`caarlos0/env`) + `app_configs` table for runtime knobs | 12-factor; ops changes without redeploy |
| Logging | `log/slog` JSON to stdout | Standard library, structured |
| Metrics | Prometheus client | Standard |
| Auth | `golang-jwt/jwt/v5` | Standard |
| Validation | `go-playground/validator` via Gin binding | Standard |
| Imaging | `disintegration/imaging` for crop/resize/composite; `govips` optional later for speed | Pure Go first, no cgo in V1 |
| WeChat | `silenceper/wechat/v2` for access tokens, subscribe messages, content security | Maintained, covers the needed endpoints |
| Testing | `testing` + `testify`, `testcontainers-go` for MySQL/Redis integration tests | Standard |

## 2. Binaries

```
backend/
├── cmd/
│   ├── api/main.go        # HTTP server: public API, admin API, webhooks, health, metrics
│   ├── worker/main.go     # Asynq server: generation, notify, moderation, cleanup, scheduler
│   └── migrate/main.go    # golang-migrate wrapper (up/down/version)
```

`api` and `worker` share the same module and `internal/` packages; they differ only in `main`. Both are stateless and can run multiple replicas.

## 3. Package layout (modular monolith)

```
backend/
├── internal/
│   ├── config/            # env parsing, typed Config struct, runtime config loader (DB + Redis cache)
│   ├── domain/            # entities, value objects, enums, domain errors — no imports from other internal pkgs
│   │   ├── user.go credit.go task.go work.go photo.go template.go spec.go adsession.go
│   ├── service/           # use cases; one package per bounded context
│   │   ├── auth/          # code2session, JWT issue/verify
│   │   ├── credit/        # ledger, daily reset, consume/refund, ad reward (transactional core)
│   │   ├── ads/           # ad session create/claim rules
│   │   ├── catalogue/     # specs, templates, collections, home aggregation
│   │   ├── photo/         # upload, face check, retention
│   │   ├── task/          # create (credit + enqueue), query, state transitions
│   │   ├── work/          # works, download URLs, recolor
│   │   ├── favorite/ record/ event/ profile/ share/
│   │   └── admin/         # admin auth, CRUD, stats
│   ├── repo/              # persistence, one file per aggregate; GORM inside, interfaces in service pkgs
│   ├── transport/
│   │   ├── http/          # gin router, handlers, middleware (auth, request id, recovery, rate limit, CORS for admin)
│   │   │   ├── public/    # /v1/*
│   │   │   ├── admin/     # /admin/v1/*
│   │   │   └── hooks/     # /webhooks/*
│   │   └── asynq/         # task type registry, handlers, enqueue helpers
│   ├── engine/            # three engines behind one step API (GENERATION_PIPELINE.md §2)
│   │   ├── local/         # pure Go: orient, crop_spec, composite_solid_bg, square_crop, resize, export, label
│   │   ├── vision/        # detect_face, compare_face, matte — over provider/face and provider/matting
│   │   └── genmodel/      # gen.* ops: provider router, capability check, breaker, fallback, cost capture
│   ├── pipeline/          # compositions of engine steps, selected by task.kind
│   │   ├── idphoto/       # prepare → detect → [gen.edit] → crop_spec → matte → composite → export → label
│   │   ├── template/      # prepare → detect → gen → identity → post ops → export → label
│   │   ├── poster/        # share poster rendering (SHARING.md §5)
│   │   └── steps/         # step library and Step interface shared by pipelines
│   ├── provider/          # adapters to the outside world, each behind an interface defined here
│   │   ├── wechat/        # code2session, subscribe message, mediaCheckAsync, wxacode, ad callback verify
│   │   ├── face/          # FaceDetector + FaceComparer: tencent/ (iai DetectFace, CompareFace)
│   │   ├── matting/       # Matter: tencent/ (portrait segmentation)
│   │   ├── genmodel/      # GenModel: seedream/, wanx/, hunyuan/, mock/
│   │   └── storage/       # ObjectStore: cos/, s3/ (MinIO)
│   └── pkg/               # small shared utilities: apperr, idgen (ULID), clock, jwt, httpx, breaker
├── app/                   # wiring: config, db, redis, storage, providers, engines, services
├── testutil/              # MySQL-backed test harness (skips when TEST_MYSQL is unset)
├── seed/                  # initial catalogue, configs and admin user
├── migrations/            # see migrations/README.md: AutoMigrate in V1, SQL files from the first release
├── deploy/                # Dockerfile, docker-compose.yml, k8s or lighthouse scripts
└── Makefile
```

Dependency direction: `transport → service → repo/provider`, all via interfaces owned by the service packages. `domain` is imported by everyone and imports nothing internal.

## 4. Request lifecycle (api)

```
gin.Engine
 ├─ RequestID          (X-Request-Id passthrough or ULID)
 ├─ Recovery + slog access log (method, path, status, latency, user_id, request_id)
 ├─ Prometheus         (http_requests_total, http_request_duration_seconds)
 ├─ BodyLimit          (10 MB for /v1/photos, 256 KB elsewhere)
 ├─ Auth               (public: user JWT; admin: admin JWT; hooks: signature)
 ├─ RateLimit          (Redis token bucket per user: 60 rpm general, 10 rpm task create, 20 per day ad claim)
 └─ Handler → service → repo / provider
```

Handlers do binding, validation and envelope formatting only. Services own transactions.

### Response envelope

```json
{ "code": "OK", "message": "", "data": { … }, "request_id": "01J…" }
```

Errors use `apperr.Error{Code, HTTPStatus, Message}`; the full code list is in API.md.

## 5. Credit service — the transactional core

All balance changes happen in `credit.Service` inside a single MySQL transaction with `SELECT … FOR UPDATE` on `credit_accounts`:

```go
func (s *Service) Consume(ctx, tx, userID, ref Ref) (LedgerEntry, error)
func (s *Service) Refund(ctx, tx, userID, consumedEntryID) (LedgerEntry, error)
func (s *Service) GrantAdReward(ctx, tx, userID, adSessionID) (LedgerEntry, error)
func (s *Service) EnsureDailyGrant(ctx, tx, acct *Account, now time.Time)   // lazy reset, Asia/Shanghai
```

Invariants (enforced by unique keys in DATA_MODEL.md):

- One `consume` row per task, one `refund` row per task, one `ad_reward` row per ad session, one `daily_grant` per user per day.
- `daily_free_remaining ≥ 0`, `bonus_credits ≥ 0`.
- A task row is inserted in the same transaction as its `consume` row. The Asynq job is enqueued **after** commit; if enqueue fails, a scheduler job (`task:requeue-stuck`) picks up `waiting` tasks older than 1 minute.

## 6. Worker

Asynq task types:

| Type | Queue | Timeout | Retry | Unique |
|---|---|---|---|---|
| `generation:run` | `generation` (concurrency = `GEN_CONCURRENCY`, default 4) | 5 min | 1 (transient errors only) | per task id |
| `moderation:check` | `default` | 60 s | 3 | per object key |
| `notify:task-finished` | `default` | 30 s | 3 | per task id |
| `photo:cleanup` | `low` | 10 min | 0 | scheduled daily 04:00 |
| `task:requeue-stuck` | `low` | 1 min | 0 | scheduled every minute |
| `task:expire-processing` | `low` | 1 min | 0 | scheduled every minute; fails tasks past deadline and refunds |
| `stats:daily-rollup` | `low` | 10 min | 0 | scheduled daily 00:10 |
| `share:cleanup` | `low` | 10 min | 0 | scheduled daily 04:30; removes previews/posters past `share_preview_ttl_days` |

`generation:run` handler:

1. Load task; if status ≠ `waiting` return (idempotent).
2. Transition to `processing`, set `started_at`, `stage = processing`.
3. Select pipeline by `task.kind` and run with `ctx` deadline 4 min 30 s. Free tasks (`uses_genmodel=false`) run only `local` and `vision` steps.
4. Upload output to COS, create `works` row, set `stage = finishing`.
5. Enqueue `moderation:check` for the output; on `risky` the moderation handler flips the task to `failed/CONTENT_REJECTED` and refunds (D-15).
6. Transition to `success`, record `cost_cents`, `provider`, `provider_ref`.
7. Enqueue `notify:task-finished`.

Any error → `failed` with `error_code` + `refund` in one transaction. Retries happen only for `apperr.IsTransient(err)`.

## 7. Provider interfaces

```go
type FaceDetector interface {
    Detect(ctx context.Context, img []byte) (FaceResult, error) // faces, bbox, landmarks, quality, attrs
}
type FaceComparer interface {
    Compare(ctx context.Context, a, b []byte) (score float64, err error) // 0..1 similarity
}
type Matter interface {
    Matte(ctx context.Context, img []byte) (alpha []byte, err error) // PNG alpha mask, same size
}
type GenMode string // "img2img" | "reference" | "edit"

type GenCaps struct {
    Img2Img, Reference, Edit, MaskEdit bool
    MaxSide       int
    ReturnsAlpha  bool
}
type GenModel interface {
    Name() string
    Capabilities() GenCaps
    Run(ctx context.Context, req GenRequest) (GenResult, error)
}
type GenRequest struct {
    Mode           GenMode
    Source         []byte            // user photo (img2img / edit) or primary reference
    References     [][]byte          // extra reference images (reference mode)
    Mask           []byte            // optional PNG mask for edit
    Prompt         string
    NegativePrompt string
    Strength       float64           // img2img only
    Width, Height  int
    Seed           int64
    Model          string            // provider-specific
    Extra          map[string]any    // pass-through from template.gen_config
}
type GenResult struct { Image []byte; CostCents int; ProviderRef string; Seed int64; HasAlpha bool }
type ObjectStore interface {
    Put(ctx, key string, r io.Reader, size int64, contentType string) error
    Get(ctx, key string) (io.ReadCloser, error)
    SignedGetURL(ctx, key string, ttl time.Duration) (string, error)
    Delete(ctx, key string) error
}
```

Provider selection lives in `engine/genmodel`: `templates.gen_config.provider` → capability check for `mode` → `fallback_provider` → `config.default_provider`. A `mock` provider returns the input with a stamp and is used in dev and CI. Each provider is wrapped in a circuit breaker (`pkg/breaker`, 5 failures / 30 s open) and a per-provider concurrency semaphore. The `local` and `vision` engines expose the same `Step` interface so pipelines are plain sequences: `[]steps.Step{prepare, detect, genEdit(...), cropSpec, matte, composite, export, label}`.

## 8. Runtime configuration

`app_configs` rows (JSON value) cached in Redis for 60 s and in-process for 10 s:

| Key | Default | Used by |
|---|---|---|
| `ads_enabled` | `false` | client home payload, ad session create |
| `daily_free_credits` | `1` | credit daily grant |
| `daily_free_credits_no_ads` | `3` | credit daily grant when ads disabled |
| `ad_reward_daily_cap` | `10` | ad claim |
| `ad_session_min_seconds` | `10` | ad claim |
| `ad_session_ttl_minutes` | `30` | ad claim |
| `task_timeout_seconds` | `300` | worker |
| `photo_retention_days` | `30` | cleanup |
| `default_provider` | `seedream` | worker (gen model) |
| `provider_prices` | `{"seedream/seedream-4.0": 30}` (cents per image) | worker cost capture when the provider returns none |
| `identity_threshold` | `0.75` | worker identity check |
| `share_reward_enabled` | `true` | share incentive (D-22), on at launch |
| `share_reward_daily_cap` | `3` | share incentive |
| `share_preview_ttl_days` | `90` | share cleanup |
| `share_show_nickname` | `false` | share landing |
| `upload_max_bytes` | `10485760` | api |
| `photo_min_side_px` | `600` | photo check |
| `face_min_ratio` | `0.08` | photo check (face height / image height) |

## 9. Security

- JWT HS256 with 24 h expiry; secret rotated via env; `sub = user_id`, `aud = "mp"`. Admin tokens use `aud = "admin"` and a separate secret.
- All user-scoped queries filter by `user_id` from the token; no object id is trusted without an ownership check.
- Originals and works are private COS objects; the client only ever receives signed URLs with 10 min TTL.
- Uploads: sniff MIME (`http.DetectContentType`), accept `image/jpeg`, `image/png`, `image/heic` (converted server-side), reject others; strip EXIF from originals before storage.
- Webhooks verify WeChat signatures; unknown callbacks return 200 with no side effects to avoid retries storms, and are logged.
- Secrets never in the repo; `.env.example` lists names only.
- Admin API behind a separate hostname or path prefix with IP allow-list at the load balancer.

## 10. Observability

- Logs: JSON, one line per request and per task transition with `request_id`, `user_id`, `task_id`.
- Metrics: request latency/status, queue depth per queue, task duration per pipeline, provider latency/error rate, credits consumed/refunded per day, ad claims accepted/rejected per reason.
- Health: `GET /healthz` (process), `GET /readyz` (DB + Redis ping).
- Alerts (see DEPLOYMENT.md): failed-task ratio > 10 % over 15 min, queue depth > 100, provider breaker open > 5 min, refund count spike.

## 11. What is verified today

Built and exercised locally against MySQL 8 and Redis 7 with the `mock` providers and local storage:

| Area | Status |
|---|---|
| Login, profile, privacy consent | Verified end to end |
| Catalogue (home, specs, templates, collections) | Verified end to end |
| Upload + photo check (resolution, blur, luma, face count) | Verified, including rejections |
| ID photo pipeline (free path) | Verified: 295×413 PNG, correct DPI metadata |
| Template pipeline (gen path) | Verified: credit consumed, AI label burned in, AIGC metadata written |
| Free recolor | Verified across white / blue / red |
| Credit ledger, refunds, idempotency, daily reset | Unit-tested against MySQL (`internal/service/credit`, `internal/service/task`) |
| Crop geometry | Unit-tested (`internal/engine/local`) |
| Ad gating with ads disabled | Verified (409 on session create, no-ads daily grant) |
| Shares, works summary, admin stats and config | Verified end to end |
| Tencent vision, Volcengine gen model, COS storage | Compile only — never called against the live APIs |
| WeChat login, subscribe messages, mediaCheck, ad callback | Compile only — needs a real AppID |

`backend/scripts/smoke.sh` runs the whole user journey against a running api + worker and is
deterministic (it sets the test user's balance through the admin API before the credit-gate step).

## 12. Testing strategy

- `service/credit` and `service/task` have table-driven tests covering the matrix in GENERATION_PIPELINE.md §9 against a real MySQL via testcontainers.
- Pipelines are tested with fixture images and the `mock` providers; crop rules have golden-image tests.
- Handlers are tested with `httptest` and a fake service layer.
- A `make e2e` target runs api + worker + MinIO + MySQL + Redis in docker-compose and executes a scripted flow.
