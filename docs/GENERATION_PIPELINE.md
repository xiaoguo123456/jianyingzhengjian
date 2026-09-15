# Generation Pipeline — Engines, Credits, Ads, Tasks

This document is the executable version of PRD sections 2.3, 7–8, 22–26 and 36–37 with the decisions from DECISIONS.md applied. The image-generation large model ("gen model") is the primary engine for anything that **creates or changes** image content; cheap deterministic engines handle everything that only **processes** an image (D-21).

## 1. Principle

> A credit is reserved in the database **before** any gen-model call, and refunded when the task fails for any reason that is not the user's fault. Operations that do not need the gen model cost no credit and never call it.

## 2. Engines and operation routing (D-21)

Three engines sit behind one `engine` package in the worker:

| Engine | What it is | Cost profile | Latency |
|---|---|---|---|
| `local` | Pure Go imaging: orient, resize, crop, composite, export, label, thumbnail | ~0 | ms |
| `vision` | Narrow per-call cloud APIs: face detect + attributes, face compare, portrait matting | ¥0.001–0.01 per call | < 1 s |
| `genmodel` | Large image model via provider adapters (img2img, reference-guided, mask edit) | ¥0.1–1 per image | 5–40 s |

Operation catalogue. Every user-facing feature maps to one row; new features are added here first.

| Operation | Engine | Credit | Used by |
|---|---|---|---|
| `detect_face` | vision | 0 | upload check, every pipeline |
| `compare_face` | vision | 0 | identity check after generation |
| `matte` | vision | 0 | ID photo, avatar background swap to solid colour |
| `crop_spec` | local | 0 | ID photo |
| `composite_solid_bg` | local | 0 | ID photo, free recolor |
| `square_crop` / `resize` / `export` / `label` | local | 0 | every output |
| `gen.img2img` | genmodel | 1 | professional, portrait, avatar styles |
| `gen.reference` | genmodel | 1 | same, when the provider supports subject/face reference for better identity |
| `gen.edit` (mask or instruction) | genmodel | 1 | ID clothing swap, avatar hairstyle, scene background for professional photos |
| `gen.retouch` (low strength) | genmodel | included in the task | "轻度" beauty; never a separate credit |
| `gen.upscale` | genmodel | 0 in V1 (not offered) | reserved |

Routing rules:

1. Solid-colour background changes are **never** sent to the gen model; matte once, composite as many times as needed.
2. Multiple gen operations in one task (e.g. clothing + light retouch) are merged into **one** gen call with a combined instruction when the provider supports it; otherwise executed in sequence but still billed as one credit.
3. `templates.credit_cost` (default 1) is the credit price of a template task; a spec task costs 1 only if it includes a gen operation (clothing ≠ keep or beauty = light), otherwise 0.
4. A task declares up front whether it will touch the gen model (`tasks.uses_genmodel`), so credit reservation and the 503 breaker check happen only for those tasks.

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

Free tasks (no gen model: ID photo with original clothing, recolor) run the same state machine but skip the ledger; ID photo without gen typically completes in under 3 s, so the client still polls but usually sees `success` on the first poll. Recolor is executed synchronously in the API (D-05).

Stages shown while `processing`: `queued` (照片处理中) → `processing` (正在生成) → `finishing` (正在优化结果). No percentage.

Transitions use `UPDATE … WHERE id=? AND status=?` so a late duplicate worker cannot regress a finished task.

## 6. Failure handling matrix

