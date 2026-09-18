# HTTP API

Base URL: `https://api.<domain>/`. All bodies are JSON unless stated. Every response uses the envelope:

```json
{ "code": "OK", "message": "", "data": {}, "request_id": "01J…" }
```

Authentication: `Authorization: Bearer <jwt>`. Client headers: `X-Request-Id` (uuid), `X-Client-Version`. Times are ISO-8601 UTC. Pagination: `?page=1&page_size=20` → `data.items`, `data.total`, `data.has_more`.

## 1. Error codes

| HTTP | code | Meaning |
|---|---|---|
| 400 | `BAD_REQUEST` | validation failure; `data.fields` lists problems |
| 401 | `UNAUTHORIZED` | missing/expired token; client re-logins once |
| 402 | `NO_CREDITS` | task creation refused; `data.credits` returns balances and `ads_enabled` |
| 403 | `FORBIDDEN` | not owner / banned |
| 404 | `NOT_FOUND` | |
| 409 | `CONFLICT` | idempotency key reused with different payload; duplicate favorite |
| 413 | `PAYLOAD_TOO_LARGE` | upload over limit |
| 422 | `PHOTO_REJECTED` | photo check failed; `data.reasons` |
| 422 | `AD_SESSION_INVALID` | `data.reason`: too_fast, cap_reached, expired, not_ended, duplicate |
| 429 | `RATE_LIMITED` | |
| 503 | `GENERATION_UNAVAILABLE` | provider breaker open; no credit consumed |
| 500 | `INTERNAL` | |

## 2. Auth and profile

### POST /v1/auth/login
Request `{ "provider": "wechat_mp", "code": "wx login code", "share_id": "01J… (optional, from a share landing)" }`
`provider ∈ wechat_mp` in V1.0; `wechat_app`, `phone` reserved for the App (D-23).
Response `{ "token": "…", "expires_in": 86400, "user": User, "is_new": true }`

### GET /v1/me
Response `{ "user": User, "credits": Credits, "config": ClientConfig }`

`ClientConfig = { ads_enabled, ad_unit_ids: {reward: "…"}, subscribe_template_ids: {task_finished: "…"}, privacy_policy_url, retention_days }`

### PUT /v1/me
Request `{ "nickname": "阳光小橙" }`

### POST /v1/me/avatar
multipart `file` → `{ "avatar_url": "…" }`

### POST /v1/me/privacy-agree
Records `privacy_agreed_at`. Request `{ "version": "2026-09-01" }`.

### DELETE /v1/me
Account deletion request: anonymises user, deletes photos and works (COMPLIANCE.md §2.4). Responds 202.

## 3. Credits and ads

### GET /v1/credits
Response
```json
{ "daily_free_remaining": 1, "bonus_credits": 2, "total": 3,
  "ad_rewards_today": 1, "ad_reward_daily_cap": 10, "ads_enabled": true }
```

### POST /v1/ads/sessions
Creates a pending session before the client loads the ad.
Response `{ "session_id": "01J…", "ad_unit_id": "adunit-…", "expires_at": "…" }`
Errors: 422 `AD_SESSION_INVALID` (`cap_reached`), 409 if `ads_enabled=false`.

### POST /v1/ads/sessions/{id}/claim
Request `{ "is_ended": true }`
Response `{ "granted": 1, "credits": Credits }`
Errors: 422 `AD_SESSION_INVALID` with `reason`.

## 4. Catalogue

### GET /v1/home/{module}
`module ∈ idphoto | pro | portrait | avatar`. Cached 60 s.
```json
{
  "banner": { "title": "上传自拍，快速生成标准证件照", "subtitle": "智能识别 · 自动裁切 · 多种规格", "image_url": "…", "link": {"type":"upload"} },
  "hot_specs": [Spec],                 // idphoto only, max 4
  "more_specs": [Spec],                // idphoto only, max 4 ("常见用途" rows)
  "hot_categories": [Category],        // pro / portrait / avatar, max 4; Category carries cover_url for the photo tiles
  "collections": [Collection],         // portrait only
  "hot_templates": [TemplateCard],     // pro / portrait / avatar, max 6 (horizontal rail)
  "rails": [                           // pro / portrait / avatar, 0–3 secondary rails
    { "title": "求职面试", "category_id": "…", "collection_id": null, "templates": [TemplateCard] }
  ]
}
```

