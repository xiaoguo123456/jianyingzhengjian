# Decisions and Assumptions

The PRD leaves several points undefined or contradictory. The architecture cannot be built without settling them, so each is decided here. Product can overturn any decision; when that happens, update this file and the documents that reference the ID.

Status values: **Decided** (used by the architecture), **Proposed** (recommended, awaiting product confirmation).

---

## D-01 · Credit model — Decided

**Problem.** PRD 1 says rewarded video is the primary way to use the product; PRD 28 and the Mine mockup show "今日可免费生成 3 次" plus "看视频领次数". Whether credits are granted daily, whether ad credits expire and whether there is a cap are all undefined.

**Decision.**

- Two balances per user: `daily_free_remaining` and `bonus_credits`.
- `daily_free_remaining` resets to `config.daily_free_credits` (default **1**) at the first credit operation of each day, Asia/Shanghai. Unused daily credits do not carry over.
- Each fully watched rewarded video grants **+1** `bonus_credits`. Bonus credits never expire.
- Ad rewards are capped at `config.ad_reward_daily_cap` (default **10**) per user per day.
- Consumption order: daily free first, then bonus.
- Every change is a ledger row; refunds return to the balance they were taken from.

**Why.** A small daily grant lets a new user try the product before an ad is available (see D-02) and matches the mockup. Non-expiring bonus credits avoid angry users who watched ads and lost credits. All numbers are admin-configurable.

## D-02 · Behaviour when ads are unavailable — Decided

**Problem.** WeChat's ad platform (流量主) cannot be enabled until the mini program reaches its visitor threshold, so a fresh launch has no ads. Ad loading can also fail on device.

**Decision.** `config.ads_enabled` (bool). When false, the UI hides every "watch video" button and the daily grant is raised to `config.daily_free_credits_no_ads` (default **3**). When ads are enabled but a load fails on device, the client shows "视频暂时无法加载，请稍后再试" and nothing else; no compensation credit.

## D-03 · Entry order: template-first and upload-first both exist — Decided

**Problem.** PRD flows are "select spec/template → upload". Mockups put "upload photo" buttons and tappable banners on every tab.

**Decision.** Both entries are allowed and converge on the same flow:

- Template-first: template detail → upload → (ID photo params) → confirm.
- Upload-first: tab primary button / banner → upload (with `module` param) → picker step (specs for ID photo, templates for the other modules) → (ID photo params) → confirm.

The picker step is a bottom-sheet list, not a new page tree. The upload page must be designed to accept an optional pre-selected target.

## D-04 · ID photo has specs, not templates — Decided

ID photo tasks reference a **spec** (`spec_id`). Professional, portrait and avatar tasks reference a **template** (`template_id`). The "热门模板" block on the ID photo tab is replaced by "热门规格" (hot specs from the spec library). Favourites apply to templates only, as the PRD states.

## D-05 · Result-page actions and credit cost — Decided

| Action | Cost | Implementation |
|---|---|---|
| Save image | 0 | Signed download URL |
| Regenerate (再生成一张) | same as the original task: 0 for an ID photo without gen steps, otherwise `credit_cost` | New task, same inputs, new seed |
| Change template (换一个模板) | 0 to browse; 1 when a new task is confirmed | Navigates to template list with photo pre-selected |
| ID photo: change background | 0 | Re-composite from the stored alpha layer; creates a new work |
| ID photo: change clothing | 1 credit | New task (clothing is a generative step) |

## D-06 · Task timeout and retries — Decided

A task that is not `success` or `failed` within **5 minutes** of `started_at` is marked `failed` with `error_code = TIMEOUT` and the credit is refunded. Transient provider errors (HTTP 5xx, rate limit, network) are retried at most **once** inside the same task. Content-policy rejections and invalid input are not retried.

## D-07 · Completion notification — Decided

Before the task is created, the client asks for the WeChat subscribe-message template "生成完成通知". If the user accepts, the worker sends one message on success or failure. The generating page also polls the task every 2 s (backing off to 5 s after 30 s). No WebSocket in V1.

## D-08 · Gender for clothing templates — Decided

Clothing options are shown for both genders in one list, grouped as "男士 / 女士"; the group matching the face-detection gender attribute (if the provider returns one) is listed first. No gender is stored on the user profile in V1.

## D-09 · Couple avatar — Deferred to V1.1

Requires two faces or two uploads, which conflicts with the single-face photo check. Removed from V1.0 catalogue and mockups.

## D-10 · Custom spec size — Deferred to V1.1

Conflicts with "少设置、少填写". The spec library ships with curated categories only.

## D-11 · Profile edit — Decided (kept)

WeChat no longer returns nickname and avatar via `getUserProfile`; they must be collected with `open-type="chooseAvatar"` and `type="nickname"` input. A minimal profile-edit page is therefore required and is the only way to set them.

## D-12 · Works vs records vs photo management — Decided

- **我的作品 (works)**: successful outputs only, grouped by module.
- **生成记录 (records)**: every task, all statuses, newest first; failed rows show the reason and the refund.
- **照片管理 (photos)**: uploaded originals only. Deleting an original does **not** delete works generated from it; the work keeps its own copy.

