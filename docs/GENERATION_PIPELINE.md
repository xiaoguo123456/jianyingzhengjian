# Generation Pipeline — Engines, Credits, Ads, Tasks

This document is the executable version of PRD sections 2.3, 7–8, 22–26 and 36–37 with the decisions from DECISIONS.md applied. The image-generation large model ("gen model") draws every output, ID photos included; local code only resizes, crops, labels and exports what the model returns (D-26, which revises D-21).

## 1. Principle

> A credit is reserved in the database **before** any gen-model call, and refunded whenever the task does not deliver a usable result, whatever the reason (D-25). Every task calls the gen model once and costs its credit price (D-26).

## 2. Models and operation routing (D-26)

Two model calls and one local engine, nothing else:

| Component | What it is | When | Cost profile | Latency |
|---|---|---|---|---|
| Photo check (`provider/inspect`) | Multimodal chat model on the NewAPI gateway (`INSPECT_MODEL`) | once per upload | a few fen per call | 2–5 s |
| Gen model (`engine/genmodel`) | Image-edit model on the NewAPI gateway (`gpt-image-2.5`) via provider adapters | once per task | ¥0.1–1 per image | 20–60 s |
| `local` engine | Pure Go imaging: decode/orient, centre-crop and resize, export, AI label, thumbnail | every output | ~0 | ms |

There is no face detection, face comparison or matting service.

Operation catalogue. Every user-facing feature maps to one row; new features are added here first.

| Operation | Runs on | Credit | Used by |
|---|---|---|---|
| `inspect` | photo check | 0 | upload: face count, gender, quality issues |
| `gen.edit` (instruction) | gen model | 1 | every ID photo: background colour, clothing, retouch and ID composition in one instruction |
| `gen.edit` / `gen.reference` | gen model | `credit_cost` | professional, portrait, avatar templates; `reference` when the template has reference images |
| `fill` (centre-crop + resize) / `export` / `label` | local | 0 | every output |

Routing rules:

1. One task is one gen call. Background colour, clothing and "轻度" retouch for an ID photo are written into a single instruction built from `app_configs.idphoto_prompt`.
2. A spec task costs 1 credit; a template task costs `templates.credit_cost` (default 1).
3. Changing an ID photo's background is a new task from the original photo (`POST /v1/tasks/{id}/regenerate` with `bg`) and costs 1 credit; there is no free recolor.
4. Every task has `uses_genmodel = true`, so credit reservation and the 503 check apply to all of them.

## 3. Credit state (D-01, D-02)

```
                daily_free_remaining          bonus_credits
new day ──────▶ = daily_free_credits          unchanged
ad reward ────▶ unchanged                     +1 (≤ ad cap/day)
share reward ─▶ unchanged                     +1 (≤ share cap/day, SHARING.md §7)
consume(n) ───▶ take from daily first, then bonus (n = credit_cost, usually 1)
refund ───────▶ +n to the bucket(s) it was taken from
```

Daily reset is lazy: `EnsureDailyGrant` runs at the start of every credit transaction and compares `daily_reset_date` with today's date in Asia/Shanghai. It also resets `ad_rewards_today`. When `ads_enabled=false`, the daily amount is `daily_free_credits_no_ads`.

## 4. Ad session rules

```
POST /v1/ads/sessions          →  pending session, TTL ad_session_ttl_minutes
client plays RewardedVideoAd
onClose({isEnded})
POST /v1/ads/sessions/{id}/claim {is_ended}
```

Claim is accepted only when all of the following hold, checked in one transaction with the credit row locked:

| Check | Reject reason |
|---|---|
| session exists, belongs to caller, status = pending | `duplicate` / 404 |
| `now − created_at ≥ ad_session_min_seconds` (10 s) | `too_fast` |
| `now − created_at ≤ ttl` | `expired` |
| `is_ended = true` | `not_ended` |
| `ad_rewards_today < ad_reward_daily_cap` | `cap_reached` |

On success: ledger `ad_reward` (+1 bonus), session → `claimed`, `ad_rewards_today += 1`. When WeChat's server-side reward callback is enabled for the ad unit, the callback marks the session `server_verified_at`; a nightly job reports sessions claimed without server verification. Ad units are stored per platform so App ad units can be added later.