The banner is the tab's single call to action (whole card plus an explicit button). There is no separate feature strip and no sticky button on tabs; the explanatory copy lives in `banner.subtitle`.

### GET /v1/spec-categories
### GET /v1/specs?category_id=&hot=1&page=
### GET /v1/specs/{id}
`Spec = { id, name, width_mm, height_mm, width_px, height_px, dpi, bg_default, bg_allowed, category: {id,name} }`

### GET /v1/template-categories?module=
### GET /v1/templates?module=&category_id=&collection_id=&hot=1&page=
`TemplateCard = { id, module, name, cover_url, tags, is_favorited }`
### GET /v1/templates/{id}
`Template = TemplateCard + { subtitle, sample_urls, output: {width, height, aspect}, style: "photo"|"illustration", credit_cost: 1, clothing_options?: [...] }`
### GET /v1/collections?module=
### GET /v1/collections/{id}
Returns the collection plus paged templates.

## 5. Photos

### POST /v1/photos
multipart `file` (jpeg/png/webp, ≤ 10 MB), field `module`.
Runs face check synchronously (target < 2 s).
Response 201
```json
{ "photo": { "id": "01J…", "width": 3024, "height": 4032, "preview_url": "…", "expires_at": "…" },
  "check": { "passed": true, "faces": 1, "reasons": [] } }
```
Response 422 `PHOTO_REJECTED` with `data.reasons ∈ no_face | multiple_faces | face_too_small | blurry | too_dark | occluded | low_resolution` and the user-facing message "这张照片可能影响生成效果，请换一张清晰正脸照片。"

### GET /v1/photos?page=
### DELETE /v1/photos/{id}

## 6. Tasks

### POST /v1/tasks
Header `Idempotency-Key: <uuid>` (required).
```json
{ "kind": "idphoto", "spec_id": "…", "photo_id": "…",
  "params": { "bg": "#438EDB", "clothing": "white_shirt", "beauty": "natural" },
  "notify": true }
```
or
```json
{ "kind": "template", "template_id": "…", "photo_id": "…", "params": {}, "notify": true, "parent_task_id": null }
```
Response 201 `{ "task": Task, "credits": Credits }`
Errors: 402 `NO_CREDITS`, 422 `PHOTO_REJECTED` (photo not passed or expired), 503 `GENERATION_UNAVAILABLE`, 409 `CONFLICT`.

The server decides `uses_genmodel` from the request: an ID photo with `clothing = "keep"` and `beauty = "natural"` is free (no credit check, no 402/503) and runs only local and vision steps (D-21). Template tasks cost `template.credit_cost`.

Same key + same payload → 200 with the existing task (safe retry).

### GET /v1/tasks/{id}
```json
{ "id": "…", "status": "processing", "stage": "processing", "module": "idphoto",
  "uses_genmodel": true, "credits_consumed": 1,
  "created_at": "…", "started_at": "…", "finished_at": null,
  "work": null, "error": null, "refunded": false }
```
`error = { code: "TIMEOUT", message: "本次生成失败，生成次数已返还，请重新尝试。" }`

### GET /v1/tasks?status=&page=  (generation records, D-12)

### POST /v1/tasks/{id}/regenerate
Creates a new task with the same inputs and a new seed; costs the same as the original task (0 for a free ID photo). Response as POST /v1/tasks.

## 7. Works

### GET /v1/works?module=&page=
`WorkCard = { id, module, thumb_url, created_at, spec?: {name,width_mm,height_mm,width_px,height_px}, template?: {id,name} }`
### GET /v1/works/summary
`{ total: 28, by_module: { idphoto: 12, pro: 8, portrait: 8, avatar: 0 }, recent: [WorkCard] }`
### GET /v1/works/{id}
`Work = WorkCard + { url (signed, 10 min), width, height, meta, task_id, ai_label: true }`
### GET /v1/works/{id}/download
`{ url, expires_at }` (fresh signed URL; the client logs a `work_saved` event after saving)
### POST /v1/works/{id}/recolor   (ID photo, free, D-05)
Request `{ "bg": "#FFFFFF" }` → 201 `{ "work": Work }` (synchronous composite from `alpha_key`)
### DELETE /v1/works/{id}

