# Backend Architecture — Go

## 1. Stack (D-20)

| Concern | Choice | Why |
|---|---|---|
| Language | Go 1.26 | Requested; good fit for an I/O-bound API plus image workers |
| HTTP | Gin | Most widely used Go router in Chinese teams, mature middleware, easy hiring |
| ORM / SQL | GORM v2 (`gorm.io/driver/postgres`, pgx) + versioned SQL embedded in the binary (`internal/migration`) | Fast CRUD for catalogue/admin; migrations are explicit SQL, never `AutoMigrate` in prod (DATA_MODEL.md §5) |
| Database | PostgreSQL 16 | Reuses the PostgreSQL 16 instance the team already runs on Alibaba Cloud; one database and one application account per environment |
| Queue / cache | Redis 7 + Asynq | Asynq gives retries, timeouts, uniqueness and scheduled jobs with zero extra infrastructure. The Redis instance is shared, so every key, Asynq queue key and pub/sub channel carries the `REDIS_PREFIX` namespace (`internal/pkg/redisx`) |
| Object storage | Alibaba Cloud OSS (`alibabacloud-oss-go-sdk-v2`); a local-disk driver for dev | Same cloud as the servers and database; internal endpoint for uploads, public endpoint for signed URLs. A COS driver still exists but no environment uses it |
| Gen model | NewAPI, an OpenAI-compatible gateway, using `/images/edits` with `gpt-image-2.5` | One HTTP integration reaches the image-edit model; provider routing keeps other models pluggable (§7) |
| Config | Environment variables (`caarlos0/env`) + `app_configs` table for runtime knobs | 12-factor; ops changes without redeploy |
| Logging | `log/slog` JSON to stdout | Standard library, structured |
| Metrics | Prometheus client | Standard |
| Auth | `golang-jwt/jwt/v5` | Standard |
| Validation | `go-playground/validator` via Gin binding | Standard |
| Imaging | `disintegration/imaging` for crop/resize/composite; `govips` optional later for speed | Pure Go first, no cgo in V1 |
| WeChat | Own small HTTP client in `provider/wechat` (no SDK): code2session, access token cached in Redis, subscribe messages, mediaCheckAsync, wxacode, ad callback signature | Only a handful of endpoints are needed; no third-party dependency |
| Testing | `testing` against a real PostgreSQL `_ci` database and a local Redis; each test gets its own schema / key prefix (§12) | No container runtime needed in CI beyond GitHub service containers |

## 2. Binaries

```
backend/
├── cmd/
│   ├── api/main.go        # HTTP server: public API, admin API, webhooks, health, metrics
│   ├── worker/main.go     # Asynq server: generation, notify, moderation, cleanup, scheduler
│   ├── migrate/main.go      # `up`: apply versioned PostgreSQL SQL; `seed`: initial catalogue, configs, admin
│   ├── healthcheck/main.go  # container health: api `/readyz`, or `worker` heartbeat via the Asynq inspector
│   ├── schema/main.go       # one-off: print DDL from the GORM models to draft a new migration file
│   └── storagecheck/main.go # acceptance check: put / get / sign / delete one object in the configured store
```

`api` and `worker` share the same module and `internal/` packages; they differ only in `main`. Both are stateless and can run multiple replicas.

## 3. Package layout (modular monolith)

