# System Architecture

## 1. Goals and constraints

- One product surface: a WeChat Mini Program. No H5, no app in V1.0.
- Monetisation is attention-based: a rewarded video ad grants a generation credit. The single most important rule (PRD 2.3) is **no expensive generation starts before a credit has been reserved**.
- Catalogue content (specs, templates, banners, hot lists) is operated from an admin backend without releasing the Mini Program (PRD 34–35).
- Small team, fast iteration: a modular monolith in Go, one database, one queue. No microservices.
- Human faces are sensitive personal data under PIPL; storage and retention must be explicit (COMPLIANCE.md).

## 2. Component overview

```
┌──────────────────────────────┐        ┌──────────────────────────────┐
│  Client: uni-app             │        │  Admin console (web)         │
│  Vue 3 + TS → mp-weixin      │        │  React + Ant Design          │
└──────────────┬───────────────┘        └──────────────┬───────────────┘
               │ HTTPS (JSON, multipart)               │ HTTPS
               ▼                                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│  api  (Go / Gin)                                                     │
│  auth · catalogue · photos · tasks · credits · works · admin · hooks │
└───────┬──────────────┬──────────────────┬──────────────────┬─────────┘
        │              │                  │                  │
        ▼              ▼                  ▼                  ▼
PostgreSQL 16      Redis 7          Alibaba Cloud OSS   WeChat APIs
   (system of   (cache, rate      (originals, works,   (code2session,
    record)      limits, Asynq)    template assets)     subscribe msg,
        ▲              │                  ▲              mediaCheck,
        │              ▼                  │              ad callback)
┌───────┴──────────────────────────────────┴───────────────────────────┐
│  worker  (Go / Asynq)                                                │
│  engines: local · genmodel  → pipelines · notify · cleanup           │
└───────┬──────────────────────────────────────────────────────────────┘
        ▼
  NewAPI gateway: image model gpt-image-2.5 (worker; pluggable, with fallback)
                  multimodal photo check (api, once per upload) · mock in dev
```

| Component | Responsibility | Technology |
|---|---|---|
| Client | All user-facing UI; WeChat capabilities (login, ads, album, camera, subscribe messages, privacy, share) behind a platform layer | uni-app, Vue 3, TypeScript, Pinia, SCSS; target `mp-weixin` now, `app-plus` / `h5` later |
| `api` | Public JSON API, admin API, webhooks, auth, credit ledger, task creation | Go, Gin, GORM |
| `worker` | Executes tasks: one gen-model call per task plus local resize/crop, export and labelling; writes results, renders posters, sends notifications, runs scheduled cleanup | Go, Asynq |
| PostgreSQL | System of record: users, credits, ledger, tasks, works, catalogue, config | PostgreSQL 16 (existing Alibaba Cloud instance, one database per environment) |
| Redis | Asynq queues, config cache, rate limiting, ad-session TTL index | Redis 7 (shared instance; every key under the `yingji:<env>:` prefix) |
| Object storage | Originals, works, avatars, catalogue assets and share images — all private, served as signed URLs | Alibaba Cloud OSS (shared bucket, `yingji/<env>` prefix); local disk in dev |
| Admin console | Catalogue CRUD, config, task and user lookup, funnel dashboard | React + Ant Design (can start as a thin internal tool) |
| External | Upload photo check, image generation, WeChat platform | NewAPI gateway: a multimodal chat model for the photo check and `gpt-image-2.5` (`/images/edits`) for generation, behind capability-based routing and fallback (GENERATION_PIPELINE.md §2, §8); WeChat Open API incl. wxacode. No face detection or matting service (D-26) |

## 3. Trust boundaries

1. **Client → api.** The Mini Program is untrusted. Every request carries a JWT issued after `code2session`. All business rules (credits, caps, ownership) are enforced server-side. Ad completion reported by the client is treated as a claim, validated by the ad-session rules in GENERATION_PIPELINE.md §4.
2. **api → worker.** Internal. Tasks are enqueued only after the credit ledger transaction commits.
3. **worker → external providers.** Outbound only. Provider credentials live in the worker's environment; the api never holds them.
4. **WeChat → api (webhooks).** Ad server callback and message callbacks are verified by signature before use.
5. **Admin console → api.** Separate JWT audience, role-based, IP allow-list recommended.