## 8. Favorites

`GET /v1/favorites?page=`, `PUT /v1/favorites/{template_id}`, `DELETE /v1/favorites/{template_id}`.

## 9. Shares (SHARING.md)

### POST /v1/shares
```json
{ "type": "work", "work_id": "…", "surface": "result" }
```
`type ∈ template | work | poster | tab`; `template_id` for template shares, `work_id` for work/poster shares, `module` for tab shares.
Response 201
```json
{ "share": { "id": "01J…", "path": "/pages/template-detail/index?id=…&s=01J…",
             "title": "用「韩系清透」生成了这张，试试同款", "image_url": "https://…signed OSS URL, 24 h…/shares/01J….jpg",
             "poster_url": "https://…signed… (poster type only)" } }
```
Work and poster shares copy the work thumbnail (AI label burned in) to a public object at creation; the client must call this only after the user taps a share action.

### POST /v1/shares/{id}/open
Unauthenticated allowed (landing may precede login). Body `{ "platform": "mp-weixin", "device_id": "…" }`. Increments opens, records `share_opens`; returns `{ "share": { type, template_id, spec_id, preview_url, title } }` for the landing hero (preview only for active work shares).

### GET /v1/shares?page=  (sharer's own shares with open counts)
### GET /v1/shares/rewards  → `{ enabled, per_reward: 1, daily_cap: 3, earned_today: 0, earned_total: 4 }` for the credits page
### DELETE /v1/shares/{id}  (revoke: removes public copies)

## 10. Events (D-17)

### POST /v1/events
```json
{ "events": [ { "name": "template_click", "props": { "template_id": "…", "module": "portrait" }, "ts": "…" } ] }
```
Accepted names: `tab_view`, `banner_click`, `spec_click`, `template_click`, `template_detail_view`, `upload_start`, `upload_success`, `photo_rejected`, `confirm_view`, `generate_click`, `no_credits`, `ad_show`, `ad_ended`, `ad_abandon`, `ad_error`, `task_created`, `task_success`, `task_failed`, `result_view`, `work_saved`, `regenerate_click`, `recolor_click`, `share_click`, `share_sent`, `poster_saved`. Server-side only: `share_open`, `share_attributed`, `share_reward_granted`.

## 11. Webhooks

### GET /webhooks/wechat/ad-reward
WeChat rewarded-video server callback (enable in the traffic-master console). Verifies `sign`, matches `trans_id` to the pending ad session by `openid` + time window, sets `server_verified_at`. Returns `{ "isValid": true }` per WeChat spec. Idempotent on `trans_id`.

### POST /webhooks/wechat/media-check
`mediaCheckAsync` result callback; updates `photos.moderation_status` / `works.moderation_status` and triggers the D-15 handling.

## 12. Admin API (`/admin/v1`, admin JWT)

| Method | Path | Notes |
|---|---|---|
| POST | /auth/login | username + password → token |
| CRUD | /categories, /specs, /templates, /collections, /banners | status toggle, sort, `is_hot` (max 4 hot specs per module enforced) |
| POST | /assets | upload cover/sample images to object storage `assets/` (OSS in test/prod; served as 24 h signed URLs) |
| GET/PUT | /configs | keys from BACKEND_ARCHITECTURE.md §8 |
| GET | /users?openid= , /users/{id} | with credit balance and ledger |
| POST | /users/{id}/credits | admin adjust with note (ledger `admin_adjust`) |
| GET | /tasks?status=&module=&user_id=&from=&to= | with cost, error, provider |
| POST | /tasks/{id}/refund | manual refund when automation missed |
| GET | /stats/funnel?from=&to=&module= | PRD §2 metrics from `events` + `daily_stats` |
| GET | /stats/templates?from=&to= | clicks, tasks, success rate, cost per template, gen vs free task split |
| GET | /stats/shares?from=&to=&module= | shares by type, opens, acquired users, open → task → save conversion |
| POST | /templates/validate | checks `gen_config` against provider capabilities and the banned-word list before save |
| GET | /audit-logs | |

Every mutating admin call writes `audit_logs`.