## D-13 · Sharing — Decided

PRD 13 positions portraits as shareable but defines no share action. V1.0 ships three share types: template card, work card (only when the user explicitly chooses "分享作品"), and a server-rendered poster with a mini program code. Every share carries a share id for attribution. Works stay private; only an explicitly shared copy with the AI label becomes public. Full design in SHARING.md. The share incentive is a separate decision (D-22).

## D-14 · Watermarks and paid tiers — Decided

No visible product watermark on free outputs in V1.0. The AI-content label required by regulation is separate (see COMPLIANCE.md). The `works.meta` JSON and `tasks.cost_cents` fields reserve room for a paid tier later; nothing else is built.

## D-15 · Content moderation — Decided

Every uploaded original and every generated output is submitted to WeChat `security.mediaCheckAsync`. Originals are rejected on `risky`; outputs flagged `risky` are hidden from the user and the task is marked `failed` with `error_code = CONTENT_REJECTED` and refunded.

## D-16 · Photo retention — Decided

Uploaded originals are deleted **30 days** after upload unless the user deletes them earlier. Works are kept until the user deletes them. Both values are config keys. The privacy policy states these periods.

## D-17 · Analytics — Decided

Funnel events (PRD 2) are recorded twice: `wx.reportEvent` for WeChat's built-in analytics and a batched `POST /v1/events` for the admin dashboard. Event names are listed in API.md.

## D-18 · Beauty options — Decided (confirmed by product 2026-09-11)

"自然" = no retouch and no gen-model call. "轻度" = one low-strength gen-model retouch instruction, merged into the task's single gen call. Choosing "轻度" therefore turns an otherwise free ID photo into a 1-credit task; the UI states this next to the option. No slider, no manual parameters.

## D-19 · Frontend framework — Decided

uni-app (Vue 3 + TypeScript, Vite CLI project), building `mp-weixin` first. A native App (`app-plus`) is a likely future target, and uni-app keeps that reachable from one codebase. All WeChat-specific calls are confined to a platform layer so the App build does not touch business code. Rationale and trade-offs in FRONTEND_ARCHITECTURE.md.

## D-20 · Backend stack — Decided

Go 1.23, Gin, GORM on MySQL 8, Redis 7 with Asynq for background jobs, Tencent COS for object storage. Rationale in BACKEND_ARCHITECTURE.md.

## D-21 · Image engines: gen model for creation, cheap engines for processing — Decided

**Problem.** The product will rely on an image-generation large model for most image types going forward, but many operations (solid background change, crop to spec, export) do not need it, and the model is the dominant cost and latency.

**Decision.** Three engines behind one interface: `local` (pure Go imaging), `vision` (per-call face detection, face compare, portrait matting), `genmodel` (large image model via pluggable providers). Every operation is routed to exactly one engine in the operation catalogue (GENERATION_PIPELINE.md §2). Rules:

- Solid-colour background changes never call the gen model: matte once, composite locally, unlimited free recolors.
- Anything that creates or alters content (styles, clothing, hairstyle, scene backgrounds, retouch) goes through the gen model and costs credits; several gen operations in one task are merged into one call and one credit.
- A task records `uses_genmodel`; credits and the provider breaker apply only to those tasks. An ID photo with original clothing and no retouch is free and typically finishes in seconds.
- Templates carry a `gen_config` recipe (mode, prompt, strength, post-processing ops) so new image products are added as data, not code, as long as they compose existing steps.
- Providers are chosen per template with capability checks and a fallback provider.

## D-22 · Share incentive — Decided (enabled at launch)

Confirmed by product on 2026-09-11. `share_reward_enabled=true` from V1.0. A sharer earns +1 bonus credit when a user acquired through their share completes a first successful gen task; one reward per acquired user, `share_reward_daily_cap` (default 3) per sharer, no self-referral (same unionid or device). Rewards for opens or installs are never granted. Because the switch is on from day one, the anti-abuse checks and the reward copy in the share sheet and credits page are V1.0 scope, not follow-ups. Details in SHARING.md §7.

## D-23 · Identity providers — Decided

Users are keyed by an internal id; external identities (`wechat_mp` openid/unionid now, `wechat_app` or phone later) live in `user_identities`. The login endpoint takes a `provider` field. This is a small cost now and avoids a migration when the App ships.

## D-24 · Four tabs: 头像 merged into 写真 — Decided

Confirmed by product on 2026-09-15. PRD V1.0 defines five tabs. The first build showed that five tabs plus content felt crowded, and 头像 and 写真 share the same flow (template → upload → gen model). The TabBar is now 证件照 / 职业照 / 写真 / 我的.

- The 写真 tab shows two segment titles at the top, 写真 and 头像, each with its own hero, categories and rails.
- `avatar` stays a separate module everywhere else: templates, tasks, works, the home API, analytics and the Mine page's four work categories. No backend or data change beyond link paths.
- Links into the avatar segment use `/pages/portrait/index?seg=avatar`. `uni.switchTab` cannot carry a query, so in-app navigation sets the segment through the client `ui` store.