```
backend/
├── internal/
│   ├── config/            # env parsing, typed Config struct, runtime config loader (DB + Redis cache)
│   ├── domain/            # entities, enums, error codes — no imports from other internal pkgs
│   │   ├── models.go      # one GORM model per table (DATA_MODEL.md)
│   │   ├── types.go       # modules, statuses, stages, task error codes and messages
│   │   └── genconfig.go   # template recipe (GENERATION_PIPELINE.md §7.3), crop rules, ID-photo params
│   ├── service/           # use cases; one package per bounded context
│   │   ├── auth/          # code2session, JWT issue/verify
│   │   ├── credit/        # ledger, daily reset, consume/refund, ad reward (transactional core)
│   │   ├── ads/           # ad session create/claim rules
│   │   ├── catalogue/     # specs, templates, collections, home aggregation
│   │   ├── photo/         # upload, face check, retention
│   │   ├── task/          # create (credit + enqueue), query, state transitions
│   │   ├── work/          # works, download URLs, recolor
│   │   ├── favorite/ event/ profile/ share/
│   │   ├── notify/        # subscribe message on task finish
│   │   └── admin/         # admin auth, CRUD, stats
│   ├── transport/
│   │   ├── http/          # gin router, middleware (request id, logging, recovery, CORS, auth, rate limit, body limit)
│   │   │   ├── public/    # /v1/*
│   │   │   ├── admin/     # /admin/v1/*
│   │   │   ├── hooks/     # /webhooks/*
│   │   │   └── dto/       # response shapes, signed URLs for catalogue images
│   │   └── queue/         # Asynq task types, enqueue helpers, handlers, scheduler
│   ├── engine/            # three engines behind one step API (GENERATION_PIPELINE.md §2)
│   │   ├── local/         # pure Go: decode/orient, crop to spec, solid-background composite, square crop, resize, encode, blur/luma checks, AI label + metadata
│   │   ├── vision/        # detect_face, compare_face, matte — over provider/face and provider/matting
│   │   └── genmodel/      # gen.* ops: provider router, capability check, breaker, fallback, cost capture
│   ├── pipeline/          # compositions of engine steps, selected by task.kind
│   │   ├── idphoto/       # prepare → detect → [gen.edit] → crop_spec → matte → composite → export → label
│   │   ├── template/      # prepare → detect → gen → identity → post ops → export → label
│   │   ├── poster/        # share poster rendering (SHARING.md §5)
│   │   └── steps/         # shared State, Deps and step helpers (prepare, detect, identity, finish)
│   ├── provider/          # adapters to the outside world, each behind an interface defined here
│   │   ├── wechat/        # code2session, subscribe message, mediaCheckAsync, wxacode, ad callback verify
│   │   ├── face/          # FaceDetector + FaceComparer: tencent/ (iai DetectFace, CompareFace)
│   │   ├── matting/       # Matter: tencent/ (portrait segmentation)
│   │   ├── genmodel/      # GenModel: newapi (default), volcengine, mock
│   │   └── storage/       # ObjectStore: oss (test/prod), local (dev, served at /files), cos (unused)
│   ├── pkg/               # small shared utilities: apperr, idgen (ULID), clock, jwt, httpx, breaker, redisx (key prefixing)
│   ├── migration/postgres/ # versioned SQL, embedded and applied by cmd/migrate
│   ├── app/               # wiring: config, db, redis, storage, providers, engines, services
│   ├── seed/              # initial catalogue, configs and admin user
│   └── testutil/          # PostgreSQL test harness (skips when TEST_DATABASE_URL is unset)
├── seed/assets/           # images loaded by `migrate seed`
├── migrations/            # README only: migration rules (DATA_MODEL.md §5)
├── deploy/                # release Dockerfile, compose files, gateway snippets, deploy/rollback scripts (DEPLOYMENT.md)
└── Makefile
```

Dependency direction: `transport → service → provider`. Services use GORM directly on `*gorm.DB` and own their transactions; there is no separate repository layer. Provider interfaces (`face.Detector`, `matting.Matter`, `genmodel.Model`, `storage.ObjectStore`) are defined in their provider packages and chosen once in `internal/app`. `domain` is imported by everyone and imports nothing internal.

## 4. Request lifecycle (api)

