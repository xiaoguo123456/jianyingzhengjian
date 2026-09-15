# Data Model — MySQL 8

Conventions: `id` is a 26-char ULID string (sortable, safe for URLs); timestamps are `DATETIME(3)` in UTC; soft delete via `deleted_at` only where the user can delete; JSON columns hold structures that the admin edits as a whole. Charset `utf8mb4`, collation `utf8mb4_0900_ai_ci`.

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
| id | CHAR(26) PK | ULID |
| nickname | VARCHAR(64) NULL | user-entered (D-11) |
| avatar_key | VARCHAR(255) NULL | COS key |
| status | TINYINT | 1 active, 2 banned |
| privacy_agreed_at | DATETIME(3) NULL | privacy authorisation timestamp |
| acquired_share_id | CHAR(26) NULL | attribution (SHARING.md §3) |
| last_login_at | DATETIME(3) | |
| created_at / updated_at | DATETIME(3) | |

### user_identities (D-23)

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| user_id | CHAR(26) INDEX | |
| provider | ENUM('wechat_mp','wechat_app','phone') | |
| provider_uid | VARCHAR(64) | openid for WeChat providers, E.164 for phone |
| unionid | VARCHAR(64) NULL, INDEX | links wechat_mp and wechat_app to one user |
| created_at | DATETIME(3) | |

Unique key `(provider, provider_uid)`. Login resolves `(provider, uid)` → user, else `unionid` → user, else creates a user.

### credit_accounts

| Column | Type | Notes |
|---|---|---|
| user_id | CHAR(26) PK, FK users | |
| daily_free_remaining | INT UNSIGNED | reset lazily |
| daily_reset_date | DATE | Asia/Shanghai date of last reset |
| bonus_credits | INT UNSIGNED | from ads / admin |
| ad_rewards_today | INT UNSIGNED | reset with daily_reset_date |
| updated_at | DATETIME(3) | |

### credit_ledger

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| user_id | CHAR(26) INDEX | |
| kind | ENUM('daily_grant','ad_reward','consume','refund','share_reward','admin_adjust') | |
| bucket | ENUM('daily','bonus') | which balance changed |
| delta | INT | signed |
| balance_daily_after | INT UNSIGNED | snapshot |
| balance_bonus_after | INT UNSIGNED | snapshot |
| ref_type | VARCHAR(32) | 'task', 'ad_session', 'day', 'admin' |
| ref_id | VARCHAR(64) | task id, ad session id, 'YYYY-MM-DD', acquired user id, admin op id |
| note | VARCHAR(255) NULL | |
| created_at | DATETIME(3) | |

Unique key `(kind, ref_type, ref_id)` — this is what makes consume/refund/reward idempotent.

### ad_sessions

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| user_id | CHAR(26) INDEX | |
| ad_unit_id | VARCHAR(64) | |
| status | ENUM('pending','claimed','rejected','expired') | |
| reject_reason | VARCHAR(64) NULL | too_fast, cap_reached, expired, not_ended, duplicate |
| client_is_ended | TINYINT(1) NULL | what the client reported |
| server_verified_at | DATETIME(3) NULL | set by WeChat server callback when enabled |
| trans_id | VARCHAR(128) NULL UNIQUE | from server callback |
| created_at / claimed_at | DATETIME(3) | |

### photos (uploaded originals)

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| user_id | CHAR(26) INDEX | |
| object_key | VARCHAR(255) | `originals/{user_id}/{id}.jpg` |
| width / height | INT | |
| bytes | INT | |
| sha256 | CHAR(64) INDEX | dedupe within user |
| check_status | ENUM('pending','passed','rejected') | |
| check_result | JSON | `{faces:1, face_bbox:[…], quality:{blur:0.1,dark:false}, reasons:[]}` |
| moderation_status | ENUM('pending','pass','risky') | mediaCheckAsync |
| expires_at | DATETIME(3) | created_at + retention days |
| created_at | DATETIME(3) | |
| deleted_at | DATETIME(3) NULL | |