## 5. Task state machine

```
              create (credit consumed in the same tx when uses_genmodel)
                        │
                        ▼
                    ┌─────────┐   worker picks up    ┌────────────┐
                    │ waiting │ ───────────────────▶ │ processing │
                    └─────────┘                      └─────┬──────┘
                        │ requeue-stuck (>1 min)            │
                        └───────────────────────────────────┤
                                                            │
                     ┌──────────────────────────────────────┼───────────────────┐
                     ▼                                      ▼                   ▼
               ┌──────────┐                           ┌──────────┐        ┌──────────┐
               │ success  │                           │  failed  │        │  failed  │
               │ work_id  │                           │ refunded │        │ TIMEOUT  │
               └──────────┘                           └──────────┘        │ refunded │
                                                                          └──────────┘
```

Every task, ID photos included, reserves its credit at creation and calls the gen model, so all tasks take the 20–60 s path; there are no free or synchronous tasks (D-26).

Stages shown while `processing`: `queued` (照片处理中) → `processing` (正在生成) → `finishing` (正在优化结果). No percentage.

Transitions use `UPDATE … WHERE id=? AND status=?` so a late duplicate worker cannot regress a finished task.

## 6. Failure handling matrix

| Situation | Task status | Credit | User message |
|---|---|---|---|
| Gen provider 5xx / timeout on first call | retry once (fallback provider if configured), then `failed/PROVIDER_ERROR` | refund | 本次生成失败，生成次数已返还，请重新尝试。 |
| Provider content-policy rejection | `failed/CONTENT_REJECTED` | refund | 这张照片无法生成，请换一张照片。 |
| Output flagged by moderation (D-15), also after the task showed `success` | `failed/CONTENT_REJECTED`, work hidden | refund | same |
| Reference image missing or unreadable | `failed/STORAGE_ERROR` | refund | generic |
| Worker crash mid-task | `task:expire-processing` → `failed/TIMEOUT` after 5 min | refund | timeout message |
| Enqueue failed after commit | stays `waiting`; requeued after 1 min | none | none |
| Never started within `task_queue_timeout_seconds` (30 min; lost job, worker down) | `failed/TIMEOUT`; a late job skips it | refund | timeout message |
| Storage failure | `failed/STORAGE_ERROR` | refund | generic |
| Breaker open at creation | 503 before creation | none | 生成服务暂时不可用，请稍后再试。 |
| Client network timeout on POST /v1/tasks | retry with same `Idempotency-Key` → existing task | one consume only | — |

Refund and status change are one transaction. Every 30 minutes a consistency job refunds any `failed` task with a consume and no refund, rejects any `success` task whose work is `risky`, and logs an error for each.

## 7. Pipelines

Pipelines are short Go functions chosen by `task.kind`; their inputs come from the spec, the template `gen_config`, the task `params` and the upload check stored on the photo (`gender`). There is no face detection inside a pipeline: the photo was checked once at upload (§7.4).

### 7.1 ID photo (`kind = idphoto`)

```
prepare      local    orient by EXIF, downscale long side ≤ 2048 px
gen.edit     genmodel user photo + instruction from app_configs.idphoto_prompt, filled with
                      {bg_name}/{bg_hex} from params.bg (must be in spec.bg_allowed, else spec default),
                      {clothing} from the clothing option ("keep" → keep the original clothes),
                      {beauty} ("natural" → no retouch, "light" → light natural retouch),
                      {ratio} from the spec in mm; size requested as the closest model size to the spec aspect
fill         local    centre-crop and resize to spec px (e.g. 295×413)
export       local    PNG with spec DPI (pHYs); JPEG thumbnail
label        local    visible AI label + AIGC metadata (COMPLIANCE.md §3)
upload       —        works/{user}/{id}.png, thumb
```

Credit: 1 for every ID photo. The background colour is drawn by the model, so it is not guaranteed to match the hex value exactly, and head size and position follow the model's composition, not a measured crop rule; `specs.crop_rule` is no longer used. To change the background the user regenerates with another `bg` (1 credit).

Operators tune the ID photo instruction by editing `idphoto_prompt` through `PUT /admin/v1/configs`; no release is needed.