```
gin.Engine
 ├─ RequestID          (X-Request-Id passthrough, ≤ 64 chars, else a new ULID)
 ├─ Recovery + slog access log (method, path, status, ms, user_id, request_id; /healthz and /metrics skipped)
 ├─ CORS               (origins from CORS_ORIGINS, all routes)
 ├─ BodyLimit          (/v1/*: UPLOAD_MAX_BYTES + 1 MB, i.e. 11 MB by default)
 ├─ Auth               (/v1: user JWT; /admin/v1: admin JWT; hooks: see §9)
 ├─ RateLimit          (Redis fixed window per user, or per IP before login; see table)
 └─ Handler → service → provider
```

| Limit name | Routes | Limit |
|---|---|---|
| `login` | `POST /v1/auth/login` | 30 / min per IP |
| `general` | every authenticated `/v1` route | 120 / min |
| `task_create` | `POST /v1/tasks`, `POST /v1/tasks/:id/regenerate` (shared) | 10 / min |
| `upload` | `POST /v1/photos` | 30 / h |
| `ad_session` / `ad_claim` | `POST /v1/ads/sessions`, `…/claim` | 30 / h each |
| `admin_login` | `POST /admin/v1/auth/login` | 10 / min per IP |

A rejected request returns `RATE_LIMITED`. Rate-limit counters live under the Redis prefix like every other key.

Handlers do binding, validation and envelope formatting only. Services own transactions.

### Response envelope

```json
{ "code": "OK", "message": "", "data": { … }, "request_id": "01J…" }
```

Errors use `apperr.Error{Code, HTTPStatus, Message}`; the full code list is in API.md.

## 5. Credit service — the transactional core

All balance changes happen in `credit.Service` inside a single PostgreSQL transaction with `SELECT … FOR UPDATE` on `credit_accounts`:

```go
func (s *Service) EnsureAccount(ctx, tx *gorm.DB, userID string) (*domain.CreditAccount, error) // locks the row; lazy daily reset + grant, Asia/Shanghai
func (s *Service) Consume(ctx, tx *gorm.DB, userID string, n uint, refType, refID string) (*domain.CreditLedger, error) // daily first, then bonus
func (s *Service) Refund(ctx, tx *gorm.DB, consume *domain.CreditLedger) (*domain.CreditLedger, error)
func (s *Service) GrantAdReward(ctx, tx *gorm.DB, userID, sessionID string) (*domain.CreditLedger, error)
func (s *Service) GrantShareReward(ctx, tx *gorm.DB, sharerID, acquiredUserID string) (bool, error)
func (s *Service) AdminAdjust(ctx, userID string, delta int, note, opID string) (*domain.CreditLedger, error)
```

Invariants (enforced by unique keys in DATA_MODEL.md):

- One `consume` row per task, one `refund` row per task, one `ad_reward` row per ad session, one `daily_grant` per user per day.
- `daily_free_remaining ≥ 0`, `bonus_credits ≥ 0`.
- A task row is inserted in the same transaction as its `consume` row. The Asynq job is enqueued **after** commit; if enqueue fails, a scheduler job (`task:requeue-stuck`) picks up `waiting` tasks older than 1 minute.

## 6. Worker

Asynq task types:

| Type | Queue | Timeout | Max retry | Dedup / schedule |
|---|---|---|---|---|
| `generation:run` | `generation` | 5 min | 1 | Asynq task id `gen:<task id>`, kept 1 h after completion |
| `moderation:check` | `default` | 60 s | 3 | none |
| `notify:task-finished` | `default` | 30 s | 3 | none |
| `photo:cleanup` | `low` | 10 min | Asynq default (25) | daily 04:00 |
| `task:requeue-stuck` | `low` | 1 min | Asynq default (25) | every minute; re-enqueues `waiting` tasks older than 1 min |
| `task:expire-processing` | `low` | 1 min | Asynq default (25) | every minute; fails `processing` tasks older than `task_timeout_seconds` and refunds |
| `stats:daily-rollup` | `low` | 10 min | Asynq default (25) | daily 00:10 |
| `share:cleanup` | `low` | 10 min | Asynq default (25) | daily 04:30; removes previews/posters past `share_preview_ttl_days` |
| `consistency:refund` | `low` | 5 min | Asynq default (25) | every 30 min; refunds failed tasks that are missing a refund row |

