# Data Model — PostgreSQL 16

Conventions: `id` is a 26-char ULID string stored as `VARCHAR(26)` (sortable, safe for URLs); timestamps are `TIMESTAMPTZ(3)` written in UTC; soft delete via `deleted_at` only where the user can delete; `JSON` columns hold structures that the admin edits as a whole. Enumerations are `VARCHAR` columns whose allowed values are enforced by the application (no PostgreSQL `ENUM` types), and relations are not declared as foreign keys; ownership and existence are checked in the service layer. Counters and balances are `BIGINT`; non-negativity is enforced in `credit.Service`, not by the column type.

The authoritative DDL is `backend/internal/migration/postgres/*.sql` (see `backend/migrations/README.md`). This document explains it; when the two disagree, the SQL wins and this file is fixed in the same change.

## 1. Entity map

```
users ─1:N─ user_identities
  ├─1:1─ credit_accounts ─1:N─ credit_ledger
  │                                   ▲
  ├─1:N─ ad_sessions ─────────────────┘ (ad_reward)
  ├─1:N─ photos ─1:N─ tasks ─1:1─ works ─1:N─ shares ─1:N─ share_opens
  │                     └──────────────┘ (consume / refund)
  ├─1:N─ favorites ─N:1─ templates ─N:1─ categories
  └─1:N─ events                      └─N:M─ collections
specs ─N:1─ categories
app_configs, admin_users, audit_logs, daily_stats
```

## 2. Tables

### users

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | ULID |
| nickname | VARCHAR(64) NULL | user-entered (D-11) |
| avatar_key | VARCHAR(255) NULL | object storage key |
| status | SMALLINT | 1 active, 2 banned |
| privacy_agreed_at | TIMESTAMPTZ(3) NULL | privacy authorisation timestamp |
| privacy_version | VARCHAR(32) NULL | policy version the user agreed to |
| acquired_share_id | VARCHAR(26) NULL | attribution (SHARING.md §3) |
| last_login_at | TIMESTAMPTZ(3) | |
| created_at / updated_at | TIMESTAMPTZ(3) | |

### user_identities (D-23)

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| user_id | VARCHAR(26) INDEX | |
| provider | VARCHAR(16) | `wechat_mp`, `wechat_app`, `phone` (`h5_dev` in dev only) |
| provider_uid | VARCHAR(64) | openid for WeChat providers, E.164 for phone |
| union_id | VARCHAR(64) NULL, INDEX | links wechat_mp and wechat_app to one user |
| created_at | TIMESTAMPTZ(3) | |

Unique index `(provider, provider_uid)`. Login resolves `(provider, uid)` → user, else `union_id` → user, else creates a user.

### credit_accounts

| Column | Type | Notes |
|---|---|---|
| user_id | VARCHAR(26) PK | one row per user |
| daily_free_remaining | BIGINT | reset lazily; never below 0 |
| daily_reset_date | DATE | Asia/Shanghai date of last reset |
| bonus_credits | BIGINT | from ads / shares / admin; never below 0 |
| ad_rewards_today | BIGINT | reset with daily_reset_date |
| updated_at | TIMESTAMPTZ(3) | |

Every balance change locks the row with `SELECT … FOR UPDATE` (BACKEND_ARCHITECTURE.md §5).

### credit_ledger

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| user_id | VARCHAR(26) INDEX | |
| kind | VARCHAR(16) | `daily_grant`, `ad_reward`, `consume`, `refund`, `share_reward`, `admin_adjust` |
| bucket | VARCHAR(8) | `daily` or `bonus`: which balance changed |
| delta | BIGINT | signed |
| balance_daily_after | BIGINT | snapshot |
| balance_bonus_after | BIGINT | snapshot |
| ref_type | VARCHAR(32) | 'task', 'ad_session', 'day', 'admin' |
| ref_id | VARCHAR(64) | task id, ad session id, 'YYYY-MM-DD', acquired user id, admin op id |
| note | VARCHAR(255) NULL | |
| created_at | TIMESTAMPTZ(3) | |

Unique index `(kind, ref_type, ref_id)` — this is what makes consume/refund/reward idempotent. The service returns an existing row for the same ref before writing; the index rejects any concurrent duplicate.