| Situation | Task status | Credit | User message |
|---|---|---|---|
| Gen provider 5xx / timeout on first call | retry once (fallback provider if configured), then `failed/PROVIDER_ERROR` | refund | 本次生成失败，生成次数已返还，请重新尝试。 |
| Provider content-policy rejection | `failed/CONTENT_REJECTED` | refund | 这张照片无法生成，请换一张照片。 |
| Output flagged by moderation (D-15) | `failed/CONTENT_REJECTED` | refund | same |
| Identity check below threshold after one regeneration | `failed/IDENTITY_MISMATCH` | refund | 生成结果与本人差异较大，请换一张正脸照片。 |
| No face at pipeline time | `failed/NO_FACE` | refund | 未检测到清晰人脸，请换一张照片。 |
| Vision API failure (matte / detect) | retry twice, then `failed/VISION_ERROR` | refund if consumed | generic |
| Worker crash mid-task | `task:expire-processing` → `failed/TIMEOUT` after 5 min | refund | timeout message |
| Enqueue failed after commit | stays `waiting`; requeued after 1 min | none | none |
| Storage failure | `failed/STORAGE_ERROR` | refund | generic |
| Breaker open at creation (gen tasks only) | 503 before creation | none | 生成服务暂时不可用，请稍后再试。 |
| Client network timeout on POST /v1/tasks | retry with same `Idempotency-Key` → existing task | one consume only | — |

Refund and status change are one transaction. A nightly consistency job refunds any `failed` task with a consume and no refund and raises an alert.

## 7. Pipelines

Pipelines are compositions of steps from the operation catalogue. Steps are reusable Go functions; a pipeline is chosen by `task.kind`, and its parameters come from the spec, the template `gen_config`, and the task `params`.

### 7.1 ID photo (`kind = idphoto`)

```
prepare      local    orient by EXIF, downscale long side ≤ 3000 px
detect       vision   exactly 1 face; bbox, landmarks, quality, attrs
[gen.edit]   genmodel ONLY IF clothing ≠ keep OR beauty = light
                      instruction from clothing option prompt (+ "light natural skin retouch" when beauty=light)
                      mask: below-chin region from matte, or instruction-only edit if provider prefers
                      then: detect again + compare_face against original (threshold cfg, default 0.75)
crop_spec    local    head_ratio / top_margin rule (below), pad edges if the crop leaves the frame
matte        vision   alpha mask of the cropped image; stored as works.alpha_key
composite    local    solid bg from params.bg (must be in spec.bg_allowed); 2 px hair feathering
export       local    PNG at spec px, sRGB, DPI metadata; 3:4 JPEG thumb
label        local    metadata label always; visible label only when a gen step ran (COMPLIANCE.md §3)
upload       —        works/{user}/{id}.png, thumb, alpha
```

Crop rule (defaults; `specs.crop_rule` overrides):

```
est_head_h = 1.45 × face_bbox_h            bbox is chin-to-brow; add crown/hair
scale      = (spec_h × head_ratio) / est_head_h      head_ratio 0.62
head_top_y = face_bbox_top − 0.30 × face_bbox_h
crop_top   = head_top_y − top_margin × (spec_h / scale)  top_margin 0.10
crop_cx    = face_bbox_center_x
crop_w, crop_h = spec_w / scale, spec_h / scale
```

Credit: 0 when clothing = keep and beauty = natural; 1 otherwise. Free recolor (`POST /v1/works/{id}/recolor`) re-runs composite → export → label → upload from the stored alpha.

### 7.2 Template modules (`kind = template`, modules pro / portrait / avatar)

```
prepare      local    orient, downscale long side ≤ 2048 px
detect       vision   exactly 1 face
gen          genmodel mode from gen_config.mode:
                      img2img   — source photo + prompt + strength
                      reference — source photo as subject/face reference + prompt (best identity)
                      edit      — instruction / mask edit on the source (hairstyle, scene background)
identity     vision   compare_face(source, output) when gen_config.identity_check = true
                      below threshold → one regeneration with strength − 0.1, then IDENTITY_MISMATCH
post         local    ops from gen_config.post, in order (vocabulary: square_crop, resize, matte_solid_bg)
export       local    JPEG q92 (PNG when the provider returns alpha or style = illustration)
label        local    metadata + visible label
upload       —
```

Credit: `templates.credit_cost` (default 1). "Regenerate" = same request, new seed. "Change template" = new task, different template, same photo.

### 7.3 Template `gen_config` schema