Schedules use Asia/Shanghai time. The server runs `GEN_CONCURRENCY + 4` workers with queue weights `generation:6, default:3, low:1`; the gen-model router separately caps concurrent calls per provider at `GEN_CONCURRENCY` (default 4, 1 on the shared test/prod hosts).

`generation:run` handler:

1. Load the task and move it `waiting → processing` with one conditional update (`started_at`, `stage = processing`); if no row changed, it was already handled and the job returns.
2. Load the original photo and the spec or template.
3. Select the pipeline by `task.kind` and run it with a 4 min 30 s deadline. Free tasks (`uses_genmodel=false`) run only `local` and `vision` steps. Pipelines report `stage = finishing` themselves.
4. Upload the output, a thumbnail and, for ID photos, the alpha matte to object storage (OSS in test/prod).
5. In one transaction insert the `works` row and move the task to `success` with `work_id`, `cost_cents`, `provider`, `provider_ref`.
6. Enqueue `moderation:check` for the work thumbnail and `notify:task-finished`, and grant a pending share reward (D-22).

A pipeline or storage error calls `task.Fail`: `failed` with `error_code`, plus the `refund` ledger row, in one transaction; the job then returns `nil`, so Asynq does not retry it. Transient gen-model errors are retried once on the same provider and then on the fallback inside the router (GENERATION_PIPELINE.md §8); only database errors while starting or completing a task make Asynq retry the job.