### ad_sessions

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| user_id | VARCHAR(26) INDEX | |
| ad_unit_id | VARCHAR(64) | |
| status | VARCHAR(16) | `pending`, `claimed`, `rejected`, `expired` |
| reject_reason | VARCHAR(64) NULL | too_fast, cap_reached, expired, not_ended, duplicate |
| client_is_ended | BOOLEAN NULL | what the client reported |
| server_verified_at | TIMESTAMPTZ(3) NULL | set by WeChat server callback when enabled |
| trans_id | VARCHAR(128) NULL UNIQUE | from server callback |
| created_at / claimed_at | TIMESTAMPTZ(3) | |

### photos (uploaded originals)

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| user_id | VARCHAR(26) INDEX | |
| object_key | VARCHAR(255) | `originals/{user_id}/{id}.jpg` |
| width / height | BIGINT | |
| bytes | BIGINT | |
| sha256 | VARCHAR(64) INDEX | dedupe within user |
| check_status | VARCHAR(16) | `pending`, `passed`, `rejected` |
| check_result | JSON | `{faces:1, face_bbox:[…], quality:{blur:0.1,dark:false}, reasons:[]}` |
| moderation_status | VARCHAR(16) | `pending` (default), `pass`, `risky` — mediaCheckAsync |
| expires_at | TIMESTAMPTZ(3) | created_at + retention days |
| created_at | TIMESTAMPTZ(3) | |
| deleted_at | TIMESTAMPTZ(3) NULL, INDEX | |

### categories

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| module | VARCHAR(16) INDEX | `idphoto`, `pro`, `portrait`, `avatar` |
| kind | VARCHAR(16) | `spec` or `template`: spec categories vs template categories |
| name | VARCHAR(32) | |
| icon | VARCHAR(64) NULL | icon identifier |
| cover_key | VARCHAR(255) NULL | |
| desc | VARCHAR(64) NULL | one-line description |
| sort | BIGINT | |
| status | SMALLINT | 1 online, 0 offline |

### specs (ID photo sizes)

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| category_id | VARCHAR(26) INDEX | |
| name | VARCHAR(32) | 一寸 |
| width_mm / height_mm | DECIMAL(5,1) | |
| width_px / height_px | BIGINT | at `dpi` |
| dpi | BIGINT | default 300 |
| bg_default | VARCHAR(7) | `#FFFFFF` |
| bg_allowed | JSON | `["#FFFFFF","#438EDB","#FF0000","#808080"]` |
| crop_rule | JSON NULL | overrides: `{head_ratio:0.62, top_margin:0.10}` |
| source_note | VARCHAR(255) NULL | where the size was verified (UI-16) |
| note | VARCHAR(64) NULL | short hint shown on the card |
| is_hot | BOOLEAN | shows in "常用规格" (max 4) |
| sort | BIGINT | |
| status | SMALLINT | |
| created_at / updated_at | TIMESTAMPTZ(3) | |

### templates

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| module | VARCHAR(16) | `pro`, `portrait`, `avatar` |
| category_id | VARCHAR(26) INDEX | |
| name | VARCHAR(32) | |
| subtitle | VARCHAR(64) NULL | factual only (UI-02) |
| cover_key | VARCHAR(255) | 3:4 (pro/portrait) or 1:1 (avatar) |
| sample_keys | JSON | example outputs |
| credit_cost | BIGINT | default 1; price of one task with this template (D-21) |
| gen_config | JSON | recipe, schema in GENERATION_PIPELINE.md §7.3: `{engine, provider, model, mode, prompt, negative_prompt, strength, output, identity_check, post:[…], style, fallback_provider}` |
| tags | JSON | `["NEW","热门"]` |
| is_hot | BOOLEAN | |
| sort | BIGINT | |
| status | SMALLINT | |
| created_at / updated_at | TIMESTAMPTZ(3) | |

Index `(module, status, is_hot, sort)`.

### collections / collection_templates (专题)

`collections`: id, module, name, cover_key, description, sort, status. `collection_templates`: collection_id, template_id, sort; PK (collection_id, template_id).

### banners