### categories

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| module | ENUM('idphoto','pro','portrait','avatar') | |
| kind | ENUM('spec','template') | spec categories vs template categories |
| name | VARCHAR(32) | |
| icon_key | VARCHAR(255) NULL | |
| sort | INT | |
| status | TINYINT | 1 online, 0 offline |

### specs (ID photo sizes)

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| category_id | CHAR(26) FK | |
| name | VARCHAR(32) | 一寸 |
| width_mm / height_mm | DECIMAL(5,1) | |
| width_px / height_px | INT | at `dpi` |
| dpi | INT | default 300 |
| bg_default | CHAR(7) | `#FFFFFF` |
| bg_allowed | JSON | `["#FFFFFF","#438EDB","#FF0000","#808080"]` |
| crop_rule | JSON NULL | overrides: `{head_ratio:0.62, top_margin:0.10}` |
| source_note | VARCHAR(255) NULL | where the size was verified (UI-16) |
| is_hot | TINYINT(1) | shows in "常用规格" (max 4) |
| sort | INT | |
| status | TINYINT | |
| created_at / updated_at | DATETIME(3) | |

### templates

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| module | ENUM('pro','portrait','avatar') | |
| category_id | CHAR(26) FK | |
| name | VARCHAR(32) | |
| subtitle | VARCHAR(64) NULL | factual only (UI-02) |
| cover_key | VARCHAR(255) | 3:4 (pro/portrait) or 1:1 (avatar) |
| sample_keys | JSON | example outputs |
| credit_cost | INT UNSIGNED | default 1; price of one task with this template (D-21) |
| gen_config | JSON | recipe, schema in GENERATION_PIPELINE.md §7.3: `{engine, provider, model, mode, prompt, negative_prompt, strength, output, identity_check, post:[…], style, fallback_provider}` |
| tags | JSON | `["NEW","热门"]` |
| is_hot | TINYINT(1) | |
| sort | INT | |
| status | TINYINT | |
| created_at / updated_at | DATETIME(3) | |

Index `(module, status, is_hot, sort)`.

### collections / collection_templates (专题)

`collections`: id, module, name, cover_key, description, sort, status. `collection_templates`: collection_id, template_id, sort; PK (collection_id, template_id).

### banners

id, module, title, subtitle, image_key, link (JSON: `{type:'idphoto_flow'|'template'|'collection'|'url', id}`), sort, status, start_at, end_at.

### tasks

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| user_id | CHAR(26) INDEX | |
| idempotency_key | CHAR(36) | UNIQUE with user_id |
| module | ENUM('idphoto','pro','portrait','avatar') | |
| kind | ENUM('idphoto','template','idphoto_recolor') | recolor is free and synchronous |
| spec_id | CHAR(26) NULL | |
| template_id | CHAR(26) NULL | |
| photo_id | CHAR(26) | |
| parent_task_id | CHAR(26) NULL | regenerate / change clothing lineage |
| uses_genmodel | TINYINT(1) | decided at creation; credits and breaker apply only when true (D-21) |
| credits_consumed | INT UNSIGNED | 0 for free tasks |
| params | JSON | `{bg:'#438EDB', clothing:'white_shirt', beauty:'natural', seed:…}` |
| status | ENUM('waiting','processing','success','failed') | |
| stage | ENUM('queued','processing','finishing') NULL | shown to user |
| provider | VARCHAR(32) NULL | |
| provider_ref | VARCHAR(128) NULL | |
| work_id | CHAR(26) NULL | |
| error_code | VARCHAR(32) NULL | TIMEOUT, PROVIDER_ERROR, CONTENT_REJECTED, NO_FACE, … |
| error_message | VARCHAR(255) NULL | |
| cost_cents | INT | provider cost |
| consume_ledger_id | CHAR(26) NULL | null for free kinds |
| refund_ledger_id | CHAR(26) NULL | |
| notify_requested | TINYINT(1) | subscribe message accepted |
| created_at / started_at / finished_at | DATETIME(3) | |

Index `(user_id, created_at)`, `(status, started_at)` for the expiry scheduler.