### 7.2 Template modules (`kind = template`, modules pro / portrait / avatar)

```
prepare      local    orient, downscale long side ≤ 2048 px
references   storage  load gen_config.reference_keys (assets/…, at most 4)
gen          genmodel one call with the user photo, the reference images and the prompt
                      ({gender} → 男性 / 女性 / empty, from the upload check);
                      mode = reference when references exist, else gen_config.mode (default edit)
post         local    ops from gen_config.post, in order (vocabulary: square_crop, resize — both centre-crop)
fill         local    centre-crop to gen_config.output; avatars always end square
export       local    JPEG q92 (PNG when the provider returns alpha and style = illustration)
label        local    metadata + visible label
upload       —
```

Credit: `templates.credit_cost` (default 1). "Regenerate" = same request, new seed. "Change template" = new task, different template, same photo. Nothing checks automatically that the output still looks like the user; the prompt asks the model to keep the person's features.

### 7.3 Template `gen_config` schema

```json
{
  "engine": "genmodel",
  "provider": "newapi",              // optional; falls back to the default provider (GEN_PROVIDER_DEFAULT)
  "model": "gpt-image-2.5",          // optional; falls back to NEWAPI_MODEL
  "mode": "edit",                    // img2img | reference | edit; forced to reference when reference_keys is set
  "prompt": "专业半身职业照，深蓝西装、白衬衫，浅灰背景，柔和棚拍光，{gender}，保持人物五官不变",
  "negative_prompt": "text, watermark, extra fingers, distorted face",
  "strength": 0.55,                  // img2img only; not sent by the newapi provider
  "reference_keys": ["assets/2026/09/01J….jpg"],  // optional, at most 4; upload with POST /admin/v1/assets
  "output": { "width": 1200, "height": 1600 },
  "post": [ { "op": "resize", "width": 1200, "height": 1600 } ],  // square_crop | resize, both centre-crop
  "style": "photo",                  // photo | illustration (output format differs)
  "fallback_provider": "",           // optional; must be a registered provider
  "extra": { "quality": "high" }     // provider pass-through; newapi accepts only quality and background
}
```

With `newapi` the model is asked for the closest supported size by aspect ratio (`1024x1024`, `1024x1536`, `1536x1024`) and the result is then cropped and resized to `output`.

A new product is a template row: a prompt, optionally a few reference images for style, pose or scene, and an output size. For example, a "职业照办公室背景" template is `mode: edit` with a background instruction, and a "复古港风写真" template adds two reference photos of the look. `POST /admin/v1/templates/validate` rejects unknown post ops, more than 4 reference images and references outside `assets/`. Only a new local post-processing step needs code.

### 7.4 Photo check at upload (`POST /v1/photos`, synchronous)

The server first checks format (JPEG, PNG, WebP), size and resolution locally; a failure there is rejected without calling any model. It then sends a ≤ 1024 px JPEG copy to the multimodal model (`INSPECT_MODEL` via `/chat/completions`), which returns `{faces, gender, issues[]}` as JSON.

| Check | Where | Reason code |
|---|---|---|
| Min side ≥ 600 px | local, `photo_min_side_px` | `low_resolution` |
| Exactly one real face | model | `no_face`, `multiple_faces` |
| Face large enough (about 1/8 of the height) | model | `face_too_small` |
| Face sharp | model | `blurry` |
| Face bright enough, no strong backlight | model | `too_dark` |
| No sunglasses, mask, hand or hair over the face | model | `occluded` |
| A real photo, not a cartoon, screenshot or photo of a screen | model | `not_photo` |

Unknown issue codes from the model are dropped. If the model call fails, the upload returns `VISION_ERROR` ("照片检测暂时不可用") and nothing is stored. The result, including `gender`, is stored in `photos.check_result` and read by the pipelines. `INSPECT_PROVIDER=mock` accepts every photo and is refused in production.

## 8. Provider routing for the gen model

```go
type GenModel interface {
    Name() string
    Capabilities() Caps            // {Img2Img, Reference, Edit, MaskEdit, MaxSide, ReturnsAlpha}
    Run(ctx context.Context, req GenRequest) (GenResult, error)
}
```