id, module, title, subtitle, image_key, link (JSON: `{type:'idphoto_flow'|'template'|'collection'|'url', id}`), sort, status, start_at, end_at.

### tasks

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| user_id | VARCHAR(26) | |
| idempotency_key | VARCHAR(36) | UNIQUE with user_id |
| module | VARCHAR(16) | `idphoto`, `pro`, `portrait`, `avatar` |
| kind | VARCHAR(16) | `idphoto`, `template`, `idphoto_recolor`; recolor is free and synchronous |
| spec_id | VARCHAR(26) NULL | |
| template_id | VARCHAR(26) NULL | |
| photo_id | VARCHAR(26) | |
| parent_task_id | VARCHAR(26) NULL | regenerate / change clothing lineage |
| uses_genmodel | BOOLEAN | decided at creation; credits and breaker apply only when true (D-21) |
| credits_consumed | BIGINT | 0 for free tasks |
| params | JSON | `{bg:'#438EDB', clothing:'white_shirt', beauty:'natural', seed:…}` |
| status | VARCHAR(16) | `waiting`, `processing`, `success`, `failed` |
| stage | VARCHAR(16) NULL | `queued`, `processing`, `finishing`; shown to user |
| provider | VARCHAR(32) NULL | gen-model provider that produced the output, e.g. `newapi` |
| provider_ref | VARCHAR(128) NULL | |
| work_id | VARCHAR(26) NULL | |
| error_code | VARCHAR(32) NULL | TIMEOUT, PROVIDER_ERROR, CONTENT_REJECTED, NO_FACE, … |
| error_message | VARCHAR(255) NULL | |
| cost_cents | BIGINT | provider cost |
| consume_ledger_id | VARCHAR(26) NULL | null for free kinds |
| refund_ledger_id | VARCHAR(26) NULL | |
| notify_requested | BOOLEAN | subscribe message accepted |
| created_at / started_at / finished_at | TIMESTAMPTZ(3) | |

Index `(user_id, created_at)`, `(status, started_at)` for the expiry scheduler.

### works

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | |
| user_id | VARCHAR(26) INDEX | |
| task_id | VARCHAR(26) UNIQUE, NULL | the task that produced this work; NULL for a free recolor, which creates no task (PostgreSQL treats NULLs as distinct in a unique index) |
| module | VARCHAR(16) | |
| spec_id / template_id | VARCHAR(26) NULL | |
| object_key | VARCHAR(255) | `works/{user_id}/{id}.jpg` (PNG for idphoto) |
| thumb_key | VARCHAR(255) | |
| alpha_key | VARCHAR(255) NULL | idphoto matte layer for free recolor |
| width / height | BIGINT | |
| meta | JSON | `{bg:'#438EDB', clothing:'white_shirt', width_mm, height_mm, dpi, label:{visible:true, metadata:true}}` |
| ai_label | BOOLEAN | AI-generated label applied (COMPLIANCE.md §3.2) |
| moderation_status | VARCHAR(16) | `pending` (default), `pass`, `risky` |
| created_at | TIMESTAMPTZ(3) | |
| deleted_at | TIMESTAMPTZ(3) NULL, INDEX | |

### favorites

user_id, template_id, created_at; PK (user_id, template_id).

### shares (SHARING.md)

| Column | Type | Notes |
|---|---|---|
| id | VARCHAR(26) PK | used as `s=` param and mini program code scene |
| user_id | VARCHAR(26) INDEX | sharer |
| type | VARCHAR(16) | `template`, `work`, `poster`, `tab` |
| module | VARCHAR(16) | |
| template_id / spec_id | VARCHAR(26) NULL | |
| work_id | VARCHAR(26) NULL, INDEX | |
| preview_key | VARCHAR(255) NULL | 5:4 copy with AI label (work/poster) under `shares/` |
| poster_key | VARCHAR(255) NULL | rendered poster |
| path | VARCHAR(255) | landing path including `s=` |
| title | VARCHAR(128) NULL | share card title |
| status | VARCHAR(16) | `active` (default), `revoked` |
| opens | BIGINT | denormalised counter |
| last_opened_at | TIMESTAMPTZ(3) NULL | |
| created_at | TIMESTAMPTZ(3) | |