## 4. Key flows

### 4.1 Login

```
Mini Program                    api                         WeChat
    │ wx.login() ─── code ──────▶│                             │
    │                            │── code2session(code) ──────▶│
    │                            │◀── openid, unionid ─────────│
    │                            │ upsert user, issue JWT      │
    │◀─────── {token, user} ─────│                             │
```

Tokens live 24 h. On 401 the client silently re-runs `wx.login` and retries once.

### 4.2 Generation (happy path, ads enabled, no credits)

```
1  Client: pick template/spec → upload photo → POST /v1/photos
2  api: multimodal photo check (face count, quality), store original in OSS, return photo + check result
3  Client: confirm → POST /v1/tasks
4  api: credit check fails → 402 NO_CREDITS
5  Client: POST /v1/ads/sessions → {session_id}; play rewarded video
6  Client: onClose(isEnded=true) → POST /v1/ads/sessions/{id}/claim
7  api: validate session, ledger +1 bonus, return balance
8  Client: POST /v1/tasks (same idempotency key)
9  api: in one DB transaction: lock credit row, ledger −1, insert task(waiting); commit; enqueue Asynq job
10 worker: task → processing; run pipeline; upload result; insert work; task → success; ledger unchanged
11 worker: subscribe message (if granted)
12 Client: polling GET /v1/tasks/{id} sees success → result page
```

Failure at step 10 flips the task to `failed`, inserts a `refund` ledger row and notifies. Every task, ID photos included, goes through steps 3–12; there is no free path (D-26). Details and every edge case: GENERATION_PIPELINE.md.

### 4.3 Catalogue delivery

Each tab loads `GET /v1/home/{module}` once, which returns banner, hot items, feature strip and hot templates in one payload assembled from admin-managed tables and cached in Redis for 60 s. Detail and "more" pages hit the individual list endpoints.

## 5. Data ownership

| Data | Owner | Store | Lifetime |
|---|---|---|---|
| User identity, credits, ledger | api | PostgreSQL | Account lifetime |
| Uploaded originals | user | OSS `originals/` (private, 30 min signed URLs) | 30 days or user deletion (D-16) |
| Generated works | user | OSS `works/` (private, 10 min signed URLs) + PostgreSQL row | Until user deletion |
| Legacy ID-photo alpha matte | — | OSS `works/{user}/{id}_alpha.png`, only on works made before D-26 | Deleted with the last work that references it; new tasks keep all intermediates in memory |
| Catalogue assets | ops | OSS `assets/` (private, 24 h signed URLs) | Managed in admin |
| Share previews and posters | user (explicit share action) | OSS `shares/` (private, 24 h signed URLs, AI label burned in) | Until revoke, work deletion, or 90 days after last open |
| Task records, cost | api/worker | PostgreSQL | Retained for analytics |

All object keys above are relative to the environment prefix (`yingji/test/`, `yingji/production/`) in a bucket shared with other projects. The bucket stays private: this project never changes its ACL or policy, so even "public" prefixes are delivered as signed URLs.

## 6. Scalability and cost posture

- `api` is stateless; scale horizontally behind a load balancer.
- `worker` concurrency is bounded per provider (`generation` queue, concurrency 4 by default, 1 on the current shared hosts) so a spike converts into queue depth, not provider throttling and wasted retries.
- Database connections are capped per process (2 on the shared PostgreSQL instance), so scaling out means raising that budget with the database owner first.
- Provider cost is written on every task; a daily job aggregates cost per template for the admin dashboard.
- All external calls have explicit timeouts and a circuit breaker per provider. When the breaker is open, `POST /v1/tasks` returns 503 `GENERATION_UNAVAILABLE` **before** any credit is consumed.

## 7. Non-goals for V1.0

Payment, membership, community, multi-face inputs, custom spec sizes, image editor, multi-region deployment. The App and H5 builds are not in V1.0 but the uni-app codebase and per-provider identities keep them reachable without rework.