- The router picks `gen_config.provider`, verifies `Capabilities()` covers `mode`, else uses `fallback_provider`, else the default provider (`GEN_PROVIDER_DEFAULT`; `app_configs.default_provider` only when the variable is empty). A provider exists in the router only when its API key is configured. A template whose mode no provider supports fails validation in the admin API.
- Each provider has a circuit breaker (5 failures / 30 s) and a concurrency semaphore; when the chosen provider's breaker is open and a fallback exists, the task uses the fallback; when none is available `POST /v1/tasks` returns 503 before any credit is taken.
- Implemented providers: `newapi` — an OpenAI-compatible gateway calling `/images/edits` with `gpt-image-2.5`; the default in test and production and verified against the live service (BACKEND_ARCHITECTURE.md §7). `volcengine` (Seedream) — adapter kept but never called against the live API. `mock` — stamps the input; dev and CI only, refused in production. Before launch, confirm that the model behind the gateway meets the filing requirement in COMPLIANCE.md §3.3; Alibaba Wanx, Tencent Hunyuan Image and Kling Image remain candidates for a filed fallback.
- `newapi` never degrades to text-to-image and never re-sends a request whose outcome is unknown (timeout, 429, 5xx); the task fails and the credit is refunded instead.
- Cost per call comes from the provider response when available, otherwise from `app_configs.provider_prices` keyed by `provider/model` (e.g. `newapi/gpt-image-2.5`). NewAPI reports no cost, so this entry must be set for cost reporting to be meaningful.

## 9. Test matrix for the credit + task core

| # | Scenario | Expected |
|---|---|---|
| 1 | New user, ads enabled, first gen task | daily grant applied, consume from daily, task waiting |
| 2 | User with 0 daily / 0 bonus, gen task | 402 NO_CREDITS, no task row |
| 3 | ID photo with original clothing and no retouch | 1 credit consumed, uses_genmodel = true (D-26); with 0 credits → 402 NO_CREDITS |
| 4 | Claim valid ad session | +1 bonus, session claimed |
| 5 | Claim same session twice | second → 422 duplicate |
| 6 | Claim after 3 s | 422 too_fast |
| 7 | Claim with is_ended=false | 422 not_ended |
| 8 | Cap reached | 422 cap_reached |
| 9 | Create task twice with same idempotency key | one task, one consume |
| 10 | Same key, different payload | 409 |
| 11 | Gen task fails | refund row, same bucket restored |
| 12 | Task fails twice (double handler) | one refund row |
| 13 | Processing past deadline | TIMEOUT + refund |
| 14 | Day rollover mid-session | daily reset once, ad counter reset |
| 15 | ads_enabled=false | daily grant = no_ads value; ad session create → 409 |
| 16 | Breaker open, template task | 503, no consume |
| 17 | Breaker open, ID photo task | 503, no consume |
| 18 | Template credit_cost = 2 with 1 daily + 1 bonus | consume takes 1 from each; refund restores both |
| 19 | Change background (regenerate with `bg`) | new task with the new colour, same photo, spec and clothing; 1 credit |
| 20 | Upload check finds no face / two faces / not a photo | 422 PHOTO_REJECTED with the reason, nothing stored, no credit |
| 21 | Acquired user's first gen task succeeds | sharer +1 bonus once; second success → no new row |
| 22 | Acquired user's first success is an ID photo | reward granted (every task uses the gen model) |
| 23 | Sharer and acquired user share a unionid or device | no reward, logged |
| 24 | Sharer at daily share cap | no reward, logged |
| 25 | Output flagged `risky` after success, callback delivered twice | task `failed/CONTENT_REJECTED`, one refund, recorded cost kept |
| 26 | Rejection missed, work already `risky` | consistency job fails and refunds the task |
| 27 | Task still `waiting` after `task_queue_timeout_seconds` | `failed/TIMEOUT` + refund; a later start is a no-op |

## 10. Cost accounting

`tasks.cost_cents` records the gen call of the task (from `provider_prices`, since NewAPI reports no cost). Upload checks are not tied to a task and are not recorded; budget them per upload. `stats:daily-rollup` aggregates cost per template, per spec and per module; the admin template list shows cost per success and failure rate so expensive or failing templates can be taken offline quickly (PRD 36).