```json
{
  "engine": "genmodel",
  "provider": "seedream",            // optional; falls back to config.default_provider
  "model": "seedream-4.0",
  "mode": "reference",               // img2img | reference | edit
  "prompt": "professional corporate headshot, navy suit, soft studio light, neutral grey backdrop, {gender}",
  "negative_prompt": "text, watermark, extra fingers, distorted face",
  "strength": 0.55,                  // img2img only
  "output": { "width": 1024, "height": 1365 },
  "identity_check": true,
  "identity_threshold": 0.75,
  "post": [ { "op": "resize", "width": 1200, "height": 1600 } ],
  "style": "photo",                  // photo | illustration (label + identity rules differ)
  "fallback_provider": "wanx"
}
```

Examples of new products that need **no code change**: a "证件照换发型" template = `mode: edit`, prompt for hairstyle, `post: [crop_spec…]` is not allowed (spec pipelines are code), but an "avatar hairstyle" template is just a row. A "职业照办公室背景" template = `mode: edit` with a background instruction. Anything requiring a new local step is a code change to the step library, not to the pipelines.

### 7.4 Photo check at upload (`POST /v1/photos`, synchronous)

| Check | Threshold (config) | Reason code |
|---|---|---|
| Face count = 1 | — | `no_face`, `multiple_faces` |
| Face height / image height ≥ 0.08 | `face_min_ratio` | `face_too_small` |
| Min side ≥ 600 px | `photo_min_side_px` | `low_resolution` |
| Blur score (Laplacian variance) ≥ threshold | `blur_min_var` | `blurry` |
| Mean luminance in face region ≥ 60/255 | `dark_min_luma` | `too_dark` |
| Provider occlusion / mask / sunglasses flags | — | `occluded` |

The result is stored on the photo and reused by the pipelines; pipelines re-detect only after a gen step.

## 8. Provider routing for the gen model

```go
type GenModel interface {
    Name() string
    Capabilities() Caps            // {Img2Img, Reference, Edit, MaskEdit, MaxSide, ReturnsAlpha}
    Run(ctx context.Context, req GenRequest) (GenResult, error)
}
```

- The router picks `gen_config.provider`, verifies `Capabilities()` covers `mode`, else uses `fallback_provider`, else `config.default_provider`. A template whose mode no provider supports fails validation in the admin API.
- Each provider has a circuit breaker (5 failures / 30 s) and a concurrency semaphore; when the chosen provider's breaker is open and a fallback exists, the task uses the fallback; when none is available `POST /v1/tasks` returns 503 for gen tasks.
- Candidate providers (choose filed models for China: COMPLIANCE.md §3.3): Volcengine Seedream (edit + multi-reference), Alibaba Wanx (image edit), Tencent Hunyuan Image, Kling Image. The `mock` provider stamps the input and is used in dev/CI.
- Cost per call comes from the provider response when available, otherwise from `config.provider_prices` keyed by provider/model.

## 9. Test matrix for the credit + task core

| # | Scenario | Expected |
|---|---|---|
| 1 | New user, ads enabled, first gen task | daily grant applied, consume from daily, task waiting |
| 2 | User with 0 daily / 0 bonus, gen task | 402 NO_CREDITS, no task row |
| 3 | User with 0 credits, ID photo with original clothing | task created, no ledger row, uses_genmodel = false |
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
| 16 | Breaker open, gen task | 503, no consume |
| 17 | Breaker open, free ID photo task | task created and runs |
| 18 | Template credit_cost = 2 with 1 daily + 1 bonus | consume takes 1 from each; refund restores both |
| 19 | Free recolor | no ledger row, new work with same alpha |
| 20 | Identity check fails twice | IDENTITY_MISMATCH, one refund, two provider calls recorded in cost |
| 21 | Acquired user's first gen task succeeds | sharer +1 bonus once; second success → no new row |
| 22 | Acquired user's first success is a free ID photo | no reward; a later gen success rewards |
| 23 | Sharer and acquired user share a unionid or device | no reward, logged |
| 24 | Sharer at daily share cap | no reward, logged |

## 10. Cost accounting

`tasks.cost_cents` sums every gen and vision call in the task. `stats:daily-rollup` aggregates cost per template, per spec and per module; the admin template list shows cost per success and failure rate so expensive or failing templates can be taken offline quickly (PRD 36). Vision-only tasks show near-zero cost, which is the point of the routing rules.