### share_opens

id BIGSERIAL, share_id INDEX, opener_user_id NULL (set when the opener logs in), device_id INDEX, platform, created_at. Attribution: on first login, if the device opened a share within 7 days, `users.acquired_share_id` is set and a `share_attributed` event is written.

### events (analytics, D-17)

id BIGSERIAL, user_id, name VARCHAR(48), props JSON, client_ts TIMESTAMPTZ(3), created_at. Index (name, created_at). Not partitioned in V1; switch to monthly range partitioning on `created_at` when the table outgrows routine vacuuming.

### app_configs

`key VARCHAR(64) PK, value JSON, description VARCHAR(255), updated_by, updated_at`.

### admin_users / audit_logs

`admin_users`: id, username UNIQUE, password_hash (bcrypt), role VARCHAR(16) (`admin` default, `ops`, `viewer`), status, last_login_at. `audit_logs`: id BIGSERIAL, admin_id, action, target_type, target_id, before JSON, after JSON, ip, created_at.

### daily_stats (rollup)

date, module, template_id (`''` when not applicable), spec_id (`''` when not applicable), tasks_created, tasks_success, tasks_failed, works_saved, credits_consumed, credits_refunded, ad_claims, cost_cents; PK (date, module, template_id, spec_id). Empty strings instead of NULL keep the rollup rows addressable by the primary key.

### schema_migrations

Owned by `cmd/migrate`: version (file name) PK, checksum (SHA-256 of the file), applied_at. See §5.

## 3. Invariants

1. `credit_ledger` unique `(kind, ref_type, ref_id)` guarantees at most one consume and one refund per task and one reward per ad session.
2. A task with `consume_ledger_id` set and `status = failed` must have `refund_ledger_id` set, and a task whose work is `risky` must be `failed/CONTENT_REJECTED` (both checked by the `consistency:refund` job every 30 minutes). No task stays `waiting` past `task_queue_timeout_seconds` or `processing` past `task_timeout_seconds` (expiry scheduler).
3. `works.task_id` is unique where present: at most one output per task. "Regenerate" creates a new task; a free recolor creates a work with `task_id = NULL` and `meta.recolored_from` pointing at the source work.
4. `photos.expires_at` drives cleanup; works never reference the original's object key, they hold their own copy.
5. `specs.is_hot = true` rows are limited to 4 per module by the admin API, matching PRD 5.3.
6. `tasks.uses_genmodel = false` implies `consume_ledger_id IS NULL` and `credits_consumed = 0`; `uses_genmodel = true` implies `credits_consumed = templates.credit_cost` (or 1 for spec tasks) at creation.
7. A `share` with `type IN ('work','poster')` must reference a work owned by `user_id`; deleting the work revokes the share and deletes `preview_key` / `poster_key` objects.
8. `credit_ledger` rows with `kind = share_reward` are unique per `(ref_type='user', ref_id=acquired_user_id)`.

## 4. Seed data

`go run ./cmd/migrate seed` (`internal/seed`) loads the initial spec categories, specs, templates, collections, banners, `app_configs` defaults and the first admin user; images come from `backend/seed/assets`. Every spec row carries `source_note` naming the authority the size was verified against (`unverified` until checked). Sizes taken from the mockups are **not** to be marked verified until checked (UI-16). Seed overwrites rows with the same ID, so it runs once per environment (the deploy script guards it with `.initialized-postgres`); day-to-day template changes go through the admin API.

## 5. Migrations

- Versioned SQL lives in `backend/internal/migration/postgres/` and is embedded in the binary; `cmd/migrate up` applies new files in name order.
- Each file runs in its own transaction and is recorded in `schema_migrations` with its SHA-256. An applied file whose checksum changed stops the release; never edit an applied file, add a new one.
- A PostgreSQL advisory lock (`pg_try_advisory_lock`) prevents two releases from migrating at once.
- Migrations are forward-only in production. Drop or rename a column in two releases (add → backfill → switch → drop) so the previous image keeps working during a rollback.
- `backend/internal/migration/sql/` holds the retired MySQL history for reference only; it is not executed.