Moderation runs after success. When WeChat is not configured or storage is local, the work is marked `pass` immediately. A `risky` callback marks the work `risky` (hidden from the user's lists and downloads) and revokes its shares. Known gap against D-15: `task.Fail` ignores tasks already in `success`, so the task is not flipped to `CONTENT_REJECTED` and the credit is not refunded.

## 7. Provider interfaces

```go
// provider/face
type Detector interface {
    Detect(ctx context.Context, img image.Image, raw []byte) (Result, error) // Result: Faces, Box, Quality, Gender, Occluded
}
type Comparer interface {
    Compare(ctx context.Context, a, b []byte) (float64, error) // 0..1 similarity of the primary faces
}
// provider/matting
type Matter interface {
    Matte(ctx context.Context, img image.Image, raw []byte, f face.Result) (*image.Alpha, error) // same size, 255 = person
}
// provider/genmodel
type Mode string // "img2img" | "reference" | "edit"

type Caps struct {
    Img2Img, Reference, Edit, MaskEdit bool
    MaxSide       int
    ReturnsAlpha  bool
}
type Model interface {
    Name() string
    Capabilities() Caps
    Run(ctx context.Context, req Request) (Result, error)
}
type Request struct {
    Mode           Mode
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
type Result struct { Image []byte; Format string; CostCents int; ProviderRef string; Seed int64; HasAlpha bool }
// provider/storage
type ObjectStore interface {
    Put(ctx, key string, r io.Reader, size int64, contentType string) error
    Get(ctx, key string) (io.ReadCloser, error)
    Delete(ctx, key string) error
    SignedURL(ctx, key string, ttl time.Duration) (string, error) // private objects
    PublicURL(key string) string                                  // keys under assets/ and shares/
}
```

**OSS driver.** Every key is stored under `OSS_PREFIX` (`yingji/test`, `yingji/production`) inside a bucket shared with other projects; the driver refuses to start without a prefix and rejects keys containing `..` or empty segments. All objects are written with a private ACL and the bucket policy is never changed, so `PublicURL` for `assets/` and `shares/` also returns a signed URL, valid for 24 h (the maximum the driver will sign). Uploads go through `OSS_ENDPOINT` (the VPC internal endpoint on the servers) and signing uses `OSS_PUBLIC_ENDPOINT`, so URLs handed to the client always resolve on the internet. `cmd/storagecheck` verifies a new environment end to end.

**NewAPI provider.** `provider/genmodel/newapi.go` posts a multipart request to `{NEWAPI_BASE_URL}/images/edits` with the user photo (and any references as `image[]`, optional `mask`), always sending the source image: it never falls back to text-to-image. The prompt is prefixed with an identity-preservation instruction. Output size is chosen by aspect ratio from the sizes the model accepts (`1024x1024`, `1024x1536`, `1536x1024`) and the pipeline then crops to the template's output size. Only `quality` and `background` may be passed through `gen_config.extra`. Responses may be Base64 or an HTTPS URL; URL downloads are restricted to HTTPS, public IPs and a short redirect chain. Timeouts, 429 and 5xx are returned as non-transient errors, so a paid request whose outcome is unknown is never re-sent automatically. Cost comes from `provider_prices["newapi/<model>"]` because the gateway does not report it; the price is read once when the process starts, so a change takes effect after the next restart.

Provider selection lives in `engine/genmodel`: `templates.gen_config.provider` → capability check for `mode` → `fallback_provider` → the default provider (`GEN_PROVIDER_DEFAULT`, `newapi` in test and prod; `app_configs.default_provider` only when the variable is empty). A provider is registered only when its API key is set. A `mock` provider returns the input with a stamp and is used in dev and CI; it is never registered when `APP_ENV=prod`. Each provider is wrapped in a circuit breaker (`pkg/breaker`, 5 failures / 30 s open) and a per-provider concurrency semaphore. Pipelines are plain Go functions (`pipeline/idphoto.Run`, `pipeline/template.Run`) that call the engines in order and share the helpers in `pipeline/steps` (`Prepare`, `Detect`, `Identity`, `Finish`).

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
| `default_provider` | `mock` | worker (gen model) when `GEN_PROVIDER_DEFAULT` is empty |
| `provider_prices` | `{}`; set e.g. `{"newapi/gpt-image-2.5": <cents>}` | worker cost capture when the provider returns none (NewAPI never does) |
| `identity_threshold` | `0.75` | worker identity check |
| `share_reward_enabled` | `true` | share incentive (D-22), on at launch |
| `share_reward_daily_cap` | `3` | share incentive |
| `share_preview_ttl_days` | `90` | share cleanup |
| `share_show_nickname` | `false` | share landing |
| `upload_max_bytes` | `10485760` | api |
| `photo_min_side_px` | `600` | photo check |
| `face_min_ratio` | `0.08` | photo check (face height / image height) |

## 9. Security

- JWT HS256 with 24 h expiry; `sub = user_id`, `aud = "mp"`. Admin tokens use `aud = "admin"`, a separate secret and 12 h expiry. Each secret is a single env value, so rotating it signs everyone out.
- All user-scoped queries filter by `user_id` from the token; no object id is trusted without an ownership check.
- Every object is private; the client only receives signed URLs: works 10 min, originals 30 min, avatars, catalogue assets and share images 24 h. The WeChat moderation service gets a 1 h URL.
- Uploads: sniff MIME (`http.DetectContentType`), accept `image/jpeg`, `image/png`, `image/webp`, reject others, including HEIC. Originals are re-encoded as JPEG before storage, which strips EXIF/GPS and normalises orientation.
- Webhooks always return 200 to avoid retry storms. The ad-reward callback is verified with `WECHAT_AD_CALLBACK_SECRET`; a bad signature is logged and ignored. The media-check callback is **not** signature-checked yet: it only acts on a `trace_id` this service stored in Redis, which limits but does not replace verification.
- Secrets never in the repo; `.env.example` lists names only. `config.Validate` refuses to start `APP_ENV=prod` with mock providers, a non-OSS store or default signing secrets.
- Admin API: separate JWT audience and login rate limit. Roles (`admin`, `ops`, `viewer`) are carried in the token but not yet enforced per route, and `/admin/v1` is reachable through the same public gateway path as the Mini Program API. Before launch, add an IP allow-list at the gateway or a separate hostname.

## 10. Observability

- Logs: `slog` JSON to stdout — one line per request (`request_id`, `user_id`) and lines for task success, pipeline failure (`task`, `code`), expiries and consistency refunds.
- Metrics: `GET /metrics` exposes only the default Go runtime and process collectors; there are no application metrics yet. Target: request latency/status, queue depth per queue, task duration per pipeline, provider latency/error rate, credits consumed/refunded per day, ad claims accepted/rejected per reason. Until then, `daily_stats` and the admin stats endpoints are the source for business numbers.
- Health: `GET /healthz` (process), `GET /readyz` (DB + Redis ping); the worker's health is its Asynq heartbeat (`cmd/healthcheck worker`).
- Alerts (target, see DEPLOYMENT.md §6): failed-task ratio > 10 % over 15 min, queue depth > 100, provider breaker open > 5 min, refund count spike.

## 11. What is verified today

Built and exercised locally with the `mock` providers and local storage (first against MySQL 8; since the switch, the credit, task, migration and Redis-isolation suites run against PostgreSQL 16 and Redis 7 in CI):

| Area | Status |
|---|---|
| Login, profile, privacy consent | Verified end to end |
| Catalogue (home, specs, templates, collections) | Verified end to end |
| Upload + photo check (resolution, blur, luma, face count) | Verified, including rejections |
| ID photo pipeline (free path) | Verified: 295×413 PNG, correct DPI metadata |
| Template pipeline (gen path) | Verified: credit consumed, AI label burned in, AIGC metadata written |
| Free recolor | Verified across white / blue / red |
| Credit ledger, refunds, idempotency, daily reset | Integration-tested against PostgreSQL (`internal/service/credit`, `internal/service/task`) |
| Versioned migrations, checksum guard, repeat runs | Tested (`internal/migration`) and run twice plus `seed` in CI |
| Redis key / queue prefix isolation | Tested (`internal/pkg/redisx`) |
| Crop geometry | Unit-tested (`internal/engine/local`) |
| Ad gating with ads disabled | Verified (409 on session create, no-ads daily grant) |
| Shares, works summary, admin stats and config | Verified end to end |
| NewAPI image edit (`gpt-image-2.5`) | Verified against the live gateway with a project sample photo |
| Alibaba Cloud OSS (put, get, sign, delete, prefix isolation) | Verified against the live bucket with `cmd/storagecheck` |
| Tencent vision, Volcengine gen model, COS storage | Compile only — never called against the live APIs. Production runs `FACE_PROVIDER=disabled` until Tencent credentials are configured, which rejects photo processing |
| WeChat login, subscribe messages, mediaCheck, ad callback | Compile only — needs a real AppID |

`backend/scripts/smoke.sh` runs the whole user journey against a running api + worker and is
deterministic (it sets the test user's balance through the admin API before the credit-gate step).

## 12. Testing strategy

- `service/credit` and `service/task` have table-driven tests covering the matrix in GENERATION_PIPELINE.md §9 against a real PostgreSQL. `internal/testutil` accepts only a database whose name ends in `_ci`, creates a schema per test and drops it afterwards; without `TEST_DATABASE_URL` these tests skip.
- Redis tests accept only a local instance (`TEST_REDIS_ADDR`), use DB 12 and a unique prefix, and never flush the database.
- CI (`.github/workflows/backend-check.yml`) runs `go test ./...` and `go vet ./...` with PostgreSQL 16 and Redis 7 service containers, then runs `migrate up` twice and `migrate seed` to prove migrations are repeatable.
- Provider adapters (`newapi`, `oss`, matting) have unit tests with fake HTTP servers or pure helpers.
- Not yet covered: pipelines with fixture images, handler tests with `httptest`, and an e2e target; `backend/scripts/smoke.sh` is the manual end-to-end check.