### works

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | |
| user_id | CHAR(26) INDEX | |
| task_id | CHAR(26) UNIQUE, NULL | the task that produced this work; NULL for a free recolor, which creates no task (MySQL allows repeated NULLs in a unique index) |
| module | ENUM(...) | |
| spec_id / template_id | CHAR(26) NULL | |
| object_key | VARCHAR(255) | `works/{user_id}/{id}.jpg` (PNG for idphoto) |
| thumb_key | VARCHAR(255) | |
| alpha_key | VARCHAR(255) NULL | idphoto matte layer for free recolor |
| width / height | INT | |
| meta | JSON | `{bg:'#438EDB', clothing:'white_shirt', width_mm, height_mm, dpi, label:{visible:true, metadata:true}}` |
| moderation_status | ENUM('pending','pass','risky') | |
| created_at | DATETIME(3) | |
| deleted_at | DATETIME(3) NULL | |

### favorites

user_id, template_id, created_at; PK (user_id, template_id).

### shares (SHARING.md)

| Column | Type | Notes |
|---|---|---|
| id | CHAR(26) PK | used as `s=` param and mini program code scene |
| user_id | CHAR(26) INDEX | sharer |
| type | ENUM('template','work','poster','tab') | |
| module | ENUM(...) | |
| template_id / spec_id / work_id | CHAR(26) NULL | |
| preview_key | VARCHAR(255) NULL | public 5:4 copy with AI label (work/poster) |
| poster_key | VARCHAR(255) NULL | rendered poster |
| path | VARCHAR(255) | landing path including `s=` |
| status | ENUM('active','revoked') | |
| opens | INT UNSIGNED | denormalised counter |
| last_opened_at | DATETIME(3) NULL | |
| created_at | DATETIME(3) | |

### share_opens

id BIGINT AUTO_INCREMENT, share_id INDEX, opener_user_id NULL (set when the opener logs in), platform, created_at. Attribution: on first login, if the device opened a share within 7 days, `users.acquired_share_id` is set and a `share_attributed` event is written.

### events (analytics, D-17)

id BIGINT AUTO_INCREMENT, user_id, name VARCHAR(48), props JSON, client_ts DATETIME(3), created_at. Partition by month; index (name, created_at).

### app_configs

`key VARCHAR(64) PK, value JSON, description VARCHAR(255), updated_by, updated_at`.

### admin_users / audit_logs

`admin_users`: id, username UNIQUE, password_hash (bcrypt), role ENUM('admin','ops','viewer'), status, last_login_at. `audit_logs`: id, admin_id, action, target_type, target_id, before JSON, after JSON, ip, created_at.

### daily_stats (rollup)

date, module, template_id NULL, spec_id NULL, tasks_created, tasks_success, tasks_failed, works_saved, credits_consumed, credits_refunded, ad_claims, cost_cents; PK (date, module, template_id, spec_id).

## 3. Invariants

1. `credit_ledger` unique `(kind, ref_type, ref_id)` guarantees at most one consume and one refund per task and one reward per ad session.
2. A task with `consume_ledger_id` set and `status = failed` must have `refund_ledger_id` set (checked by the expiry scheduler and a nightly consistency job).
3. `works.task_id` is unique where present: at most one output per task. "Regenerate" creates a new task; a free recolor creates a work with `task_id = NULL` and `meta.recolored_from` pointing at the source work.
4. `photos.expires_at` drives cleanup; works never reference the original's object key, they hold their own copy.
5. `specs.is_hot = 1` rows are limited to 4 per module by the admin API, matching PRD 5.3.
6. `tasks.uses_genmodel = 0` implies `consume_ledger_id IS NULL` and `credits_consumed = 0`; `uses_genmodel = 1` implies `credits_consumed = templates.credit_cost` (or 1 for spec tasks) at creation.
7. A `share` with `type IN ('work','poster')` must reference a work owned by `user_id`; deleting the work revokes the share and deletes `preview_key` / `poster_key` objects.
8. `credit_ledger` rows with `kind = share_reward` are unique per `(ref_type='user', ref_id=acquired_user_id)`.

## 4. Seed data

`migrations/seed/` contains the initial spec categories and specs. Every spec row carries `source_note` naming the authority the size was verified against. Sizes taken from the mockups are **not** to be seeded until verified (UI-16).
